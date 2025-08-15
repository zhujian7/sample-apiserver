package main

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"example.com/mytest-apiserver/pkg/apis/gadgets"
	"example.com/mytest-apiserver/pkg/apis/widgets"
)

func TestSchemeRegistration(t *testing.T) {
	// Test that our resources are properly registered in the scheme
	gv := Scheme.PrioritizedVersionsForGroup("things.myorg.io")
	if len(gv) == 0 {
		t.Error("Expected group 'things.myorg.io' to be registered")
	}

	// Test Widget registration
	widgetGVK := schema.GroupVersionKind{
		Group:   "things.myorg.io",
		Version: "v1alpha1",
		Kind:    "Widget",
	}

	obj, err := Scheme.New(widgetGVK)
	if err != nil {
		t.Errorf("Failed to create Widget from scheme: %v", err)
	}

	if _, ok := obj.(*widgets.Widget); !ok {
		t.Errorf("Expected *widgets.Widget, got %T", obj)
	}

	// Test Gadget registration
	gadgetGVK := schema.GroupVersionKind{
		Group:   "things.myorg.io",
		Version: "v1alpha1",
		Kind:    "Gadget",
	}

	obj, err = Scheme.New(gadgetGVK)
	if err != nil {
		t.Errorf("Failed to create Gadget from scheme: %v", err)
	}

	if _, ok := obj.(*gadgets.Gadget); !ok {
		t.Errorf("Expected *gadgets.Gadget, got %T", obj)
	}
}

func TestWidgetREST_Interfaces(t *testing.T) {
	rest := widgets.NewWidgetREST()

	// Test that it implements required interfaces
	if rest == nil {
		t.Fatal("WidgetREST should not be nil")
	}

	// Test New method
	obj := rest.New()
	if _, ok := obj.(*widgets.Widget); !ok {
		t.Errorf("Expected *widgets.Widget, got %T", obj)
	}

	// Test NewList method
	listObj := rest.NewList()
	if _, ok := listObj.(*widgets.WidgetList); !ok {
		t.Errorf("Expected *widgets.WidgetList, got %T", listObj)
	}

	// Test NamespaceScoped
	if !rest.NamespaceScoped() {
		t.Error("Widget should be namespace scoped")
	}

	// Test GetSingularName
	if rest.GetSingularName() != "widget" {
		t.Errorf("Expected singular name 'widget', got '%s'", rest.GetSingularName())
	}
}

func TestGadgetREST_Interfaces(t *testing.T) {
	rest := gadgets.NewGadgetREST()

	// Test that it implements required interfaces
	if rest == nil {
		t.Fatal("GadgetREST should not be nil")
	}

	// Test New method
	obj := rest.New()
	if _, ok := obj.(*gadgets.Gadget); !ok {
		t.Errorf("Expected *gadgets.Gadget, got %T", obj)
	}

	// Test NewList method
	listObj := rest.NewList()
	if _, ok := listObj.(*gadgets.GadgetList); !ok {
		t.Errorf("Expected *gadgets.GadgetList, got %T", listObj)
	}

	// Test NamespaceScoped
	if !rest.NamespaceScoped() {
		t.Error("Gadget should be namespace scoped")
	}

	// Test GetSingularName
	if rest.GetSingularName() != "gadget" {
		t.Errorf("Expected singular name 'gadget', got '%s'", rest.GetSingularName())
	}
}

func TestWidgetREST_CRUD(t *testing.T) {
	rest := widgets.NewWidgetREST()
	ctx := context.Background()

	// Test Create
	widget := &widgets.Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-widget",
			Namespace: "default",
		},
		Spec: widgets.WidgetSpec{
			Name:        "Test Widget",
			Description: "A test widget",
			Size:        42,
		},
	}

	created, err := rest.Create(ctx, widget, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	createdWidget, ok := created.(*widgets.Widget)
	if !ok {
		t.Fatalf("Expected *widgets.Widget, got %T", created)
	}

	if createdWidget.Name != "test-widget" {
		t.Errorf("Expected name 'test-widget', got '%s'", createdWidget.Name)
	}

	// Test Get
	retrieved, err := rest.Get(ctx, "test-widget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get widget: %v", err)
	}

	retrievedWidget, ok := retrieved.(*widgets.Widget)
	if !ok {
		t.Fatalf("Expected *widgets.Widget, got %T", retrieved)
	}

	if retrievedWidget.Name != "test-widget" {
		t.Errorf("Expected name 'test-widget', got '%s'", retrievedWidget.Name)
	}

	// Test List
	listed, err := rest.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	listedWidgets, ok := listed.(*widgets.WidgetList)
	if !ok {
		t.Fatalf("Expected *widgets.WidgetList, got %T", listed)
	}

	if len(listedWidgets.Items) != 1 {
		t.Errorf("Expected 1 widget, got %d", len(listedWidgets.Items))
	}

	// Test Delete
	deleted, wasDeleted, err := rest.Delete(ctx, "test-widget", nil, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete widget: %v", err)
	}

	if !wasDeleted {
		t.Error("Expected widget to be deleted immediately")
	}

	deletedWidget, ok := deleted.(*widgets.Widget)
	if !ok {
		t.Fatalf("Expected *widgets.Widget, got %T", deleted)
	}

	if deletedWidget.Name != "test-widget" {
		t.Errorf("Expected deleted widget name 'test-widget', got '%s'", deletedWidget.Name)
	}

	// Verify deletion
	_, err = rest.Get(ctx, "test-widget", &metav1.GetOptions{})
	if err == nil {
		t.Error("Expected error when getting deleted widget")
	}
}

func TestGadgetREST_CRUD(t *testing.T) {
	rest := gadgets.NewGadgetREST()
	ctx := context.Background()

	// Test Create
	gadget := &gadgets.Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gadget",
			Namespace: "default",
		},
		Spec: gadgets.GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 10,
		},
	}

	created, err := rest.Create(ctx, gadget, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	createdGadget, ok := created.(*gadgets.Gadget)
	if !ok {
		t.Fatalf("Expected *gadgets.Gadget, got %T", created)
	}

	if createdGadget.Name != "test-gadget" {
		t.Errorf("Expected name 'test-gadget', got '%s'", createdGadget.Name)
	}

	// Test Get
	retrieved, err := rest.Get(ctx, "test-gadget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get gadget: %v", err)
	}

	retrievedGadget, ok := retrieved.(*gadgets.Gadget)
	if !ok {
		t.Fatalf("Expected *gadgets.Gadget, got %T", retrieved)
	}

	if retrievedGadget.Name != "test-gadget" {
		t.Errorf("Expected name 'test-gadget', got '%s'", retrievedGadget.Name)
	}

	// Test List
	listed, err := rest.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	listedGadgets, ok := listed.(*gadgets.GadgetList)
	if !ok {
		t.Fatalf("Expected *gadgets.GadgetList, got %T", listed)
	}

	if len(listedGadgets.Items) != 1 {
		t.Errorf("Expected 1 gadget, got %d", len(listedGadgets.Items))
	}

	// Test Delete
	deleted, wasDeleted, err := rest.Delete(ctx, "test-gadget", nil, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete gadget: %v", err)
	}

	if !wasDeleted {
		t.Error("Expected gadget to be deleted immediately")
	}

	deletedGadget, ok := deleted.(*gadgets.Gadget)
	if !ok {
		t.Fatalf("Expected *gadgets.Gadget, got %T", deleted)
	}

	if deletedGadget.Name != "test-gadget" {
		t.Errorf("Expected deleted gadget name 'test-gadget', got '%s'", deletedGadget.Name)
	}

	// Verify deletion
	_, err = rest.Get(ctx, "test-gadget", &metav1.GetOptions{})
	if err == nil {
		t.Error("Expected error when getting deleted gadget")
	}
}

func TestConfig_New(t *testing.T) {
	config := NewConfig()
	if config == nil {
		t.Fatal("Config should not be nil")
	}

	if config.GenericConfig == nil {
		t.Error("GenericConfig should not be nil")
	}

	// Skip the API server creation test as it requires TLS configuration
	// that's not available in test environments
	t.Skip("Skipping API server creation test - requires proper TLS/secure port configuration")
}

func TestDeepCopy(t *testing.T) {
	// Test Widget DeepCopy
	widget := &widgets.Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-widget",
			Namespace: "default",
		},
		Spec: widgets.WidgetSpec{
			Name:        "Test Widget",
			Description: "A test widget",
			Size:        42,
		},
		Status: widgets.WidgetStatus{
			Phase: "Active",
		},
	}

	copied := widget.DeepCopyObject()
	copiedWidget, ok := copied.(*widgets.Widget)
	if !ok {
		t.Fatalf("Expected *widgets.Widget, got %T", copied)
	}

	if copiedWidget.Name != widget.Name {
		t.Error("DeepCopy should preserve name")
	}

	if copiedWidget.Spec.Size != widget.Spec.Size {
		t.Error("DeepCopy should preserve spec fields")
	}

	// Test Gadget DeepCopy
	gadget := &gadgets.Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gadget",
			Namespace: "default",
		},
		Spec: gadgets.GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 10,
		},
		Status: gadgets.GadgetStatus{
			State: "Active",
		},
	}

	copied = gadget.DeepCopyObject()
	copiedGadget, ok := copied.(*gadgets.Gadget)
	if !ok {
		t.Fatalf("Expected *gadgets.Gadget, got %T", copied)
	}

	if copiedGadget.Name != gadget.Name {
		t.Error("DeepCopy should preserve name")
	}

	if copiedGadget.Spec.Priority != gadget.Spec.Priority {
		t.Error("DeepCopy should preserve spec fields")
	}
}
