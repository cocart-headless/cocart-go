package cocart

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
)

// recordedRequest is a request seen by a recordingServer.
type recordedRequest struct {
	Method  string
	Path    string // escaped, as sent
	Query   url.Values
	Headers http.Header
	Body    string
}

// recordingServer answers every request with the queued responses (then
// {}) and records what it received.
type recordingServer struct {
	*httptest.Server
	mu        sync.Mutex
	requests  []recordedRequest
	responses []queuedResponse
}

type queuedResponse struct {
	status  int
	headers map[string]string
	body    string
}

func newRecordingServer(t *testing.T) *recordingServer {
	t.Helper()
	s := &recordingServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{
			Method:  r.Method,
			Path:    r.URL.EscapedPath(),
			Query:   r.URL.Query(),
			Headers: r.Header.Clone(),
			Body:    string(body),
		})
		resp := queuedResponse{status: 200, body: `{}`}
		if len(s.responses) > 0 {
			resp = s.responses[0]
			s.responses = s.responses[1:]
		}
		s.mu.Unlock()
		for k, v := range resp.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.status)
		w.Write([]byte(resp.body))
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *recordingServer) queue(status int, body string, headers ...map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := queuedResponse{status: status, body: body}
	if len(headers) > 0 {
		r.headers = headers[0]
	}
	s.responses = append(s.responses, r)
}

func (s *recordingServer) last(t *testing.T) recordedRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		t.Fatal("no request was sent")
	}
	return s.requests[len(s.requests)-1]
}

func (s *recordingServer) all() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]recordedRequest(nil), s.requests...)
}

func jsonBody(t *testing.T, r recordedRequest) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
		t.Fatalf("body %q: %v", r.Body, err)
	}
	return m
}

// --- Cart ---

func TestCartApplyCouponWithEmail(t *testing.T) {
	s := newRecordingServer(t)
	NewClient(s.URL).Cart().ApplyCoupon(context.Background(), "VIP", "jane@example.com")
	req := s.last(t)
	if req.Path != "/wp-json/cocart/v2/cart/apply-coupon" {
		t.Errorf("path = %s", req.Path)
	}
	if got := jsonBody(t, req); !reflect.DeepEqual(got, map[string]any{"code": "VIP", "email": "jane@example.com"}) {
		t.Errorf("body = %v", got)
	}
}

func TestCartRemoveCouponEncodesTheCode(t *testing.T) {
	s := newRecordingServer(t)
	c := NewClient(s.URL)
	for _, code := range []string{"summer sale", "10%off", "a#b?c", "café"} {
		c.Cart().RemoveCoupon(context.Background(), code)
	}
	var paths []string
	for _, r := range s.all() {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		paths = append(paths, r.Path)
	}
	want := []string{
		"/wp-json/cocart/v2/cart/coupons/summer%20sale",
		"/wp-json/cocart/v2/cart/coupons/10%25off",
		"/wp-json/cocart/v2/cart/coupons/a%23b%3Fc",
		"/wp-json/cocart/v2/cart/coupons/caf%C3%A9",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
}

func TestCartPlusRoutes(t *testing.T) {
	s := newRecordingServer(t)
	cart := NewClient(s.URL).Cart()
	ctx := context.Background()
	cart.RemoveCoupons(ctx)
	cart.CheckCoupons(ctx)
	cart.CheckCoupons(ctx, "jane@example.com")
	cart.GetMini(ctx, map[string]string{"_fields": "items"})
	cart.RemoveAllItems(ctx)

	reqs := s.all()
	type call struct{ method, path, query string }
	var got []call
	for _, r := range reqs {
		got = append(got, call{r.Method, r.Path, r.Query.Encode()})
	}
	want := []call{
		{"DELETE", "/wp-json/cocart/v2/cart/coupons", ""},
		{"GET", "/wp-json/cocart/v2/cart/coupons/validate", ""},
		{"GET", "/wp-json/cocart/v2/cart/coupons/validate", "email=jane%40example.com"},
		{"GET", "/wp-json/cocart/v2/cart/mini", "_fields=items"},
		{"DELETE", "/wp-json/cocart/v2/cart/items", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v\nwant %v", got, want)
	}
}

func TestCartAddFeeSendsTheAmountAsAString(t *testing.T) {
	s := newRecordingServer(t)
	cart := NewClient(s.URL).Cart()
	cart.AddFee(context.Background(), "Rush", 5.99, true)
	if got := jsonBody(t, s.last(t)); !reflect.DeepEqual(got, map[string]any{"name": "Rush", "amount": "5.99", "taxable": true}) {
		t.Errorf("body = %v", got)
	}
	cart.AddFee(context.Background(), "Gift wrap", 2.5, true, "reduced-rate")
	if got := jsonBody(t, s.last(t)); got["amount"] != "2.5" || got["tax_class"] != "reduced-rate" {
		t.Errorf("body = %v", got)
	}
	if s.last(t).Path != "/wp-json/cocart/v2/cart/add-fee" {
		t.Errorf("path = %s", s.last(t).Path)
	}
}

func TestCartAddItemSendsPluginChoices(t *testing.T) {
	s := newRecordingServer(t)
	cart := NewClient(s.URL).Cart()
	ctx := context.Background()
	cart.AddItem(ctx, 306, 1, map[string]any{"convert_to_sub": "1_month"})
	cart.AddItem(ctx, 302, 1, map[string]any{"recipient_email": "friend@example.com"})
	cart.AddItem(ctx, 154, 1, map[string]any{"item_data": map[string]any{"warranty": "1-year"}})
	reqs := s.all()
	if jsonBody(t, reqs[0])["convert_to_sub"] != "1_month" ||
		jsonBody(t, reqs[1])["recipient_email"] != "friend@example.com" ||
		!reflect.DeepEqual(jsonBody(t, reqs[2])["item_data"], map[string]any{"warranty": "1-year"}) {
		t.Errorf("bodies = %v", reqs)
	}
}

func TestCartChangePlan(t *testing.T) {
	s := newRecordingServer(t)
	cart := NewClient(s.URL).Cart()
	if _, err := cart.ChangePlan(context.Background(), "abc123", "1_month"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := s.last(t)
	if req.Method != http.MethodPost || req.Path != "/wp-json/cocart/v2/cart/update" {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	want := map[string]any{"namespace": "update-item-schema", "item_key": "abc123", "convert_to_sub": "1_month"}
	if got := jsonBody(t, req); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v, want %v", got, want)
	}

	cart.ChangePlan(context.Background(), "abc123", "0")
	if jsonBody(t, s.last(t))["convert_to_sub"] != "0" {
		t.Error("one-time should send convert_to_sub 0")
	}

	before := len(s.all())
	_, err := cart.ChangePlan(context.Background(), "", "0")
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.ErrorCode != "cocart_missing_item_key" {
		t.Errorf("err = %v, want a ValidationError", err)
	}
	if len(s.all()) != before {
		t.Error("no request should be sent without an item key")
	}
}

// --- Products ---

func TestProductsShippingRatesWithoutTheCartKey(t *testing.T) {
	// The route prices a simulated cart and refuses a cart key (400).
	s := newRecordingServer(t)
	c := NewClient(s.URL, WithCartKey("t_guest_key"))
	if _, err := c.Products().ShippingRates(context.Background(), 154, ProductShippingRatesParams{Quantity: 2}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := s.last(t)
	if req.Method != http.MethodGet || req.Path != "/wp-json/cocart/v2/products/154/shipping-rates" {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	if req.Query.Encode() != "quantity=2" {
		t.Errorf("query = %s", req.Query.Encode())
	}
	if req.Headers.Get("Cart-Key") != "" {
		t.Error("the cart key header should not be sent")
	}

	// Other requests keep the cart key.
	c.Products().Find(context.Background(), 154)
	other := s.last(t)
	if other.Query.Get("cart_key") != "t_guest_key" || other.Headers.Get("Cart-Key") != "t_guest_key" {
		t.Errorf("other request lost the cart key: %v %v", other.Query, other.Headers)
	}
}

func TestProductsShippingRatesVariationItemDataAndSKU(t *testing.T) {
	s := newRecordingServer(t)
	NewClient(s.URL, WithNamespace("mystore")).Products().ShippingRates(context.Background(), "HOODIE/BLUE", ProductShippingRatesParams{
		Variation: map[string]string{"attribute_pa_color": "blue"},
		ItemData:  map[string]string{"engraving": "Hi"},
	})
	req := s.last(t)
	if req.Path != "/wp-json/mystore/v2/products/HOODIE%2FBLUE/shipping-rates" {
		t.Errorf("path = %s", req.Path)
	}
	want := url.Values{"variation[attribute_pa_color]": {"blue"}, "item_data[engraving]": {"Hi"}}
	if !reflect.DeepEqual(req.Query, want) {
		t.Errorf("query = %v, want %v", req.Query, want)
	}
}

func TestProductsCreateReview(t *testing.T) {
	s := newRecordingServer(t)
	NewClient(s.URL).Products().CreateReview(context.Background(), ProductReviewInput{
		ProductID: 42, Review: "Great", Rating: 5, Reviewer: "Jane", ReviewerEmail: "jane@example.com",
	})
	req := s.last(t)
	if req.Method != http.MethodPost || req.Path != "/wp-json/cocart/v2/products/reviews" {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	want := map[string]any{"product_id": float64(42), "review": "Great", "rating": float64(5), "reviewer": "Jane", "reviewer_email": "jane@example.com"}
	if got := jsonBody(t, req); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v", got)
	}
}

func TestProductsSortByPriceAsc(t *testing.T) {
	s := newRecordingServer(t)
	NewClient(s.URL).Products().SortBy(context.Background(), OrderByPriceDesc, SortDesc)
	if q := s.last(t).Query; q.Get("orderby") != "price_desc" || q.Get("order") != "desc" {
		t.Errorf("query = %v", q)
	}
}

// --- Account ---

func TestAccountPlusRoutes(t *testing.T) {
	s := newRecordingServer(t)
	account := NewClient(s.URL, WithBasicAuth("user", "pass")).Account()
	ctx := context.Background()
	account.UpdateAddress(ctx, map[string]any{"billing_address": map[string]any{"city": "Paris"}})
	account.CancelOrder(ctx, 42)
	account.VerifyEmail(ctx, "k3y", 7)
	account.SendOrderEmailVerification(ctx)
	account.VerifyOrderEmail(ctx, "k3y", 7)
	account.GetSubscriptions(ctx, map[string]any{"page": 2})
	account.GetSubscription(ctx, 9)
	account.GetSubscription(ctx, 9, "guest@example.com")
	account.SuspendSubscription(ctx, 9)
	account.ReactivateSubscription(ctx, 9)
	account.CancelSubscription(ctx, 9)
	account.SetSubscriptionAutoRenew(ctx, 9, false)
	account.UpdateSubscriptionPaymentMethod(ctx, 9, map[string]any{"payment_method": "stripe", "token_id": 3})
	account.Resubscribe(ctx, 9)
	account.CancelResubscribe(ctx, 9)

	base := "/wp-json/cocart/v2/my-account"
	type call struct{ method, path, query, body string }
	want := []call{
		{"POST", base + "/address", "", `{"billing_address":{"city":"Paris"}}`},
		{"POST", base + "/orders/42/cancel", "", ""},
		{"GET", base + "/verify-email", "key=k3y&user_id=7", ""},
		{"POST", base + "/verify-order-email", "", ""},
		{"GET", base + "/verify-order-email", "key=k3y&user_id=7", ""},
		{"GET", base + "/subscriptions", "page=2", ""},
		{"GET", base + "/subscriptions/9", "", ""},
		{"GET", base + "/subscriptions/9", "email=guest%40example.com", ""},
		{"POST", base + "/subscriptions/9/suspend", "", ""},
		{"POST", base + "/subscriptions/9/reactivate", "", ""},
		{"POST", base + "/subscriptions/9/cancel", "", ""},
		{"POST", base + "/subscriptions/9/auto-renew", "", `{"enabled":false}`},
		{"POST", base + "/subscriptions/9/payment-method", "", `{"payment_method":"stripe","token_id":3}`},
		{"POST", base + "/subscriptions/9/resubscribe", "", ""},
		{"DELETE", base + "/subscriptions/9/resubscribe", "", ""},
	}
	var got []call
	for _, r := range s.all() {
		got = append(got, call{r.Method, r.Path, r.Query.Encode(), r.Body})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("calls =\n%v\nwant\n%v", got, want)
	}
}

func TestAccountRegister(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(201, `{"user_id":42,"username":"jane","message":"Registration complete."}`)
	c := NewClient(s.URL)
	resp, err := c.Account().Register(context.Background(), RegisterCustomerInput{
		Email: "jane@example.com", Username: "jane", Password: "a-strong-password",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := s.last(t)
	if req.Method != http.MethodPost || req.Path != "/wp-json/cocart/v2/register" {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	// Not username/password: CoCart would treat those as a Basic Auth attempt.
	want := map[string]any{"email": "jane@example.com", "requested_username": "jane", "requested_password": "a-strong-password"}
	if got := jsonBody(t, req); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %v", got)
	}
	if req.Headers.Get("Authorization") != "" {
		t.Error("no Authorization header expected")
	}
	if resp.GetInt("user_id", 0) != 42 {
		t.Errorf("user_id = %d", resp.GetInt("user_id", 0))
	}

	c.Account().Register(context.Background(), RegisterCustomerInput{Email: "jane@example.com"})
	if got := jsonBody(t, s.last(t)); !reflect.DeepEqual(got, map[string]any{"email": "jane@example.com"}) {
		t.Errorf("email-only body = %v", got)
	}
}

func TestAccountRequestOrderWithdrawal(t *testing.T) {
	s := newRecordingServer(t)
	data := map[string]any{
		"first_name":         "Jane",
		"last_name":          "Doe",
		"email":              "jane@example.com",
		"email_confirmation": "jane@example.com",
		"order_number":       "1234",
	}
	NewClient(s.URL).Account().RequestOrderWithdrawal(context.Background(), data)
	req := s.last(t)
	if req.Method != http.MethodPost || req.Path != "/wp-json/cocart/v2/order-withdrawal" {
		t.Errorf("request = %s %s", req.Method, req.Path)
	}
	if got := jsonBody(t, req); !reflect.DeepEqual(got, data) {
		t.Errorf("body = %v", got)
	}
}
