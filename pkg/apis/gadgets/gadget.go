package gadgets

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

type Gadget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GadgetSpec   `json:"spec,omitempty"`
	Status            GadgetStatus `json:"status,omitempty"`
}

type GadgetSpec struct {
	Type     string `json:"type"`
	Version  string `json:"version"`
	Enabled  bool   `json:"enabled"`
	Priority int32  `json:"priority"`
}

type GadgetStatus struct {
	State string `json:"state,omitempty"`
}

type GadgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gadget `json:"items"`
}

func (g *Gadget) DeepCopyObject() runtime.Object {
	return &Gadget{
		TypeMeta:   g.TypeMeta,
		ObjectMeta: *g.ObjectMeta.DeepCopy(),
		Spec:       g.Spec,
		Status:     g.Status,
	}
}

func (gl *GadgetList) DeepCopyObject() runtime.Object {
	out := &GadgetList{
		TypeMeta: gl.TypeMeta,
		ListMeta: gl.ListMeta,
		Items:    make([]Gadget, len(gl.Items)),
	}
	for i := range gl.Items {
		out.Items[i] = *gl.Items[i].DeepCopyObject().(*Gadget)
	}
	return out
}

type GadgetStorage struct {
	mu             sync.RWMutex
	gadgets        map[string]*Gadget
	versionCounter int64
}

func NewGadgetStorage() *GadgetStorage {
	return &GadgetStorage{
		gadgets:        make(map[string]*Gadget),
		versionCounter: 1,
	}
}

func (s *GadgetStorage) Get(name string) (*Gadget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	gadget, exists := s.gadgets[name]
	if !exists {
		return nil, fmt.Errorf("gadget %s not found", name)
	}
	return gadget.DeepCopyObject().(*Gadget), nil
}

func (s *GadgetStorage) List() (*GadgetList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &GadgetList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: common.GroupName + "/" + common.APIVersion,
			Kind:       "GadgetList",
		},
		Items: make([]Gadget, 0, len(s.gadgets)),
	}

	for _, gadget := range s.gadgets {
		list.Items = append(list.Items, *gadget.DeepCopyObject().(*Gadget))
	}

	return list, nil
}

func (s *GadgetStorage) Create(gadget *Gadget) (*Gadget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gadget.Name == "" {
		gadget.Name = string(uuid.NewUUID())
	}

	if _, exists := s.gadgets[gadget.Name]; exists {
		return nil, fmt.Errorf("gadget %s already exists", gadget.Name)
	}

	now := metav1.NewTime(time.Now())
	gadget.CreationTimestamp = now
	gadget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++
	gadget.UID = uuid.NewUUID()
	gadget.Status.State = "Active"

	s.gadgets[gadget.Name] = gadget.DeepCopyObject().(*Gadget)
	return gadget, nil
}

func (s *GadgetStorage) Update(gadget *Gadget) (*Gadget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.gadgets[gadget.Name]
	if !exists {
		return nil, fmt.Errorf("gadget %s not found", gadget.Name)
	}

	gadget.CreationTimestamp = existing.CreationTimestamp
	gadget.UID = existing.UID
	gadget.ResourceVersion = fmt.Sprintf("%d", s.versionCounter)
	s.versionCounter++

	s.gadgets[gadget.Name] = gadget.DeepCopyObject().(*Gadget)
	return gadget, nil
}

func (s *GadgetStorage) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.gadgets[name]; !exists {
		return fmt.Errorf("gadget %s not found", name)
	}

	delete(s.gadgets, name)
	return nil
}

type GadgetREST struct {
	storage *GadgetStorage
}

// Ensure GadgetREST implements the required interfaces
var _ rest.Creater = &GadgetREST{}
var _ rest.Lister = &GadgetREST{}
var _ rest.Getter = &GadgetREST{}
var _ rest.Updater = &GadgetREST{}
var _ rest.GracefulDeleter = &GadgetREST{}
var _ rest.Scoper = &GadgetREST{}

func NewGadgetREST() *GadgetREST {
	return &GadgetREST{
		storage: NewGadgetStorage(),
	}
}

func (r *GadgetREST) New() runtime.Object {
	return &Gadget{}
}

func (r *GadgetREST) NewList() runtime.Object {
	return &GadgetList{}
}

func (r *GadgetREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.storage.Get(name)
}

func (r *GadgetREST) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	return r.storage.List()
}

func (r *GadgetREST) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	gadget := obj.(*Gadget)
	gadget.TypeMeta = metav1.TypeMeta{
		APIVersion: common.GroupName + "/" + common.APIVersion,
		Kind:       "Gadget",
	}
	return r.storage.Create(gadget)
}

func (r *GadgetREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	oldObj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	updatedObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, err
	}

	gadget := updatedObj.(*Gadget)
	gadget.Name = name
	updatedGadget, err := r.storage.Update(gadget)
	return updatedGadget, false, err
}

func (r *GadgetREST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := r.storage.Get(name)
	if err != nil {
		return nil, false, err
	}

	err = r.storage.Delete(name)
	return obj, true, err
}

func (r *GadgetREST) Watch(ctx context.Context, options *metav1.ListOptions) (watch.Interface, error) {
	return nil, fmt.Errorf("watch not implemented")
}

func (r *GadgetREST) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return rest.NewDefaultTableConvertor(schema.GroupResource{Group: common.GroupName, Resource: "gadgets"}).ConvertToTable(ctx, object, tableOptions)
}

func (r *GadgetREST) NamespaceScoped() bool {
	return true
}

func (r *GadgetREST) GetSingularName() string {
	return "gadget"
}
