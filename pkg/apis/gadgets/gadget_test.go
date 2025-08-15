package gadgets

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGadgetStorage_Create(t *testing.T) {
	storage := NewGadgetStorage()

	gadget := &Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gadget",
		},
		Spec: GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 10,
		},
	}

	// Test successful creation
	created, err := storage.Create(gadget)
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	if created.Name != "test-gadget" {
		t.Errorf("Expected name 'test-gadget', got '%s'", created.Name)
	}

	if created.Spec.Type != "sensor" {
		t.Errorf("Expected type 'sensor', got '%s'", created.Spec.Type)
	}

	if created.Spec.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", created.Spec.Priority)
	}

	if created.Status.State != "Active" {
		t.Errorf("Expected state 'Active', got '%s'", created.Status.State)
	}

	if created.ResourceVersion == "" {
		t.Error("ResourceVersion should be set")
	}

	if created.UID == "" {
		t.Error("UID should be set")
	}

	// Test duplicate creation
	_, err = storage.Create(gadget)
	if err == nil {
		t.Error("Expected error when creating duplicate gadget")
	}
}

func TestGadgetStorage_Get(t *testing.T) {
	storage := NewGadgetStorage()

	// Test getting non-existent gadget
	_, err := storage.Get("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent gadget")
	}

	// Create a gadget
	gadget := &Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gadget",
		},
		Spec: GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 10,
		},
	}
	_, err = storage.Create(gadget)
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	// Test getting existing gadget
	retrieved, err := storage.Get("test-gadget")
	if err != nil {
		t.Fatalf("Failed to get gadget: %v", err)
	}

	if retrieved.Name != "test-gadget" {
		t.Errorf("Expected name 'test-gadget', got '%s'", retrieved.Name)
	}

	if retrieved.Spec.Type != "sensor" {
		t.Errorf("Expected type 'sensor', got '%s'", retrieved.Spec.Type)
	}

	if retrieved.Spec.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", retrieved.Spec.Priority)
	}
}

func TestGadgetStorage_Update(t *testing.T) {
	storage := NewGadgetStorage()

	// Test updating non-existent gadget
	gadget := &Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "non-existent",
		},
		Spec: GadgetSpec{
			Priority: 20,
		},
	}
	_, err := storage.Update(gadget)
	if err == nil {
		t.Error("Expected error when updating non-existent gadget")
	}

	// Create a gadget
	originalGadget := &Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gadget",
		},
		Spec: GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Enabled:  true,
			Priority: 10,
		},
	}
	created, err := storage.Create(originalGadget)
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	// Update the gadget
	created.Spec.Priority = 20
	created.Spec.Version = "v2.0"
	created.Spec.Enabled = false
	updated, err := storage.Update(created)
	if err != nil {
		t.Fatalf("Failed to update gadget: %v", err)
	}

	if updated.Spec.Priority != 20 {
		t.Errorf("Expected priority 20, got %d", updated.Spec.Priority)
	}

	if updated.Spec.Version != "v2.0" {
		t.Errorf("Expected version 'v2.0', got '%s'", updated.Spec.Version)
	}

	if updated.Spec.Enabled != false {
		t.Errorf("Expected enabled false, got %t", updated.Spec.Enabled)
	}

	// ResourceVersion should be updated
	if updated.ResourceVersion == created.ResourceVersion {
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

func TestGadgetStorage_Delete(t *testing.T) {
	storage := NewGadgetStorage()

	// Test deleting non-existent gadget
	err := storage.Delete("non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent gadget")
	}

	// Create a gadget
	gadget := &Gadget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gadget",
		},
		Spec: GadgetSpec{
			Type:     "sensor",
			Version:  "v1.0",
			Priority: 10,
		},
	}
	_, err = storage.Create(gadget)
	if err != nil {
		t.Fatalf("Failed to create gadget: %v", err)
	}

	// Delete the gadget
	err = storage.Delete("test-gadget")
	if err != nil {
		t.Fatalf("Failed to delete gadget: %v", err)
	}

	// Verify it's deleted
	_, err = storage.Get("test-gadget")
	if err == nil {
		t.Error("Gadget should be deleted")
	}
}

func TestGadgetStorage_List(t *testing.T) {
	storage := NewGadgetStorage()

	// Test listing empty storage
	list, err := storage.List()
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	if len(list.Items) != 0 {
		t.Errorf("Expected 0 gadgets, got %d", len(list.Items))
	}

	// Create multiple gadgets
	for i := 0; i < 3; i++ {
		gadget := &Gadget{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("gadget-%d", i),
			},
			Spec: GadgetSpec{
				Type:     fmt.Sprintf("type-%d", i),
				Version:  fmt.Sprintf("v1.%d", i),
				Enabled:  i%2 == 0,
				Priority: int32(i * 5),
			},
		}
		_, err = storage.Create(gadget)
		if err != nil {
			t.Fatalf("Failed to create gadget %d: %v", i, err)
		}
	}

	// List all gadgets
	list, err = storage.List()
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	if len(list.Items) != 3 {
		t.Errorf("Expected 3 gadgets, got %d", len(list.Items))
	}

	// Verify list metadata
	if list.Kind != "GadgetList" {
		t.Errorf("Expected kind 'GadgetList', got '%s'", list.Kind)
	}
}

func TestGadgetStorage_ThreadSafety(t *testing.T) {
	storage := NewGadgetStorage()
	const numGoroutines = 10
	const numOperations = 100

	// Test concurrent creates
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				gadget := &Gadget{
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("gadget-%d-%d", id, j),
					},
					Spec: GadgetSpec{
						Type:     fmt.Sprintf("type-%d", id),
						Version:  fmt.Sprintf("v%d.%d", id, j),
						Enabled:  j%2 == 0,
						Priority: int32(j),
					},
				}
				_, err := storage.Create(gadget)
				if err != nil {
					t.Errorf("Failed to create gadget %d-%d: %v", id, j, err)
				}
			}
			done <- true
		}(id)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all gadgets were created
	list, err := storage.List()
	if err != nil {
		t.Fatalf("Failed to list gadgets: %v", err)
	}

	expected := numGoroutines * numOperations
	if len(list.Items) != expected {
		t.Errorf("Expected %d gadgets, got %d", expected, len(list.Items))
	}
}