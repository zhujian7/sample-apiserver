package widgets

import (
	"fmt"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWidgetStorage_Create(t *testing.T) {
	storage := NewMemoryStorage()

	widget := &Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-widget",
		},
		Spec: WidgetSpec{
			Name:        "Test Widget",
			Description: "A test widget",
			Size:        42,
		},
	}

	// Test successful creation
	created, err := storage.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	if created.Name != "test-widget" {
		t.Errorf("Expected name 'test-widget', got '%s'", created.Name)
	}

	if created.Spec.Size != 42 {
		t.Errorf("Expected size 42, got %d", created.Spec.Size)
	}

	if created.Status.Phase != "Active" {
		t.Errorf("Expected status 'Active', got '%s'", created.Status.Phase)
	}

	if created.ResourceVersion == "" {
		t.Error("ResourceVersion should be set")
	}

	if created.UID == "" {
		t.Error("UID should be set")
	}

	// Test duplicate creation
	_, err = storage.Create(widget)
	if err == nil {
		t.Error("Expected error when creating duplicate widget")
	}
}

func TestWidgetStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()

	// Test getting non-existent widget
	_, err := storage.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent widget")
	}

	// Create a widget
	widget := &Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-widget",
		},
		Spec: WidgetSpec{
			Name:        "Test Widget",
			Description: "A test widget",
			Size:        42,
		},
	}
	_, err = storage.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Test getting existing widget
	retrieved, err := storage.Get("test-widget")
	if err != nil {
		t.Fatalf("Failed to get widget: %v", err)
	}

	if retrieved.Name != "test-widget" {
		t.Errorf("Expected name 'test-widget', got '%s'", retrieved.Name)
	}

	if retrieved.Spec.Size != 42 {
		t.Errorf("Expected size 42, got %d", retrieved.Spec.Size)
	}
}

func TestWidgetStorage_Update(t *testing.T) {
	storage := NewMemoryStorage()

	// Test updating non-existent widget
	widget := &Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-existent",
		},
		Spec: WidgetSpec{
			Size: 100,
		},
	}
	_, err := storage.Update(widget)
	if err == nil {
		t.Error("Expected error when updating non-existent widget")
	}

	// Create a widget
	originalWidget := &Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-widget",
		},
		Spec: WidgetSpec{
			Name:        "Test Widget",
			Description: "A test widget",
			Size:        42,
		},
	}
	created, err := storage.Create(originalWidget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Store original ResourceVersion before update
	originalResourceVersion := created.ResourceVersion

	// Update the widget (add small delay to ensure different timestamp)
	time.Sleep(time.Millisecond)
	created.Spec.Size = 100
	created.Spec.Description = "Updated description"
	updated, err := storage.Update(created)
	if err != nil {
		t.Fatalf("Failed to update widget: %v", err)
	}

	if updated.Spec.Size != 100 {
		t.Errorf("Expected size 100, got %d", updated.Spec.Size)
	}

	if updated.Spec.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Spec.Description)
	}

	// ResourceVersion should be updated
	if updated.ResourceVersion == originalResourceVersion {
		t.Error("ResourceVersion should be updated")
	}

	// UID and CreationTimestamp should remain the same
	if updated.UID != created.UID {
		t.Error("UID should remain the same")
	}

	if !updated.CreationTimestamp.Equal(&created.CreationTimestamp) {
		t.Error("CreationTimestamp should remain the same")
	}
}

func TestWidgetStorage_Delete(t *testing.T) {
	storage := NewMemoryStorage()

	// Test deleting non-existent widget
	err := storage.Delete("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent widget")
	}

	// Create a widget
	widget := &Widget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-widget",
		},
		Spec: WidgetSpec{
			Name: "Test Widget",
			Size: 42,
		},
	}
	_, err = storage.Create(widget)
	if err != nil {
		t.Fatalf("Failed to create widget: %v", err)
	}

	// Delete the widget
	err = storage.Delete("test-widget")
	if err != nil {
		t.Fatalf("Failed to delete widget: %v", err)
	}

	// Verify it's deleted
	_, err = storage.Get("test-widget")
	if err == nil {
		t.Error("Widget should be deleted")
	}
}

func TestWidgetStorage_List(t *testing.T) {
	storage := NewMemoryStorage()

	// Test listing empty storage
	list, err := storage.List()
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	if len(list.Items) != 0 {
		t.Errorf("Expected 0 widgets, got %d", len(list.Items))
	}

	// Create multiple widgets
	for i := 0; i < 3; i++ {
		widget := &Widget{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("widget-%d", i),
			},
			Spec: WidgetSpec{
				Name: fmt.Sprintf("Widget %d", i),
				Size: int32(i * 10),
			},
		}
		_, err = storage.Create(widget)
		if err != nil {
			t.Fatalf("Failed to create widget %d: %v", i, err)
		}
	}

	// List all widgets
	list, err = storage.List()
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	if len(list.Items) != 3 {
		t.Errorf("Expected 3 widgets, got %d", len(list.Items))
	}

	// Verify list metadata
	if list.Kind != "WidgetList" {
		t.Errorf("Expected kind 'WidgetList', got '%s'", list.Kind)
	}
}

func TestWidgetStorage_ThreadSafety(t *testing.T) {
	storage := NewMemoryStorage()
	const numGoroutines = 10
	const numOperations = 100

	// Test concurrent creates
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				widget := &Widget{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("widget-%d-%d", id, j),
					},
					Spec: WidgetSpec{
						Name: fmt.Sprintf("Widget %d-%d", id, j),
						Size: int32(j),
					},
				}
				_, err := storage.Create(widget)
				if err != nil {
					t.Errorf("Failed to create widget %d-%d: %v", id, j, err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all widgets were created
	list, err := storage.List()
	if err != nil {
		t.Fatalf("Failed to list widgets: %v", err)
	}

	expected := numGoroutines * numOperations
	if len(list.Items) != expected {
		t.Errorf("Expected %d widgets, got %d", expected, len(list.Items))
	}
}
