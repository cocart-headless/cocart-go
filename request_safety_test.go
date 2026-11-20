package cocart

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var secrets = []string{"t_secret_cart_key", "ck_live_secret", "cs_live_secret"}

func TestErrorMessagesNameTheRouteButNeverTheQueryString(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(400, `{"code":"cocart_coupon_invalid","message":"Coupon is not valid.","data":{"status":400}}`)
	c := NewClient(s.URL, WithCartKey("t_secret_cart_key"), WithWooCommerceKeys("ck_live_secret", "cs_live_secret"))

	_, err := c.Cart().ApplyCoupon(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected an error")
	}
	message := err.Error()
	if message != "POST /cocart/v2/cart/apply-coupon: Coupon is not valid. [cocart_coupon_invalid]" {
		t.Errorf("message = %q", message)
	}
	if strings.Contains(message, "127.0.0.1") {
		t.Errorf("message names the host: %q", message)
	}
	for _, secret := range secrets {
		if strings.Contains(message, secret) {
			t.Errorf("message contains %s: %q", secret, message)
		}
	}
}

func TestErrorMessagesFromRawRoutesNameTheRoute(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, `{"code":"cocart_product_not_found","message":"Product not found."}`)
	c := NewClient(s.URL, WithCartKey("t_secret_cart_key"), WithRESTPrefix("api"))

	_, err := c.Products().ShippingRates(context.Background(), 99)
	if err == nil || err.Error() != "GET /api/cocart/v2/products/99/shipping-rates: Product not found. [cocart_product_not_found]" {
		t.Errorf("err = %v", err)
	}
}

func TestConnectionErrorsKeepTheCartKeyOut(t *testing.T) {
	// Nothing listens on port 1, so net/http fails with a *url.Error naming the URL.
	c := NewClient("http://127.0.0.1:1", WithCartKey("t_secret_cart_key"), WithTimeout(5*time.Second))

	for name, call := range map[string]func() error{
		"Get": func() error { _, err := c.Cart().Get(context.Background()); return err },
		"RequestRaw": func() error {
			_, err := c.Account().GetProfile(context.Background())
			return err
		},
		"ShippingRates": func() error {
			_, err := c.Products().ShippingRates(context.Background(), 1)
			return err
		},
	} {
		err := call()
		var cocartErr *CoCartError
		if !errors.As(err, &cocartErr) || cocartErr.ErrorCode != "network_error" {
			t.Errorf("%s: err = %#v, want a network_error CoCartError", name, err)
			continue
		}
		if strings.Contains(err.Error(), "t_secret_cart_key") {
			t.Errorf("%s: message contains the cart key: %q", name, err.Error())
		}
		if errors.Unwrap(err) != nil {
			t.Errorf("%s: the original error should not be wrapped", name)
		}
	}
}

func TestSafeErrorMessage(t *testing.T) {
	tests := map[string]string{
		`Get "https://store.com/wp-json/cocart/v2/cart?cart_key=t_secret_cart_key": dial tcp: connection refused`: `Get "https://store.com/wp-json/cocart/v2/cart": dial tcp: connection refused`,
		`fetch /wp-json/cocart/v2/cart?cart_key=t_secret&x=1#frag failed`:                                         `fetch /wp-json/cocart/v2/cart failed`,
		`connection refused`: `connection refused`,
	}
	for in, want := range tests {
		if got := safeErrorMessage(in); got != want {
			t.Errorf("safeErrorMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRoutePath(t *testing.T) {
	tests := map[string]string{
		"https://store.com/wp-json/cocart/v2/cart/add-item?cart_key=x": "/cocart/v2/cart/add-item",
		"https://store.com/shop/wp-json/cocart/v2/cart":                "/cocart/v2/cart",
		"https://store.com/api/cocart/v2/cart?x=1":                     "/api/cocart/v2/cart",
		"https://store.com/wp-json/cocart/v2/cart/coupons/a%20b%25":    "/cocart/v2/cart/coupons/a%20b%25",
	}
	for in, want := range tests {
		if got := routePath(in); got != want {
			t.Errorf("routePath(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- GetCart ---

func TestGetCartReturnsTheCartSentWithCocartItemNotInCart(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, `{"code":"cocart_item_not_in_cart","message":"Item specified does not exist in cart.","data":{"status":404,"cart":{"items":[{"item_key":"def"}],"item_count":1,"totals":{"total":"3600"}}}}`)

	_, err := NewClient(s.URL).Cart().UpdateItem(context.Background(), "abc", 2)
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) {
		t.Fatalf("err = %v", err)
	}
	cart := cocartErr.GetCart()
	if cart == nil || len(cart.Items) != 1 || cart.Items[0].ItemKey != "def" || cart.Totals.Total != "3600" {
		t.Errorf("cart = %+v", cart)
	}
}

func TestGetCartIsNilWhenTheErrorHasNoCart(t *testing.T) {
	for _, data := range []map[string]any{
		nil,
		{"data": map[string]any{"status": 404}},
		{"data": map[string]any{"cart": []any{}}},
	} {
		if cart := NewCoCartError("Nope", 404, "x", data).GetCart(); cart != nil {
			t.Errorf("data %v: cart = %+v, want nil", data, cart)
		}
	}
}
