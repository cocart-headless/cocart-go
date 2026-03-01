package cocart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProductsAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("X-WP-Total", "50")
		w.Header().Set("X-WP-TotalPages", "5")
		w.Write([]byte(`[{"id":1,"name":"Widget"}]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Products().All(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Error("expected success")
	}
	if resp.GetTotalResults() != 50 {
		t.Errorf("total results = %d, want 50", resp.GetTotalResults())
	}
}

func TestProductsFind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsStr(r.URL.Path, "/products/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":42,"name":"Widget"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Products().Find(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	product, _ := Unmarshal[Product](resp)
	if product.ID != 42 {
		t.Errorf("product ID = %d, want 42", product.ID)
	}
}

func TestProductsFindBySlugRequiresBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	_, err := c.Products().FindBySlug(context.Background(), "widget")
	if err == nil {
		t.Error("expected version error")
	}
}

func TestProductsSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "widget" {
			t.Errorf("expected search=widget, got %s", r.URL.Query().Get("search"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().Search(context.Background(), "widget")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsByCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") != "electronics" {
			t.Errorf("expected category=electronics, got %s", r.URL.Query().Get("category"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().ByCategory(context.Background(), "electronics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsFeatured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("featured") != "true" {
			t.Errorf("expected featured=true, got %s", r.URL.Query().Get("featured"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().Featured(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsOnSale(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("on_sale") != "true" {
			t.Errorf("expected on_sale=true, got %s", r.URL.Query().Get("on_sale"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().OnSale(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsAllPaginated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-WP-TotalPages", "2")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	pages, err := c.Products().AllPaginated().Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(pages))
	}
}

func TestProductsByPriceRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("min_price") != "10" {
			t.Errorf("expected min_price=10, got %s", r.URL.Query().Get("min_price"))
		}
		if r.URL.Query().Get("max_price") != "50" {
			t.Errorf("expected max_price=50, got %s", r.URL.Query().Get("max_price"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().ByPriceRange(context.Background(), "10", "50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsByPriceRangePartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("min_price") != "" {
			t.Errorf("expected no min_price, got %s", r.URL.Query().Get("min_price"))
		}
		if r.URL.Query().Get("max_price") != "25" {
			t.Errorf("expected max_price=25, got %s", r.URL.Query().Get("max_price"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Products().ByPriceRange(context.Background(), "", "25")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductsBrandsRequiresBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	_, err := c.Products().Brands(context.Background())
	if err == nil {
		t.Error("expected version error")
	}
}
