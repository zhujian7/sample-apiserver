//go:build integration
// +build integration

package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"example.com/mytest-apiserver/pkg/apis/gadgets"
	"example.com/mytest-apiserver/pkg/apis/widgets"
)

// TestWidgetGadgetIntegration tests the interaction between Widget and Gadget resources
func TestWidgetGadgetIntegration(t *testing.T) {
	// Create REST handlers
	widgetREST := widgets.NewWidgetREST()
	gadgetREST := gadgets.NewGadgetREST()
	ctx := context.Background()

	// Test scenario: Create a widget and related gadgets
	widget := &widgets.Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "main-widget",
			Namespace: "default",
		},
		Spec: widgets.WidgetSpec{
			Name:        "Main Control Widget",
			Description: "Primary control interface",
			Size:        100,
		},
	}

	// Create the widget
	_, err := widgetREST.Create(ctx, widget, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Create related gadgets
	testGadgets := []*gadgets.Gadget{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-1",
				Namespace: "default",
			},
			Spec: gadgets.GadgetSpec{
				Type:     "temperature-sensor",
				Version:  "v1.0",
				Enabled:  true,
				Priority: 10,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sensor-2",
				Namespace: "default",
			},
			Spec: gadgets.GadgetSpec{
				Type:     "humidity-sensor",
				Version:  "v1.1",
				Enabled:  true,
				Priority: 5,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "actuator-1",
				Namespace: "default",
			},
			Spec: gadgets.GadgetSpec{
				Type:     "motor-controller",
				Version:  "v2.0",
				Enabled:  false,
				Priority: 20,
			},
		},
	}

	// Create all gadgets
	for i, gadget := range testGadgets {
		_, err := gadgetREST.Create(ctx, gadget, nil, &metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Failed to create gadget %d: %v", i, err)
		}
	}

	// List all widgets
	widgetList, err := widgetREST.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	widgetItems := widgetList.(*widgets.WidgetList)
	if len(widgetItems.Items) != 1 {
		t.Errorf("Expected 1 widget, got %d", len(widgetItems.Items))
	}

	// List all gadgets
	gadgetList, err := gadgetREST.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	gadgetItems := gadgetList.(*gadgets.GadgetList)
	if len(gadgetItems.Items) != 3 {
		t.Errorf("Expected 3 gadgets, got %d", len(gadgetItems.Items))
	}

	// Test updating widget based on gadget states
	retrievedWidget, err := widgetREST.Get(ctx, "main-widget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get widget: %v", err)
	}

	mainWidget := retrievedWidget.(*widgets.Widget)
	// Simulate updating widget size based on number of active gadgets
	activeGadgets := 0
	for _, gadget := range gadgetItems.Items {
		if gadget.Spec.Enabled {
			activeGadgets++
		}
	}

	mainWidget.Spec.Size = int32(activeGadgets * 50)
	mainWidget.Spec.Description = "Widget with connected gadgets"

	// Mock update info for testing
	updateInfo := &mockUpdateInfo{updatedObj: mainWidget}
	_, _, err = widgetREST.Update(ctx, "main-widget", updateInfo, nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update widget: %v", err)
	}

	// Verify the update
	updatedWidget, err := widgetREST.Get(ctx, "main-widget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated widget: %v", err)
	}

	finalWidget := updatedWidget.(*widgets.Widget)
	expectedSize := int32(activeGadgets * 50)
	if finalWidget.Spec.Size != expectedSize {
		t.Errorf("Expected widget size %d, got %d", expectedSize, finalWidget.Spec.Size)
	}

	// Clean up - delete all resources
	for _, gadget := range gadgetItems.Items {
		_, _, err := gadgetREST.Delete(ctx, gadget.Name, nil, &metav1.DeleteOptions{})
		if err != nil {
			t.Errorf("Failed to delete gadget %s: %v", gadget.Name, err)
		}
	}

	_, _, err = widgetREST.Delete(ctx, "main-widget", nil, &metav1.DeleteOptions{})
	if err != nil {
		t.Errorf("Failed to delete widget: %v", err)
	}
}

// TestConcurrentOperations tests thread safety with concurrent operations
func TestConcurrentOperations(t *testing.T) {
	widgetREST := widgets.NewWidgetREST()
	gadgetREST := gadgets.NewGadgetREST()
	ctx := context.Background()

	const numWorkers = 5
	const numOperations = 20

	// Test concurrent widget operations
	done := make(chan bool, numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for j := 0; j < numOperations; j++ {
				// Create widget
				widget := &widgets.Widget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("widget-%d-%d", workerID, j),
						Namespace: "default",
					},
					Spec: widgets.WidgetSpec{
						Name:        fmt.Sprintf("Widget %d-%d", workerID, j),
						Description: "Concurrent test widget",
						Size:        int32(j),
					},
				}

				_, err := widgetREST.Create(ctx, widget, nil, &metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Worker %d: Failed to create widget %d: %v", workerID, j, err)
					continue
				}

				// Create corresponding gadget
				gadget := &gadgets.Gadget{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("gadget-%d-%d", workerID, j),
						Namespace: "default",
					},
					Spec: gadgets.GadgetSpec{
						Type:     "test-sensor",
						Version:  "v1.0",
						Enabled:  true,
						Priority: int32(workerID * 10),
					},
				}

				_, err = gadgetREST.Create(ctx, gadget, nil, &metav1.CreateOptions{})
				if err != nil {
					t.Errorf("Worker %d: Failed to create gadget %d: %v", workerID, j, err)
				}

				// Small delay to interleave operations
				time.Sleep(time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all workers to complete
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Verify all resources were created
	widgetList, err := widgetREST.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	widgetItems := widgetList.(*widgets.WidgetList)
	expectedWidgets := numWorkers * numOperations
	if len(widgetItems.Items) != expectedWidgets {
		t.Errorf("Expected %d widgets, got %d", expectedWidgets, len(widgetItems.Items))
	}

	gadgetList, err := gadgetREST.List(ctx, &internalversion.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	gadgetItems := gadgetList.(*gadgets.GadgetList)
	expectedGadgets := numWorkers * numOperations
	if len(gadgetItems.Items) != expectedGadgets {
		t.Errorf("Expected %d gadgets, got %d", expectedGadgets, len(gadgetItems.Items))
	}
}

// TestResourceLifecycle tests the complete lifecycle of resources
func TestResourceLifecycle(t *testing.T) {
	widgetREST := widgets.NewWidgetREST()
	gadgetREST := gadgets.NewGadgetREST()
	ctx := context.Background()

	// Phase 1: Create resources
	widget := &widgets.Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lifecycle-widget",
			Namespace: "default",
		},
		Spec: widgets.WidgetSpec{
			Name:        "Lifecycle Test Widget",
			Description: "Testing complete lifecycle",
			Size:        50,
		},
	}

	createdWidget, err := widgetREST.Create(ctx, widget, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	widget = createdWidget.(*widgets.Widget)

	gadget := &gadgets.Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lifecycle-gadget",
			Namespace: "default",
		},
		Spec: gadgets.GadgetSpec{
			Type:     "lifecycle-sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 15,
		},
	}

	createdGadget, err := gadgetREST.Create(ctx, gadget, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	gadget = createdGadget.(*gadgets.Gadget)

	// Phase 2: Read and verify
	retrievedWidget, err := widgetREST.Get(ctx, "lifecycle-widget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get widget: %v", err)
	}

	if retrievedWidget.(*widgets.Widget).Spec.Size != 50 {
		t.Error("Widget spec not preserved after creation")
	}

	retrievedGadget, err := gadgetREST.Get(ctx, "lifecycle-gadget", &metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get gadget: %v", err)
	}

	if retrievedGadget.(*gadgets.Gadget).Spec.Priority != 15 {
		t.Error("Gadget spec not preserved after creation")
	}

	// Phase 3: Update
	widget.Spec.Size = 75
	widget.Spec.Description = "Updated lifecycle widget"

	updateInfo := &mockUpdateInfo{updatedObj: widget}
	updatedWidget, _, err := widgetREST.Update(ctx, "lifecycle-widget", updateInfo, nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update widget: %v", err)
	}

	if updatedWidget.(*widgets.Widget).Spec.Size != 75 {
		t.Error("Widget update failed")
	}

	gadget.Spec.Priority = 25
	gadget.Spec.Enabled = false

	gadgetUpdateInfo := &mockUpdateInfo{updatedObj: gadget}
	updatedGadget, _, err := gadgetREST.Update(ctx, "lifecycle-gadget", gadgetUpdateInfo, nil, nil, false, &metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Failed to update gadget: %v", err)
	}

	if updatedGadget.(*gadgets.Gadget).Spec.Priority != 25 {
		t.Error("Gadget update failed")
	}

	// Phase 4: Delete
	_, _, err = widgetREST.Delete(ctx, "lifecycle-widget", nil, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete widget: %v", err)
	}

	_, _, err = gadgetREST.Delete(ctx, "lifecycle-gadget", nil, &metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Failed to delete gadget: %v", err)
	}

	// Phase 5: Verify deletion
	_, err = widgetREST.Get(ctx, "lifecycle-widget", &metav1.GetOptions{})
	if err == nil {
		t.Error("Widget should be deleted")
	}

	_, err = gadgetREST.Get(ctx, "lifecycle-gadget", &metav1.GetOptions{})
	if err == nil {
		t.Error("Gadget should be deleted")
	}
}

// mockUpdateInfo implements rest.UpdatedObjectInfo for testing
type mockUpdateInfo struct {
	updatedObj runtime.Object
}

func (m *mockUpdateInfo) UpdatedObject(ctx context.Context, oldObj runtime.Object) (runtime.Object, error) {
	return m.updatedObj, nil
}

func (m *mockUpdateInfo) Preconditions() *metav1.Preconditions {
	return nil
}
