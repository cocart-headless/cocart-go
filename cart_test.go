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
		if body["id"] != "100" {
			t.Errorf("expected id=100, got %v", body["id"])
		}
		quantity, ok := body["quantity"].(map[string]any)
		if !ok {
			t.Fatal("expected quantity map")
		}
		if quantity["123"] != "2" {
			t.Errorf("expected quantity[123]=2, got %v", quantity["123"])
		}
		if quantity["456"] != "1" {
			t.Errorf("expected quantity[456]=1, got %v", quantity["456"])
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().AddItems(context.Background(), 100, map[string]int{
		"123": 2,
		"456": 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartAddItemsRequiresItems(t *testing.T) {
	c := NewClient("https://example.com")
	_, err := c.Cart().AddItems(context.Background(), 100, map[string]int{})
	if err == nil {
		t.Error("expected validation error for empty items")
	}
}

func TestCartCreateRequiresBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	_, err := c.Cart().Create(context.Background())
	if err == nil {
		t.Error("expected version error for legacy plugin")
	}
}

func TestCartUpdateItemsSequential(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		if !containsStr(r.URL.Path, "/cart/item/") {
			t.Errorf("expected a per-item request path, got %s", r.URL.Path)
		}
		w.Write([]byte(`{"item_count":2}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Cart().UpdateItems(context.Background(), map[string]int{
		"abc": 2,
		"def": 3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 sequential requests, got %d", len(requests))
	}
	if resp.GetItemCount() != 2 {
		t.Errorf("expected the last response to be returned")
	}
}

func TestCartUpdateItemsRequiresItems(t *testing.T) {
	c := NewClient("https://example.com")
	_, err := c.Cart().UpdateItems(context.Background(), map[string]int{})
	if err == nil {
		t.Error("expected validation error for empty items")
	}
}

func TestCartRemoveItemsSequential(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		requests = append(requests, r.URL.Path)
		w.Write([]byte(`{"item_count":0}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().RemoveItems(context.Background(), []string{"abc", "def"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 sequential requests, got %d", len(requests))
	}
}

func TestCartRemoveItemsRequiresKeys(t *testing.T) {
	c := NewClient("https://example.com")
	_, err := c.Cart().RemoveItems(context.Background(), []string{})
	if err == nil {
		t.Error("expected validation error for empty item keys")
	}
}

func TestCartBatchUpdateItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsStr(r.URL.Path, "/cocart/batch") {
			t.Errorf("expected batch endpoint, got %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		reqs, ok := body["requests"].([]any)
		if !ok || len(reqs) != 1 {
			t.Fatalf("expected 1 queued request, got %v", body["requests"])
		}
		first := reqs[0].(map[string]any)
		if first["method"] != "POST" {
			t.Errorf("expected method POST, got %v", first["method"])
		}
		if first["path"] != "/cocart/v2/cart/item/abc" {
			t.Errorf("expected versioned item path, got %v", first["path"])
		}
		w.Write([]byte(`{"item_count":1}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().BatchUpdateItems(context.Background(), map[string]int{"abc": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartBatchRemoveItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		reqs := body["requests"].([]any)
		first := reqs[0].(map[string]any)
		if first["method"] != "DELETE" {
			t.Errorf("expected method DELETE, got %v", first["method"])
		}
		w.Write([]byte(`{"item_count":0}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().BatchRemoveItems(context.Background(), []string{"abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartUpdateCustomerMirrorsBillingToShipping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["namespace"] != "update-customer" {
			t.Errorf("expected namespace=update-customer, got %v", body["namespace"])
		}
		if body["first_name"] != "John" {
			t.Errorf("expected unprefixed billing field, got %v", body["first_name"])
		}
		if body["s_first_name"] != "John" {
			t.Errorf("expected billing mirrored into s_first_name, got %v", body["s_first_name"])
		}
		if _, ok := body["ship_to_different_address"]; ok {
			t.Error("ship_to_different_address should not be set when shipping is omitted")
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().UpdateCustomer(context.Background(), map[string]string{"first_name": "John"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartUpdateCustomerWithDistinctShipping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["first_name"] != "John" {
			t.Errorf("expected billing first_name=John, got %v", body["first_name"])
		}
		if body["s_first_name"] != "Jane" {
			t.Errorf("expected shipping s_first_name=Jane, got %v", body["s_first_name"])
		}
		if body["ship_to_different_address"] != true {
			t.Errorf("expected ship_to_different_address=true, got %v", body["ship_to_different_address"])
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().UpdateCustomer(context.Background(),
		map[string]string{"first_name": "John"},
		map[string]string{"first_name": "Jane"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartSetShippingMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["rate_id"] != "flat_rate:2" {
			t.Errorf("expected rate_id=flat_rate:2, got %v", body["rate_id"])
		}
		if body["package_id"] != "0" {
			t.Errorf("expected package_id=0, got %v", body["package_id"])
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().SetShippingMethod(context.Background(), "flat_rate:2", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartSetShippingMethodWithoutPackageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["package_id"]; ok {
			t.Error("package_id should be omitted when not provided")
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().SetShippingMethod(context.Background(), "flat_rate:2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCartCalculateShippingDelegatesToCalculate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsStr(r.URL.Path, "/cart/calculate") || containsStr(r.URL.Path, "/cart/calculate/shipping") {
			t.Errorf("expected /cart/calculate, got %s", r.URL.Path)
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Cart().CalculateShipping(context.Background(), map[string]string{"country": "US"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
