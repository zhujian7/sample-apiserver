package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/uuid"
	version "k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/klog/v2"
)

const (
	groupName  = "things.myorg.io"
	apiVersion = "v1alpha1"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WidgetSpec   `json:"spec,omitempty"`
	Status            WidgetStatus `json:"status,omitempty"`
}

type WidgetSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int32  `json:"size"`
}

type WidgetStatus struct {
	Phase string `json:"phase,omitempty"`
}

type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Widget `json:"items"`
}

func (w *Widget) DeepCopyObject() runtime.Object {
	return &Widget{
		TypeMeta:   w.TypeMeta,
		ObjectMeta: *w.ObjectMeta.DeepCopy(),
		Spec:       w.Spec,
		Status:     w.Status,
	}
}

func (wl *WidgetList) DeepCopyObject() runtime.Object {
	out := &WidgetList{
		TypeMeta: wl.TypeMeta,
		ListMeta: wl.ListMeta,
		Items:    make([]Widget, len(wl.Items)),
	}
	for i := range wl.Items {
		out.Items[i] = *wl.Items[i].DeepCopyObject().(*Widget)
	}
	return out
}

func init() {
	gv := schema.GroupVersion{Group: groupName, Version: apiVersion}
	Scheme.AddKnownTypes(gv, &Widget{}, &WidgetList{})
	metav1.AddToGroupVersion(Scheme, gv)

	// Register meta types
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}

type MemoryStorage struct {
	mu      sync.RWMutex
	widgets map[string]*Widget
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		widgets: make(map[string]*Widget),
	}
}

func (s *MemoryStorage) Get(name string) (*Widget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	widget, exists := s.widgets[name]
	if !exists {
		return nil, fmt.Errorf("widget %s not found", name)
	}
	return widget.DeepCopyObject().(*Widget), nil
}

func (s *MemoryStorage) List() (*WidgetList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &WidgetList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: groupName + "/" + apiVersion,
			Kind:       "WidgetList",
		},
		Items: make([]Widget, 0, len(s.widgets)),
	}

	for _, widget := range s.widgets {
		list.Items = append(list.Items, *widget.DeepCopyObject().(*Widget))
	}

	return list, nil
}

func (s *MemoryStorage) Create(widget *Widget) (*Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if widget.Name == "" {
		widget.Name = string(uuid.NewUUID())
	}

	if _, exists := s.widgets[widget.Name]; exists {
		return nil, fmt.Errorf("widget %s already exists", widget.Name)
	}

	now := metav1.NewTime(time.Now())
	widget.CreationTimestamp = now
	widget.ResourceVersion = "1"
	widget.UID = uuid.NewUUID()
	widget.Status.Phase = "Active"

	s.widgets[widget.Name] = widget.DeepCopyObject().(*Widget)
	return widget, nil
}

func (s *MemoryStorage) Update(widget *Widget) (*Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.widgets[widget.Name]
	if !exists {
		return nil, fmt.Errorf("widget %s not found", widget.Name)
	}

	widget.CreationTimestamp = existing.CreationTimestamp
	widget.UID = existing.UID
	widget.ResourceVersion = fmt.Sprintf("%d", time.Now().UnixNano())

	s.widgets[widget.Name] = widget.DeepCopyObject().(*Widget)
	return widget, nil
}

func (s *MemoryStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.widgets[name]; !exists {
		return fmt.Errorf("widget %s not found", name)
	}

	delete(s.widgets, name)
	return nil
}

type WidgetREST struct {
	storage *MemoryStorage
}

func NewWidgetREST() *WidgetREST {
	return &WidgetREST{
		storage: NewMemoryStorage(),
	}
}

func (r *WidgetREST) New() runtime.Object {
	return &Widget{}
}

func (r *WidgetREST) NewList() runtime.Object {
	return &WidgetList{}
}

func (r *WidgetREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.storage.Get(name)
}

func (r *WidgetREST) List(ctx context.Context, options *metav1.ListOptions) (runtime.Object, error) {
	return r.storage.List()
}

func (r *WidgetREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	widget := obj.(*Widget)
	widget.TypeMeta = metav1.TypeMeta{
		APIVersion: groupName + "/" + apiVersion,
		Kind:       "Widget",
	}
	return r.storage.Create(widget)
}

func (r *WidgetREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	oldObj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	updatedObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	widget := updatedObj.(*Widget)
	widget.Name = name
	updatedWidget, err := r.storage.Update(widget)
	return updatedWidget, false, err
}

func (r *WidgetREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(name)
	return obj, true, err
}

func (r *WidgetREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *WidgetREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: groupName, Resource: "widgets"}).ConvertToTable(ctx, object, tableOptions)
}

func (r *WidgetREST) NamespaceScoped() bool {
	return true
}

func (r *WidgetREST) GetSingularName() string {
	return "widget"
}

func installAPI(s *genericapiserver.GenericAPIServer) error {
	widgetREST := NewWidgetREST()

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(groupName, Scheme, metav1.ParameterCodec, Codecs)
	apiGroupInfo.VersionedResourcesStorageMap[apiVersion] = map[string]rest.Storage{
		"widgets": widgetREST,
	}

	return s.InstallAPIGroup(&apiGroupInfo)
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
}

type WidgetAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

func (s *WidgetAPIServer) Run(stopCh <-chan struct{}) error {
	return s.GenericAPIServer.PrepareRun().Run(stopCh)
}

func NewConfig() *Config {
	return &Config{
		GenericConfig: genericapiserver.NewRecommendedConfig(Codecs),
	}
}

func (c *Config) Complete() *Config {
	c.GenericConfig.Complete()
	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return c
}

func (c *Config) New() (*WidgetAPIServer, error) {
	genericServer, err := c.GenericConfig.Complete().New("widget-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &WidgetAPIServer{
		GenericAPIServer: genericServer,
	}

	if err := installAPI(s.GenericAPIServer); err != nil {
		return nil, err
	}

	return s, nil
}

func main() {
	klog.InitFlags(nil)

	stopCh := genericapiserver.SetupSignalHandler()

	options := genericoptions.NewRecommendedOptions("", Codecs.LegacyCodec())

	// // Set dummy etcd servers to pass validation, we'll override this later
	// options.Etcd.StorageConfig.Transport.ServerList = []string{"http://localhost:2379"}

	// Now disable etcd for in-memory storage after validation passes
	options.Etcd = nil

	// // Set default authentication options to avoid validation errors
	// options.Authentication.RemoteKubeConfigFileOptional = true
	// options.Authorization.RemoteKubeConfigFileOptional = true

	options.AddFlags(pflag.CommandLine)

	pflag.Parse()

	if errs := options.Validate(); len(errs) != 0 {
		klog.Errorf("Error validating options: %v", errs)
	}

	config := NewConfig()
	if err := options.ApplyTo(config.GenericConfig); err != nil {
		klog.Fatalf("Error applying options: %v", err)
	}

	config = config.Complete()

	server, err := config.New()
	if err != nil {
		klog.Fatalf("Error creating server: %v", err)
	}

	server.GenericAPIServer.Handler.NonGoRestfulMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			responsewriters.WriteRawJSON(http.StatusOK, map[string]interface{}{"paths": []string{"/api", "/apis"}}, w)
		}
	})

	klog.Infof("Starting widget-apiserver...")
	if err := server.Run(stopCh); err != nil {
		klog.Fatalf("Error running server: %v", err)
	}
}
