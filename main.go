package main

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/apiserver/pkg/registry/generic"

	rrest "k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"
)

// ---------------------
// 1) Widget resource
// ---------------------
type Widget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WidgetSpec `json:"spec,omitempty"`
}

type WidgetSpec struct {
	Size int `json:"size"`
}

type WidgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Widget `json:"items"`
}

// Implement runtime.Object
func (w *Widget) DeepCopyObject() runtime.Object {
	out := new(Widget)
	*out = *w
	out.ObjectMeta = *w.ObjectMeta.DeepCopy()
	return out
}

func (wl *WidgetList) DeepCopyObject() runtime.Object {
	out := new(WidgetList)
	*out = *wl
	out.Items = make([]Widget, len(wl.Items))
	for i := range wl.Items {
		out.Items[i] = *wl.Items[i].DeepCopyObject().(*Widget)
	}
	return out
}

// Implement resource.Object
func (w *Widget) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "things.myorg.io",
		Version:  "v1alpha1",
		Resource: "widgets",
	}
}
func (w *Widget) IsStorageVersion() bool                  { return true }
func (w *Widget) New() runtime.Object                     { return &Widget{} }
func (w *Widget) NewList() runtime.Object                 { return &WidgetList{} }
func (w *Widget) GetObjectMeta() *metav1.ObjectMeta       { return &w.ObjectMeta }
func (w *Widget) NamespaceScoped() bool                   { return true }
func (w *Widget) ValidateCreate() error                   { return nil }
func (w *Widget) ValidateUpdate(old runtime.Object) error { return nil }
func (w *Widget) ValidateDelete() error                   { return nil }

// ---------------------
// 2) In-memory storage implementing rest.StandardStorage
// ---------------------
type WidgetStorage struct {
	mu    sync.Mutex
	items map[string]*Widget
}

func NewWidgetStorage() *WidgetStorage {
	return &WidgetStorage{
		items: make(map[string]*Widget),
	}
}

// Storage interface
func (s *WidgetStorage) New() runtime.Object {
	return &Widget{}
}

// // Implement rest.StandardStorage by wrapping genericregistry.Store
// func (s *WidgetStorage) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
// 	w := obj.(*Widget)
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	if w.Name == "" {
// 		w.Name = string(uuid.NewUUID())
// 	}
// 	s.items[w.Name] = w.DeepCopyObject().(*Widget)
// 	return w.DeepCopyObject(), nil
// }

// func (s *WidgetStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	w, ok := s.items[name]
// 	if !ok {
// 		return nil, fmt.Errorf("widget %s not found", name)
// 	}
// 	return w.DeepCopyObject(), nil
// }

// func (s *WidgetStorage) List(ctx context.Context, options *metav1.ListOptions) (runtime.Object, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	list := &WidgetList{}
// 	for _, w := range s.items {
// 		list.Items = append(list.Items, *w.DeepCopyObject().(*Widget))
// 	}
// 	return list, nil
// }

// func (s *WidgetStorage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	w, ok := s.items[name]
// 	if !ok {
// 		return nil, false, fmt.Errorf("widget %s not found", name)
// 	}
// 	delete(s.items, name)
// 	return w.DeepCopyObject(), true, nil
// }

// func (s *WidgetStorage) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	existing, ok := s.items[name]
// 	var old runtime.Object
// 	if ok {
// 		old = existing.DeepCopyObject()
// 	} else {
// 		old = &Widget{}
// 	}
// 	obj, err := objInfo.GetUpdatedObject(ctx, old)
// 	if err != nil {
// 		return nil, false, err
// 	}
// 	w := obj.(*Widget)
// 	s.items[name] = w.DeepCopyObject().(*Widget)
// 	return w.DeepCopyObject(), !ok, nil
// }

// ---------------------
// 3) StorageProvider for WithResourceAndHandler
// ---------------------
func WidgetStorageProvider(scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rrest.Storage, error) {
	return NewWidgetStorage(), nil
}

// ---------------------
// 4) main
// ---------------------
func main() {
	if err := builder.APIServer.
		WithResourceAndHandler(&Widget{}, WidgetStorageProvider).
		WithLocalDebugExtension().
		Execute(); err != nil {
		panic(err)
	}
}
