package cocart

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://example.com")
	if c.GetStoreURL() != "https://example.com" {
		t.Errorf("expected store URL https://example.com, got %s", c.GetStoreURL())
	}
	if c.GetRESTPrefix() != "wp-json" {
		t.Errorf("expected rest prefix wp-json, got %s", c.GetRESTPrefix())
	}
	if c.GetNamespace() != "cocart" {
		t.Errorf("expected namespace cocart, got %s", c.GetNamespace())
	}
	if c.GetMainPlugin() != MainPluginBasic {
		t.Errorf("expected main plugin basic, got %s", c.GetMainPlugin())
	}
}

func TestNewClientWithOptions(t *testing.T) {
	c := NewClient("https://example.com/",
		WithCartKey("test-key"),
		WithBasicAuth("user", "pass"),
		WithTimeout(10*time.Second),
		WithRESTPrefix("api"),
		WithNamespace("myshop"),
		WithMaxRetries(3),
		WithDebug(true),
		WithMainPlugin(MainPluginLegacy),
		WithETag(false),
	)

	if c.GetStoreURL() != "https://example.com" {
		t.Errorf("trailing slash not trimmed: %s", c.GetStoreURL())
	}
	if c.GetCartKey() != "test-key" {
		t.Errorf("cart key not set")
	}
	if !c.IsAuthenticated() {
		t.Error("should be authenticated with basic auth")
	}
	if c.GetRESTPrefix() != "api" {
		t.Errorf("rest prefix: got %s", c.GetRESTPrefix())
	}
	if c.GetNamespace() != "myshop" {
		t.Errorf("namespace: got %s", c.GetNamespace())
	}
	if c.GetMainPlugin() != MainPluginLegacy {
		t.Errorf("main plugin: got %s", c.GetMainPlugin())
	}
}

func TestClientSetters(t *testing.T) {
	c := NewClient("https://example.com")

	c.SetCartKey("key1")
	if c.GetCartKey() != "key1" {
		t.Error("SetCartKey failed")
	}

	c.SetAuth("user", "pass")
	if !c.IsAuthenticated() {
		t.Error("SetAuth failed")
	}
	if c.IsGuest() {
		t.Error("should not be guest after SetAuth")
	}

	c.SetJWTToken("token123")
	if c.GetJWTToken() != "token123" {
		t.Error("SetJWTToken failed")
	}
	if !c.HasJWTToken() {
		t.Error("HasJWTToken should return true")
	}

	c.SetRefreshToken("refresh123")
	if c.GetRefreshToken() != "refresh123" {
		t.Error("SetRefreshToken failed")
	}

	c.ClearJWTToken()
	if c.HasJWTToken() {
		t.Error("ClearJWTToken failed")
	}
}

func TestClientHTTPGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("missing Accept header")
		}
		if r.Header.Get("User-Agent") != "CoCart-Go-SDK/"+Version {
			t.Errorf("wrong User-Agent: %s", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}, "item_count": 0})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Errorf("expected success, got %d", resp.StatusCode)
	}
}

func TestClientHTTPPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type header")
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Post(context.Background(), "cart/add-item", map[string]any{"id": "42", "quantity": "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccessful() {
		t.Error("expected success")
	}
}

func TestClientBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("missing Authorization header")
		}
		if auth != "Basic dXNlcjpwYXNz" { // base64("user:pass")
			t.Errorf("wrong auth header: %s", auth)
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithBasicAuth("user", "pass"))
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientJWTAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("wrong auth header: %s", auth)
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithJWTToken("mytoken"))
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientCartKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cartKey := r.Header.Get("Cart-Key")
		if cartKey != "guest-key" {
			t.Errorf("wrong Cart-Key header: %s", cartKey)
		}
		// Check cart_key also in query params
		if r.URL.Query().Get("cart_key") != "guest-key" {
			t.Errorf("wrong cart_key query param: %s", r.URL.Query().Get("cart_key"))
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithCartKey("guest-key"))
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientCartKeyFromResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cart-Key", "server-assigned-key")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.GetCartKey() != "server-assigned-key" {
		t.Errorf("cart key not extracted from response: %s", c.GetCartKey())
	}
}

func TestClientErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		errType    string
	}{
		{"auth error", 401, `{"code":"rest_forbidden","message":"Forbidden"}`, "auth"},
		{"validation error", 400, `{"code":"cocart_invalid","message":"Invalid"}`, "validation"},
		{"general error", 500, `{"code":"server_error","message":"Server Error"}`, "general"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c := NewClient(server.URL)
			_, err := c.Get(context.Background(), "cart")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestClientETag(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("ETag", `"etag-123"`)
			w.Write([]byte(`{"data": "fresh"}`))
			return
		}
		if r.Header.Get("If-None-Match") == `"etag-123"` {
			w.WriteHeader(304)
			return
		}
		w.Write([]byte(`{"data": "fresh"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)

	// First request - should get ETag
	resp, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("first request error: %v", err)
	}
	if resp.GetETag() != `"etag-123"` {
		t.Errorf("expected ETag, got: %s", resp.GetETag())
	}

	// Second request - should send If-None-Match and get 304
	resp2, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("second request error: %v", err)
	}
	if resp2.StatusCode != 304 {
		t.Errorf("expected 304, got %d", resp2.StatusCode)
	}
}

func TestClientEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)

	requestFired := false
	responseFired := false

	c.OnRequest(func(e RequestEvent) {
		requestFired = true
		if e.Method != "GET" {
			t.Errorf("expected GET in event, got %s", e.Method)
		}
	})
	c.OnResponse(func(e ResponseEvent) {
		responseFired = true
		if e.Status != 200 {
			t.Errorf("expected 200 in event, got %d", e.Status)
		}
	})

	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !requestFired {
		t.Error("request event not fired")
	}
	if !responseFired {
		t.Error("response event not fired")
	}
}

func TestClientRequiresBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	err := c.RequiresBasic("testMethod")
	if err == nil {
		t.Error("expected error for legacy plugin")
	}
	var vErr *VersionError
	if !errorAs(err, &vErr) {
		t.Error("expected VersionError")
	}
}

func TestClientClearSession(t *testing.T) {
	c := NewClient("https://example.com",
		WithBasicAuth("user", "pass"),
		WithCartKey("key"),
	)
	err := c.ClearSession()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IsAuthenticated() {
		t.Error("should not be authenticated after clear")
	}
	if c.GetCartKey() != "" {
		t.Error("cart key should be empty after clear")
	}
}

func TestURLBuilding(t *testing.T) {
	c := NewClient("https://example.com")
	url := c.buildURL("cart", nil)
	expected := "https://example.com/wp-json/cocart/v2/cart"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestURLBuildingWithParams(t *testing.T) {
	c := NewClient("https://example.com")
	url := c.buildURL("products", map[string]string{"page": "2", "per_page": "10"})
	if url == "" {
		t.Error("URL should not be empty")
	}
	// Should contain query params
	if !contains(url, "page=2") || !contains(url, "per_page=10") {
		t.Errorf("URL missing query params: %s", url)
	}
}

func TestFieldNormalizationBasic(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginBasic))
	url := c.buildURL("cart", map[string]string{"fields": "items,totals"})
	if !contains(url, "_fields=items") {
		t.Errorf("basic plugin should normalize fields to _fields: %s", url)
	}
}

func TestFieldNormalizationLegacy(t *testing.T) {
	c := NewClient("https://example.com", WithMainPlugin(MainPluginLegacy))
	url := c.buildURL("cart", map[string]string{"_fields": "items,totals"})
	if !contains(url, "fields=items") {
		t.Errorf("legacy plugin should normalize _fields to fields: %s", url)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestClientSession(t *testing.T) {
	c := NewClient("https://example.com",
		WithStorage(NewMemoryStorage()),
	)

	// Session should be lazily created
	session := c.Session()
	if session == nil {
		t.Fatal("Session() returned nil")
	}

	// Should return the same instance
	session2 := c.Session()
	if session != session2 {
		t.Error("Session() should return the same instance")
	}

	// Should start as guest
	if !session.IsGuest() {
		t.Error("session should start as guest")
	}
	if session.IsAuthenticated() {
		t.Error("session should not be authenticated initially")
	}
}

func TestFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/cart", nil)
	req.Header.Set("X-Cart-Key", "from-request-key")

	c := FromRequest("https://example.com", req)
	if c.GetCartKey() != "from-request-key" {
		t.Errorf("expected cart key from-request-key, got %s", c.GetCartKey())
	}

	// Without X-Cart-Key header
	req2 := httptest.NewRequest("GET", "/api/cart", nil)
	c2 := FromRequest("https://example.com", req2)
	if c2.GetCartKey() != "" {
		t.Errorf("expected empty cart key, got %s", c2.GetCartKey())
	}
}

func TestClientCartKeyHeaderLegacyPlugin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cart-Key") != "" {
			t.Errorf("legacy plugin should not send Cart-Key header, got %s", r.Header.Get("Cart-Key"))
		}
		if r.Header.Get("CoCart-API-Cart-Key") != "guest-key" {
			t.Errorf("legacy plugin should send CoCart-API-Cart-Key, got %s", r.Header.Get("CoCart-API-Cart-Key"))
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithCartKey("guest-key"), WithMainPlugin(MainPluginLegacy))
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientCartKeyHeaderBasicPluginOnlySendsOneHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cart-Key") != "guest-key" {
			t.Errorf("basic plugin should send Cart-Key, got %s", r.Header.Get("Cart-Key"))
		}
		if r.Header.Get("CoCart-API-Cart-Key") != "" {
			t.Errorf("basic plugin should not send CoCart-API-Cart-Key, got %s", r.Header.Get("CoCart-API-Cart-Key"))
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithCartKey("guest-key"), WithMainPlugin(MainPluginBasic))
	_, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientETagReturnsCachedBodyOn304(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("ETag", `"etag-123"`)
			w.Header().Set("X-Custom", "fresh")
			w.Write([]byte(`{"data": "fresh"}`))
			return
		}
		if r.Header.Get("If-None-Match") == `"etag-123"` {
			w.WriteHeader(304)
			return
		}
		w.Write([]byte(`{"data": "should-not-see-this"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)

	resp1, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("first request error: %v", err)
	}
	if resp1.GetString("data", "") != "fresh" {
		t.Fatalf("expected fresh body, got %s", resp1.ToJSON())
	}

	resp2, err := c.Get(context.Background(), "cart")
	if err != nil {
		t.Fatalf("second request error: %v", err)
	}
	if resp2.StatusCode != 304 {
		t.Errorf("expected 304, got %d", resp2.StatusCode)
	}
	// The cached body/headers from the fresh 2xx GET must be returned instead
	// of the empty 304 body.
	if resp2.GetString("data", "") != "fresh" {
		t.Errorf("expected cached body on 304, got %s", resp2.ToJSON())
	}
	if resp2.GetHeader("X-Custom") != "fresh" {
		t.Errorf("expected cached headers on 304, got %s", resp2.GetHeader("X-Custom"))
	}
}

func TestGetRetryDelayJitter(t *testing.T) {
	c := NewClient("https://example.com")

	// Base exponential delay for attempt 3 is min(2^2, 30) = 4s.
	// ±20% jitter should keep the result within [3.2s, 4.8s].
	for i := 0; i < 50; i++ {
		delay := c.getRetryDelay(3, nil)
		if delay < 3200*time.Millisecond || delay > 4800*time.Millisecond {
			t.Fatalf("delay %v out of expected jitter range [3.2s, 4.8s]", delay)
		}
	}
}

func TestGetRetryDelayHonorsRetryAfterWithoutJitter(t *testing.T) {
	c := NewClient("https://example.com")
	headers := http.Header{}
	headers.Set("Retry-After", "5")
	resp := &Response{StatusCode: 429, Headers: headers}

	delay := c.getRetryDelay(1, resp)
	if delay != 5*time.Second {
		t.Errorf("expected exact Retry-After delay of 5s, got %v", delay)
	}
}

func TestClientInFlightGetDeduplication(t *testing.T) {
	var callCount int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		<-release
		w.Write([]byte(`{"item_count":1}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)

	const concurrency = 5
	var wg sync.WaitGroup
	results := make([]*Response, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = c.Get(context.Background(), "cart")
		}(i)
	}

	// Give all goroutines a chance to register as in-flight before letting
	// the (single) real request complete.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := atomic.LoadInt32(&callCount); got != 1 {
		t.Errorf("expected exactly 1 network request for concurrent identical GETs, got %d", got)
	}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d unexpected error: %v", i, err)
		}
		if results[i].GetItemCount() != 1 {
			t.Errorf("caller %d did not get the shared response", i)
		}
	}
}

func TestClientBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !containsStr(r.URL.Path, "/cocart/batch") {
			t.Errorf("expected /cocart/batch, got %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		reqs, ok := body["requests"].([]any)
		if !ok || len(reqs) != 1 {
			t.Fatalf("expected 1 request in batch body, got %v", body["requests"])
		}
		w.Write([]byte(`{"item_count":3}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	resp, err := c.Batch(context.Background(), []BatchRequestItem{
		{Method: "POST", Path: "/cocart/v2/cart/add-item", Body: map[string]any{"id": "1", "quantity": "1"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetItemCount() != 3 {
		t.Errorf("expected item_count 3, got %d", resp.GetItemCount())
	}
}

func TestClientBatchRequiresRequests(t *testing.T) {
	c := NewClient("https://example.com")
	_, err := c.Batch(context.Background(), nil)
	if err == nil {
		t.Error("expected validation error for empty requests")
	}
}

func TestClientBatchNoRouteReturnsPluginRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"code":"rest_no_route","message":"No route was found"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.Batch(context.Background(), []BatchRequestItem{{Method: "POST", Path: "/cocart/v2/cart/clear"}})
	if err == nil {
		t.Fatal("expected error")
	}
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) {
		t.Fatalf("expected *CoCartError, got %T", err)
	}
	if cocartErr.ErrorCode != "cocart_plugin_required" {
		t.Errorf("expected cocart_plugin_required, got %s", cocartErr.ErrorCode)
	}
}

// errorAs is a test helper wrapping errors.As to avoid import in test.
func errorAs(err error, target any) bool {
	return err != nil && func() bool {
		switch t := target.(type) {
		case **VersionError:
			var ve *VersionError
			if e, ok := err.(*VersionError); ok {
				*t = e
				return true
			}
			_ = ve
		}
		return false
	}()
}
