package cocart

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountGetProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"user":{"id":1,"email":"test@example.com"}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Account().GetProfile(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Error("expected success")
	}
}

func TestAccountUpdateProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"user":{"email":"new@example.com"}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().UpdateProfile(context.Background(), map[string]any{
		"account_email": "new@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountChangePasswordRemapsFields(t *testing.T) {
	var body map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/change-password") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&body)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	c.Account().ChangePassword(context.Background(), "old", "newpass", "newpass")

	if body["password_current"] != "old" {
		t.Errorf("password_current = %q, want %q", body["password_current"], "old")
	}
	if body["password_1"] != "newpass" {
		t.Errorf("password_1 = %q, want %q", body["password_1"], "newpass")
	}
	if body["password_2"] != "newpass" {
		t.Errorf("password_2 = %q, want %q", body["password_2"], "newpass")
	}
}

func TestAccountGetOrders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/orders") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "5" {
			t.Errorf("expected per_page=5, got %s", r.URL.Query().Get("per_page"))
		}
		w.Write([]byte(`{"orders":[]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetOrders(context.Background(), map[string]string{"per_page": "5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/orders/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"order_id":42}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetOrder(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetGuestOrderSendsEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/orders/7") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("email") != "guest@example.com" {
			t.Errorf("expected email param, got %s", r.URL.Query().Get("email"))
		}
		w.Write([]byte(`{"order_id":7}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetGuestOrder(context.Background(), 7, "guest@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetOrderDownloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/orders/3/downloads") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetOrderDownloads(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetGuestOrderDownloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/orders/3/downloads") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("email") != "g@x.com" {
			t.Errorf("expected email param")
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetGuestOrderDownloads(context.Background(), 3, "g@x.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetDownloads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/downloads") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetDownloads(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountGetReviews(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/cocart/v2/my-account/reviews") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetReviews(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAccountNoRouteBecomesPluginRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"code":"rest_no_route","message":"No route was found matching the URL and request method.","data":{"status":404}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetProfile(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	cocartErr, ok := err.(*CoCartError)
	if !ok {
		t.Fatalf("expected *CoCartError, got %T", err)
	}
	if cocartErr.ErrorCode != "cocart_plugin_required" {
		t.Errorf("error code = %q, want %q", cocartErr.ErrorCode, "cocart_plugin_required")
	}
}
