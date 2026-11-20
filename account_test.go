package cocart

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestAccountChangePasswordPostsOnlyThePasswordFields(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(200, `{"object":"customer","result":{"code":"account_updated"}}`)
	c := NewClient(s.URL, WithBasicAuth("user", "pass"))
	if _, err := c.Account().ChangePassword(context.Background(), "old-password", "new-password", "new-password"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	requests := s.all()
	if len(requests) != 1 {
		t.Fatalf("sent %d requests, want 1", len(requests))
	}
	if requests[0].Method != http.MethodPost || requests[0].Path != "/wp-json/cocart/v2/my-account" {
		t.Errorf("request = %s %s", requests[0].Method, requests[0].Path)
	}
	want := map[string]any{"password_current": "old-password", "password_1": "new-password", "password_2": "new-password"}
	if got := jsonBody(t, requests[0]); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v, want %v", got, want)
	}
}

func TestAccountGetOrders(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]any
		query  string
	}{
		{"no filters", nil, ""},
		{"pagination", map[string]any{"page": 2, "per_page": 20, "order": "ASC"}, "order=ASC&page=2&per_page=20"},
		{
			"filters, joining a list of statuses and meta keys with commas",
			map[string]any{
				"status":                  []string{"processing", "on-hold"},
				"after":                   "2026-01-01T00:00:00",
				"before":                  "2026-12-31T23:59:59",
				"subscription_order_type": "renewal",
				"formatted":               true,
				"meta_keys":               []string{"gift_message", "_configuration"},
			},
			"after=2026-01-01T00%3A00%3A00&before=2026-12-31T23%3A59%3A59&formatted=true&meta_keys=gift_message%2C_configuration&status=processing%2Con-hold&subscription_order_type=renewal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newRecordingServer(t)
			c := NewClient(s.URL, WithBasicAuth("user", "pass"))
			if _, err := c.Account().GetOrders(context.Background(), tt.params); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req := s.last(t)
			if req.Path != "/wp-json/cocart/v2/my-account/orders" {
				t.Errorf("path = %s", req.Path)
			}
			if got := req.Query.Encode(); got != tt.query {
				t.Errorf("query = %s, want %s", got, tt.query)
			}
		})
	}
}

func TestAccountDocumentParams(t *testing.T) {
	s := newRecordingServer(t)
	c := NewClient(s.URL, WithBasicAuth("user", "pass"))
	ctx := context.Background()
	params := map[string]any{"formatted": false, "meta_keys": []string{"gift_message"}}
	c.Account().GetProfile(ctx, params)
	c.Account().GetOrder(ctx, 42, params)
	c.Account().GetGuestOrder(ctx, 42, "jane@example.com", params)
	c.Account().GetSubscriptions(ctx, map[string]any{"status": []string{"active", "on-hold"}, "formatted": true})

	want := []string{
		"formatted=false&meta_keys=gift_message",
		"formatted=false&meta_keys=gift_message",
		"email=jane%40example.com&formatted=false&meta_keys=gift_message",
		"formatted=true&status=active%2Con-hold",
	}
	var got []string
	for _, r := range s.all() {
		got = append(got, r.Query.Encode())
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("queries =\n%v\nwant\n%v", got, want)
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

func TestAccountOrderDownloads(t *testing.T) {
	tests := []struct {
		name  string
		call  func(ctx context.Context, a *AccountEndpoint) (*Response, error)
		query string
	}{
		{"customer", func(ctx context.Context, a *AccountEndpoint) (*Response, error) { return a.GetOrderDownloads(ctx, 123) }, ""},
		{"guest", func(ctx context.Context, a *AccountEndpoint) (*Response, error) {
			return a.GetGuestOrderDownloads(ctx, 123, "jane@example.com")
		}, "email=jane%40example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newRecordingServer(t)
			s.queue(200, `{"downloads":[{"download_id":"abc","download_name":"Guide"}]}`)
			resp, err := tt.call(context.Background(), NewClient(s.URL).Account())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req := s.last(t)
			if req.Method != http.MethodGet || req.Path != "/wp-json/cocart/v2/my-account/orders/123/downloads" {
				t.Errorf("request = %s %s", req.Method, req.Path)
			}
			if got := req.Query.Encode(); got != tt.query {
				t.Errorf("query = %s, want %s", got, tt.query)
			}
			// Returned as sent.
			if string(resp.Body) != `{"downloads":[{"download_id":"abc","download_name":"Guide"}]}` {
				t.Errorf("body = %s", resp.Body)
			}
		})
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

func TestAccountGetReviewsUsesProductsReviewsMine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wp-json/cocart/v2/products/reviews/mine" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "5" {
			t.Errorf("per_page = %q, want 5", r.URL.Query().Get("per_page"))
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Account().GetReviews(context.Background(), &PaginationParams{PerPage: 5})
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
