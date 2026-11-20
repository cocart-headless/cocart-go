package cocart

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Client hardening for CoCart Plus 2.0.

func TestGetRetryDelayReadsRateLimitRetryAfterFirst(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    time.Duration
	}{
		{"RateLimit-Retry-After, as CoCart Plus sends", map[string]string{"RateLimit-Retry-After": "7"}, 7 * time.Second},
		{"Retry-After, when that is all a proxy sends", map[string]string{"Retry-After": "3"}, 3 * time.Second},
		{"RateLimit-Retry-After wins over Retry-After", map[string]string{"RateLimit-Retry-After": "7", "Retry-After": "3"}, 7 * time.Second},
		{"capped at 60 seconds", map[string]string{"RateLimit-Retry-After": "120"}, 60 * time.Second},
	}
	c := NewClient("https://store.example.com")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}
			if got := c.getRetryDelay(1, &Response{StatusCode: 429, Headers: headers}); got != tt.want {
				t.Errorf("delay = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestA429IsRetriedAfterRateLimitRetryAfter(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(429, `{"code":"cocart_rate_limited","message":"Slow down"}`, map[string]string{"RateLimit-Retry-After": "0"})
	s.queue(200, `{"items":[]}`)
	c := NewClient(s.URL, WithMaxRetries(1))
	var delays []time.Duration
	c.OnRetry(func(e RetryEvent) { delays = append(delays, e.Delay) })

	if _, err := c.Cart().Get(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(delays, []time.Duration{0}) {
		t.Errorf("delays = %v, want [0s]", delays)
	}
}

func TestDebugOutputHidesEveryQueryValue(t *testing.T) {
	s := newRecordingServer(t)
	var out bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&out)
	c := NewClient(s.URL, WithDebug(true))
	ctx := context.Background()

	c.Get(ctx, "cart", map[string]string{"order_key": "wc_order_secret", "email": "jane@example.com"})
	c.RequestRaw(ctx, http.MethodGet, "cocart/v2/order-received/12", map[string]string{"order_key": "wc_order_secret", "email": "jane@example.com"}, nil)

	output := out.String()
	for _, want := range []string{"order_key=***", "email=***", "/wp-json/cocart/v2/order-received/12"} {
		if !strings.Contains(output, want) {
			t.Errorf("debug output lacks %q:\n%s", want, output)
		}
	}
	for _, secret := range []string{"wc_order_secret", "jane@example.com", "jane%40example.com"} {
		if strings.Contains(output, secret) {
			t.Errorf("debug output shows %q:\n%s", secret, output)
		}
	}
}

func TestDebugOutputIsOffByDefault(t *testing.T) {
	s := newRecordingServer(t)
	var out bytes.Buffer
	defer log.SetOutput(log.Writer())
	log.SetOutput(&out)
	NewClient(s.URL).Get(context.Background(), "cart")
	if out.Len() != 0 {
		t.Errorf("debug output = %q, want none", out.String())
	}
}

func TestRedactQuery(t *testing.T) {
	tests := map[string]string{
		"https://store.example.com/wp-json/cocart/v2/cart":                               "https://store.example.com/wp-json/cocart/v2/cart",
		"https://store.example.com/wp-json/cocart/v2/cart?cart_key=abc&thumb=true":       "https://store.example.com/wp-json/cocart/v2/cart?cart_key=***&thumb=***",
		"https://store.example.com/wp-json/cocart/v2/my-account/verify-email?key=k&flag": "https://store.example.com/wp-json/cocart/v2/my-account/verify-email?key=***&flag",
	}
	for in, want := range tests {
		if got := redactQuery(in); got != want {
			t.Errorf("redactQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestETagCacheSkipsResponsesThatMustNotBeStored(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
		wantIfMatch  string
	}{
		{"no-store, private", "no-cache, no-store, must-revalidate, max-age=0, private", ""},
		{"private", "private, max-age=0", ""},
		{"no-store", "no-store", ""},
		{"may be stored", "", `"abc"`},
		{"max-age only", "max-age=60", `"abc"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newRecordingServer(t)
			headers := map[string]string{"ETag": `"abc"`}
			if tt.cacheControl != "" {
				headers["Cache-Control"] = tt.cacheControl
			}
			s.queue(200, `{"items":[]}`, headers)
			s.queue(200, `{"items":[]}`, headers)
			c := NewClient(s.URL)
			c.Get(context.Background(), "cart")
			c.Get(context.Background(), "cart")

			if got := s.last(t).Headers.Get("If-None-Match"); got != tt.wantIfMatch {
				t.Errorf("If-None-Match = %q, want %q", got, tt.wantIfMatch)
			}
		})
	}
}

func TestRequestRawEmitsRequestAndResponseEventsWithTheResponse(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(200, `{"object":"order","result":{"code":"order_placed"}}`)
	c := NewClient(s.URL)
	var seen []string
	var code any
	c.OnRequest(func(e RequestEvent) { seen = append(seen, "request "+e.Method) })
	c.OnResponse(func(e ResponseEvent) {
		seen = append(seen, fmt.Sprintf("response %d", e.Status))
		if e.Response != nil {
			code = e.Response.Get("result.code")
		}
	})

	if _, err := c.RequestRaw(context.Background(), http.MethodPost, "cocart/v2/checkout", nil, map[string]any{"payment_method": "cod"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(seen, []string{"request POST", "response 200"}) {
		t.Errorf("events = %v", seen)
	}
	if code != "order_placed" {
		t.Errorf("response event result.code = %v", code)
	}
}

func TestRequestRawErrorNamesTheRouteAndEmitsAnErrorEvent(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, `{"code":"cocart_order_not_found","message":"Not found.","data":{"status":404}}`)
	c := NewClient(s.URL)
	var events []string
	c.OnError(func(e ErrorEvent) { events = append(events, e.Err.Error()) })

	_, err := c.RequestRaw(context.Background(), http.MethodGet, "cocart/v2/order-received/12", map[string]string{"order_key": "wc_order_secret"}, nil)

	want := "GET /cocart/v2/order-received/12: Not found. [cocart_order_not_found]"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if !reflect.DeepEqual(events, []string{want}) {
		t.Errorf("error events = %v", events)
	}
}

func TestRequestRawIsNeverRetried(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(429, `{"code":"cocart_rate_limited","message":"Slow down"}`, map[string]string{"RateLimit-Retry-After": "0"})
	c := NewClient(s.URL, WithMaxRetries(3))
	c.RequestRaw(context.Background(), http.MethodPost, "cocart/v2/checkout", nil, map[string]any{"payment_method": "cod"})
	if n := len(s.all()); n != 1 {
		t.Errorf("sent %d times, want 1", n)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		status     int
		code       string
		validation bool
		auth       bool
	}{
		// A refused state, a rate limit or the server's fault is never a
		// validation error, whatever its code says.
		{409, "cocart_order_invalid_status", false, false},
		{429, "cocart_invalid_rate", false, false},
		{500, "cocart_missing_gateway", false, false},
		{503, "cocart_invalid_upstream", false, false},
		{400, "cocart_field_required", true, false},
		{404, "cocart_invalid_product", true, false},
		{401, "cocart_rest_authentication_required", false, true},
		{403, "cocart_rest_forbidden", false, true},
		{404, "cocart_order_not_found", false, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d %s", tt.status, tt.code), func(t *testing.T) {
			s := newRecordingServer(t)
			s.queue(tt.status, fmt.Sprintf(`{"code":%q,"message":"Nope.","data":{"status":%d}}`, tt.code, tt.status))
			_, err := NewClient(s.URL).RequestRaw(context.Background(), http.MethodPost, "cocart/v2/order-received/12/pay", nil, nil)

			var base *CoCartError
			if !errors.As(err, &base) || base.HTTPCode != tt.status || base.ErrorCode != tt.code {
				t.Fatalf("err = %#v", err)
			}
			var validation *ValidationError
			if errors.As(err, &validation) != tt.validation {
				t.Errorf("validation error = %v, want %v", !tt.validation, tt.validation)
			}
			var auth *AuthenticationError
			if errors.As(err, &auth) != tt.auth {
				t.Errorf("authentication error = %v, want %v", !tt.auth, tt.auth)
			}
		})
	}
}

func TestErrorFieldsFieldAndReason(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(400, `{"code":"cocart_invalid_address_fields","message":"Check the address.","data":{"status":400,"field":"billing.postcode","fields":{"billing.postcode":"Enter a valid postcode.","billing.phone":"Enter a phone number."},"reason":"postcode_format"}}`)
	_, err := NewClient(s.URL).RequestRaw(context.Background(), http.MethodPut, "cocart/v2/checkout", nil, map[string]any{})

	var apiErr *CoCartError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %#v", err)
	}
	wantFields := map[string]string{"billing.postcode": "Enter a valid postcode.", "billing.phone": "Enter a phone number."}
	if got := apiErr.Fields(); !reflect.DeepEqual(got, wantFields) {
		t.Errorf("Fields() = %v", got)
	}
	if got := apiErr.Field(); got != "billing.postcode" {
		t.Errorf("Field() = %q", got)
	}
	if got := apiErr.Reason(); got != "postcode_format" {
		t.Errorf("Reason() = %q", got)
	}
}

func TestErrorWithoutFieldDetails(t *testing.T) {
	err := NewCoCartError("Nope", 404, "cocart_order_not_found", map[string]any{"code": "cocart_order_not_found", "data": map[string]any{"status": 404}})
	if got := err.Fields(); got == nil || len(got) != 0 {
		t.Errorf("Fields() = %#v, want an empty map", got)
	}
	if err.Field() != "" || err.Reason() != "" {
		t.Errorf("Field() = %q, Reason() = %q, want empty", err.Field(), err.Reason())
	}
	if got := NewCoCartError("Nope", 0, "network_error").Fields(); got == nil || len(got) != 0 {
		t.Errorf("Fields() without response data = %#v, want an empty map", got)
	}
}
