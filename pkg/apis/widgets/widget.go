package widgets

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"example.com/mytest-apiserver/pkg/common"
)

// Widget represents a sample widget resource
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of Widget
	Spec WidgetSpec `json:"spec,omitempty"`

	// Status defines the observed state of Widget
	Status WidgetStatus `json:"status,omitempty"`
}

// WidgetSpec defines the desired state of Widget
type WidgetSpec struct {
	// Name is the name of the widget
	Name string `json:"name"`

	// Description describes what the widget does
	Description string `json:"description"`

	// Size indicates the size of the widget
	Size int32 `json:"size"`
}

// WidgetStatus defines the observed state of Widget
type WidgetStatus struct {
	// Phase indicates the current phase of the widget
	Phase string `json:"phase,omitempty"`
}

// WidgetList contains a list of Widget
type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of Widget objects
	Items []Widget `json:"items"`
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

type MemoryStorage struct {
	mu             sync.RWMutex
	widgets        map[string]*Widget
	versionCounter int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		widgets:        make(map[string]*Widget),
		versionCounter: 1,
	}
}

func (s *MemoryStorage) getKey(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

func (s *MemoryStorage) parseKey(key string) (namespace, name string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

func (s *MemoryStorage) Get(namespace, name string) (*Widget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.getKey(namespace, name)
	widget, exists := s.widgets[key]
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{Group: common.GroupName, Resource: "widgets"}, name)
	}
	return widget.DeepCopyObject().(*Widget), nil
}

func (s *MemoryStorage) List(ns string) (*WidgetList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &WidgetList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: common.GroupName + "/" + common.APIVersion,
			Kind:       "WidgetList",
		},
		Items: make([]Widget, 0, len(s.widgets)),
	}

	for _, widget := range s.widgets {
		if len(ns) == 0 {
			list.Items = append(list.Items, *widget.DeepCopyObject().(*Widget))
		} else if widget.Namespace == ns {
			list.Items = append(list.Items, *widget.DeepCopyObject().(*Widget))
		}

	}

	return list, nil
}

func (s *MemoryStorage) Create(widget *Widget) (*Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if widget.Name == "" {
		widget.Name = string(uuid.NewUUID())
	}

	key := s.getKey(widget.Namespace, widget.Name)
	if _, exists := s.widgets[key]; exists {
		return nil, fmt.Errorf("widget %s already exists in namespace %s", widget.Name, widget.Namespace)
	}

	now := metav1.NewTime(time.Now())
	widget.CreationTimestamp = now
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	widget.UID = uuid.NewUUID()
	widget.Status.Phase = "Active"

	s.widgets[key] = widget.DeepCopyObject().(*Widget)
	return widget, nil
}

func (s *MemoryStorage) Update(widget *Widget) (*Widget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(widget.Namespace, widget.Name)
	existing, exists := s.widgets[key]
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{Group: common.GroupName, Resource: "widgets"}, widget.Name)
	}

	widget.CreationTimestamp = existing.CreationTimestamp
	widget.UID = existing.UID
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	s.widgets[key] = widget.DeepCopyObject().(*Widget)
	return widget, nil
}

func (s *MemoryStorage) Delete(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.getKey(namespace, name)
	if _, exists := s.widgets[key]; !exists {
		return errors.NewNotFound(schema.GroupResource{Group: common.GroupName, Resource: "widgets"}, name)
	}

	delete(s.widgets, key)
	return nil
}

type WidgetREST struct {
	storage      *MemoryStorage
	namespace    string
	resourceName string
}

// Ensure WidgetREST implements the required interfaces
var _ rest.Creater = &WidgetREST{}
var _ rest.Lister = &WidgetREST{}
var _ rest.Getter = &WidgetREST{}
var _ rest.Updater = &WidgetREST{}
var _ rest.GracefulDeleter = &WidgetREST{}
var _ rest.Scoper = &WidgetREST{}
var _ rest.Storage = &WidgetREST{}
var _ rest.GroupVersionKindProvider = &WidgetREST{}

func NewWidgetREST() *WidgetREST {
	return &WidgetREST{
		storage:      NewMemoryStorage(),
		resourceName: "widgets",
	}
}

func (r *WidgetREST) New() runtime.Object {
	return &Widget{}
}

func (r *WidgetREST) NewList() runtime.Object {
	return &WidgetList{}
}

func (r *WidgetREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	var namespace string

	// Get namespace from request context (this is how Kubernetes passes the namespace)
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.parseKey(name)
		} else {
			namespace = "default" // fallback
		}
	}

	// For namespace-specific endpoints (like widgetsmce, widgetsdefault),
	// the r.namespace field is just for filtering/validation, but we still
	// use the namespace from the request context for the actual operation

	return r.storage.Get(namespace, name)
}

func (r *WidgetREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	var namespace string

	// For namespace-specific endpoints (like widgetsmce, widgetsdefault),
	// use the configured namespace to filter the view
	if r.namespace != "" {
		namespace = r.namespace
	} else {
		// For the main widgets endpoint, get namespace from request context
		requestInfo, ok := request.RequestInfoFrom(ctx)
		if ok && requestInfo.Namespace != "" {
			namespace = requestInfo.Namespace
		}
		// If no namespace specified, list all (namespace = "")
	}

	return r.storage.List(namespace)
}

func (r *WidgetREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc,
	options *metav1.CreateOptions) (runtime.Object, error) {
	widget := obj.(*Widget)
	widget.TypeMeta = metav1.TypeMeta{
		APIVersion: common.GroupName + "/" + common.APIVersion,
		Kind:       "Widget",
	}
	return r.storage.Create(widget)
}

func (r *WidgetREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {

	var namespace string

	// Get namespace from request context (this is how Kubernetes passes the namespace)
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.parseKey(name)
		} else {
			namespace = "default" // fallback
		}
	}

	oldObj, err := r.storage.Get(namespace, name)
	if err != nil {
		return nil, false, err
	}

	updatedObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	widget := updatedObj.(*Widget)
	widget.Name = name
	widget.Namespace = namespace
	updatedWidget, err := r.storage.Update(widget)
	return updatedWidget, false, err
}

func (r *WidgetREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {

	var namespace string

	// Get namespace from request context (this is how Kubernetes passes the namespace)
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if ok && requestInfo.Namespace != "" {
		namespace = requestInfo.Namespace
	} else {
		// Check if the name contains namespace info (format: namespace/name)
		if strings.Contains(name, "/") {
			namespace, name = r.storage.parseKey(name)
		} else {
			namespace = "default" // fallback
		}
	}

	obj, err := r.storage.Get(namespace, name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(namespace, name)
	return obj, true, err
}

func (r *WidgetREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *WidgetREST) ConvertToTable(ctx context.Context, object runtime.Object,
	tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: common.GroupName, Resource: "widgets"}).
		ConvertToTable(ctx, object, tableOptions)
}

func (r *WidgetREST) NamespaceScoped() bool {
	return true
}

func (r *WidgetREST) GetSingularName() string {
	return "widget"
}

// resourceToKind converts a resource name to a Kind name (e.g., "widgetsdefault" -> "Widgetsdefault")
func resourceToKind(resource string) string {
	if resource == "" || resource == "widgets" {
		return "Widget"
	}
	// Capitalize first letter
	if len(resource) > 0 {
		resource = strings.ToUpper(resource[:1]) + resource[1:]
	}
	return resource
}

func (r *WidgetREST) GroupVersionKind(containingGV schema.GroupVersion) schema.GroupVersionKind {
	kind := resourceToKind(r.resourceName)
	return schema.GroupVersionKind{
		Group:   containingGV.Group,
		Version: containingGV.Version,
		Kind:    kind,
	}
}

func (r *WidgetREST) Destroy() {
	// Cleanup resources if needed
}

func (r *WidgetREST) GetStorage() *MemoryStorage {
	return r.storage
}

func NewNSWidgetREST(ns string, resourceName string) *WidgetREST {
	return &WidgetREST{
		storage:      NewMemoryStorage(),
		namespace:    ns,
		resourceName: resourceName,
	}
}

func NewNSWidgetRESTWithSharedStorage(ns string, resourceName string, sharedStorage *MemoryStorage) *WidgetREST {
	return &WidgetREST{
		storage:      sharedStorage,
		namespace:    ns,
		resourceName: resourceName,
	}
}
