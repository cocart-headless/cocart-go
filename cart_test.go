package cocart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCartGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !containsStr(r.URL.Path, "/cocart/v2/cart") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "item_count": 0})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Cart().Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Error("expected success")
	}
}

func TestCartAddItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["id"] != "42" {
			t.Errorf("expected id=42, got %v", body["id"])
		}
		if body["quantity"] != "1" {
			t.Errorf("expected quantity=1, got %v", body["quantity"])
		}
		w.Write([]byte(`{"item_key":"abc123"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Cart().AddItem(context.Background(), 42, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Error("expected success")
	}
}

func TestCartAddItemValidation(t *testing.T) {
	c := NewClient("https://example.com")

	_, err := c.Cart().AddItem(context.Background(), 0, 1)
	if err == nil {
		t.Error("expected validation error for product ID 0")
	}

	_, err = c.Cart().AddItem(context.Background(), 1, 0)
	if err == nil {
		t.Error("expected validation error for quantity 0")
	}
}

func TestCartClear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"items":[],"item_count":0}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().Clear(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartApplyCoupon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["coupon"] != "SAVE10" {
			t.Errorf("expected coupon=SAVE10, got %v", body["coupon"])
		}
		w.Write([]byte(`{"coupons":[{"coupon":"SAVE10"}]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().ApplyCoupon(context.Background(), "SAVE10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartAddItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		items, ok := body["items"].([]any)
		if !ok {
			t.Fatal("expected items array")
		}
		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}
		first := items[0].(map[string]any)
		if first["id"] != "123" {
			t.Errorf("expected id=123, got %v", first["id"])
		}
		if first["quantity"] != "2" {
			t.Errorf("expected quantity=2, got %v", first["quantity"])
		}
		second := items[1].(map[string]any)
		if second["id"] != "456" {
			t.Errorf("expected id=456, got %v", second["id"])
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().AddItems(context.Background(), []CartItemData{
		{ID: "123", Quantity: "2"},
		{ID: "456", Quantity: "1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartCreateRequiresBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	_, err := c.Cart().Create(context.Background())
	if err == nil {
		t.Error("expected version error for legacy plugin")
	}
}
