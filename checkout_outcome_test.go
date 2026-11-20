package cocart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// bodyTransport answers each request from a queue (a status code, or 0 for a
// timeout) and keeps the JSON bodies it was sent.
type bodyTransport struct {
	answers []int
	sent    []map[string]any
}

func (b *bodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body map[string]any
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		json.Unmarshal(raw, &body)
	}
	b.sent = append(b.sent, body)
	answer := b.answers[0]
	b.answers = b.answers[1:]
	if answer == 0 {
		return nil, timeoutError{}
	}
	reply := `{"object":"order","result":{"code":"order_placed","message":"Thank you. Your order has been received."},"id":217,"key":"wc_order_abc"}`
	switch {
	case answer == http.StatusGatewayTimeout:
		reply = `<html>Gateway Time-out</html>`
	case answer == http.StatusConflict:
		reply = `{"code":"cocart_payment_in_progress","message":"This order is already being placed.","data":{"status":409}}`
	case answer >= 400:
		reply = `{"code":"cocart_cant_checkout_empty_cart","message":"Cannot checkout with empty cart.","data":{"status":400}}`
	}
	return &http.Response{
		StatusCode: answer,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(reply)),
	}, nil
}

func outcomeClient(answers ...int) (*Client, *bodyTransport) {
	transport := &bodyTransport{answers: answers}
	return NewClient("https://store.example.com", WithHTTPClient(&http.Client{Transport: transport})), transport
}

func outcomeOrder() map[string]any {
	return map[string]any{"payment_method": "cod", "billing_address": map[string]any{"email": "jane@example.com"}}
}

func TestPlaceOrderSendsANewRandomCheckoutID(t *testing.T) {
	c, transport := outcomeClient(200, 200)
	ctx := context.Background()
	if _, err := c.Checkout().PlaceOrder(ctx, outcomeOrder()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Checkout().PlaceOrder(ctx, outcomeOrder()); err != nil {
		t.Fatal(err)
	}

	pattern := regexp.MustCompile(`^chk_[0-9a-f]{32}$`)
	first, _ := transport.sent[0]["checkout_id"].(string)
	second, _ := transport.sent[1]["checkout_id"].(string)
	if !pattern.MatchString(first) || !pattern.MatchString(second) {
		t.Errorf("checkout IDs %q, %q", first, second)
	}
	if first == second {
		t.Error("each order needs its own checkout ID")
	}
}

func TestPlaceOrderSendsTheCheckoutIDGiven(t *testing.T) {
	c, transport := outcomeClient(200)
	input := outcomeOrder()
	input["checkout_id"] = "chk_mine_00000000001"
	if _, err := c.Checkout().PlaceOrder(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if got := transport.sent[0]["checkout_id"]; got != "chk_mine_00000000001" {
		t.Errorf("checkout_id = %v", got)
	}
	if _, changed := outcomeOrder()["checkout_id"]; changed {
		t.Error("the input must not change")
	}
}

func TestPlaceOrderWithNoAnswerSaysTheOutcomeIsUnknown(t *testing.T) {
	for name, answer := range map[string]int{"timeout": 0, "proxy gave up": http.StatusGatewayTimeout} {
		t.Run(name, func(t *testing.T) {
			c, transport := outcomeClient(answer)
			_, err := c.Checkout().PlaceOrder(context.Background(), outcomeOrder())

			var unknown *CheckoutOutcomeUnknownError
			if !errors.As(err, &unknown) {
				t.Fatalf("err = %#v, want *CheckoutOutcomeUnknownError", err)
			}
			if unknown.CheckoutID != transport.sent[0]["checkout_id"] {
				t.Errorf("CheckoutID = %q, sent %v", unknown.CheckoutID, transport.sent[0]["checkout_id"])
			}
			var cocartErr *CoCartError
			if !errors.As(err, &cocartErr) {
				t.Error("it must still be a *CoCartError")
			}
			if len(transport.sent) != 1 {
				t.Errorf("sent %d times, want 1", len(transport.sent))
			}
		})
	}
}

func TestPlaceOrderKeepsAnErrorTheStoreAnsweredWith(t *testing.T) {
	c, _ := outcomeClient(400)
	_, err := c.Checkout().PlaceOrder(context.Background(), outcomeOrder())

	var unknown *CheckoutOutcomeUnknownError
	if errors.As(err, &unknown) {
		t.Fatal("the store answered: the outcome is known")
	}
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) || cocartErr.ErrorCode != "cocart_cant_checkout_empty_cart" {
		t.Errorf("err = %#v", err)
	}
}

func TestPlaceOrderChecksTheCheckoutIDGivenBeforeSending(t *testing.T) {
	tests := []struct {
		id    any
		valid bool
	}{
		{"chk_0123456789abcdef", true},
		{"0123456789abcdef", true},
		{strings.Repeat("a", 64), true},
		{"too short", false},
		{"chk_012345678", false},
		{strings.Repeat("a", 65), false},
		{"chk_0123456789abcdef!", false},
		{"chk_0123456789 abcdef", false},
		{12345678901234567, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.id), func(t *testing.T) {
			c, transport := outcomeClient(200)
			input := outcomeOrder()
			input["checkout_id"] = tt.id
			_, err := c.Checkout().PlaceOrder(context.Background(), input)

			if tt.valid {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var apiErr *CoCartError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != "checkout_invalid_checkout_id" {
				t.Fatalf("err = %v, want checkout_invalid_checkout_id", err)
			}
			if len(transport.sent) != 0 {
				t.Error("nothing should be sent")
			}
		})
	}
}

func TestPlaceOrderPaymentInProgressMeansTheOutcomeIsUnknown(t *testing.T) {
	c, _ := outcomeClient(http.StatusConflict)
	_, err := c.Checkout().PlaceOrder(context.Background(), outcomeOrder())

	var unknown *CheckoutOutcomeUnknownError
	if !errors.As(err, &unknown) {
		t.Fatalf("err = %#v, want *CheckoutOutcomeUnknownError", err)
	}
	if unknown.ErrorCode != "cocart_payment_in_progress" || unknown.HTTPCode != http.StatusConflict {
		t.Errorf("wrapped error = %d %s", unknown.HTTPCode, unknown.ErrorCode)
	}
}

func TestCheckoutResultWithoutACodeReadsAsFailed(t *testing.T) {
	for name, body := range map[string]string{
		"no result":  `{"object":"order","id":217}`,
		"empty code": `{"object":"order","result":{"code":""},"id":217}`,
		"not JSON":   `<html></html>`,
	} {
		t.Run(name, func(t *testing.T) {
			r := checkoutResultFromResponse(&Response{StatusCode: 200, Body: []byte(body)})
			if r.Code != PaymentFailed || !r.IsFailed() || r.IsComplete() {
				t.Errorf("Code = %q, IsComplete = %v", r.Code, r.IsComplete())
			}
		})
	}
}

func TestPrintingACheckoutResultHidesTheClientSecretAndOrderKey(t *testing.T) {
	body := `{"object":"order","result":{"code":"action_required","message":"Confirm the payment.","action":{"type":"stripe_confirm_payment","data":{"client_secret":"pi_123_secret_456","intent_type":"payment_intent"}}},"id":217,"number":"217","key":"wc_order_abc","status":{"code":"pending"}}`
	r := checkoutResultFromResponse(&Response{StatusCode: 200, Body: []byte(body)})
	if r.ClientSecret != "pi_123_secret_456" || r.OrderKey != "wc_order_abc" {
		t.Fatalf("result = ClientSecret %q, OrderKey %q", r.ClientSecret, r.OrderKey)
	}

	for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
		for _, value := range []any{r, *r, struct{ Result *CheckoutResult }{r}} {
			out := fmt.Sprintf(format, value)
			if strings.Contains(out, "pi_123_secret_456") || strings.Contains(out, "wc_order_abc") {
				t.Errorf("%s of %T shows a secret: %s", format, value, out)
			}
			if !strings.Contains(out, "action_required") {
				t.Errorf("%s of %T = %s, want the code", format, value, out)
			}
		}
	}
}
