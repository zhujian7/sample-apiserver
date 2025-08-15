package widgets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"

	"example.com/mytest-apiserver/pkg/common"
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
			APIVersion: common.GroupName + "/" + common.APIVersion,
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
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
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
	widget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

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

// Ensure WidgetREST implements the required interfaces
var _ rest.Creater = &WidgetREST{}
var _ rest.Lister = &WidgetREST{}
var _ rest.Getter = &WidgetREST{}
var _ rest.Updater = &WidgetREST{}
var _ rest.GracefulDeleter = &WidgetREST{}
var _ rest.Scoper = &WidgetREST{}

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

func (r *WidgetREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return r.storage.List()
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

func (r *WidgetREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc,
	options *metav1.DeleteOptions) (runtime.Object, bool, error) {
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
