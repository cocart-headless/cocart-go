package cocart

import (
	"errors"
	"testing"
)

func TestMemoryStorage(t *testing.T) {
	s := NewMemoryStorage()

	// Get missing key
	_, err := s.Get("missing")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}

	// Set and get
	if err := s.Set("key", "value"); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	v, err := s.Get("key")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if v != "value" {
		t.Errorf("Get = %s, want value", v)
	}

	// Delete
	if err := s.Delete("key"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	_, err = s.Get("key")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Error("key should be deleted")
	}

	// Delete non-existent key (should not error)
	if err := s.Delete("nonexistent"); err != nil {
		t.Fatalf("Delete non-existent error: %v", err)
	}
}

func TestMemoryStorageOverwrite(t *testing.T) {
	s := NewMemoryStorage()

	s.Set("key", "v1")
	s.Set("key", "v2")

	v, _ := s.Get("key")
	if v != "v2" {
		t.Errorf("expected v2, got %s", v)
	}
}
