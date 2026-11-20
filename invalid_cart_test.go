package cocart

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const invalidCart = `{"code":"cocart_invalid_cart","message":"Cart is not valid.","data":{"status":404}}`

func storageWith(key, value string) *MemoryStorage {
	s := NewMemoryStorage()
	s.Set(key, value)
	return s
}

func storedValue(s Storage, key string) string {
	v, _ := s.Get(key)
	return v
}

// CoCart Starter 5 refuses a cart key with no cart session (e.g. an expired cart).

func TestInvalidCartForgetsTheKeyAndRetriesOnceAsANewGuest(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, invalidCart)
	s.queue(200, `{"items":[]}`, map[string]string{"Cart-Key": "t_fresh"})
	storage := storageWith("cocart_cart_key", "t_expired")

	c := NewClient(s.URL, WithStorage(storage))
	if c.GetCartKey() != "t_expired" {
		t.Fatalf("cart key = %q, want the stored one", c.GetCartKey())
	}
	resp, err := c.Cart().Get(context.Background())
	if err != nil || !resp.IsSuccessful() {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := s.all()
	if len(reqs) != 2 {
		t.Fatalf("requests = %d, want 2", len(reqs))
	}
	if reqs[0].Query.Get("cart_key") != "t_expired" {
		t.Errorf("first request cart_key = %q", reqs[0].Query.Get("cart_key"))
	}
	if reqs[1].Query.Has("cart_key") || reqs[1].Headers.Get("Cart-Key") != "" {
		t.Error("the retry should be sent without a cart key")
	}
	if c.GetCartKey() != "t_fresh" || storedValue(storage, "cocart_cart_key") != "t_fresh" {
		t.Errorf("cart key = %q, stored = %q, want t_fresh", c.GetCartKey(), storedValue(storage, "cocart_cart_key"))
	}
}

func TestInvalidCartRetriesAWriteWithTheSameBody(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, invalidCart)
	c := NewClient(s.URL, WithCartKey("t_expired"))
	c.Cart().AddItem(context.Background(), 42, 1)

	reqs := s.all()
	if len(reqs) != 2 || reqs[0].Method != http.MethodPost || reqs[0].Body != reqs[1].Body {
		t.Errorf("requests = %+v", reqs)
	}
}

func TestInvalidCartRetriesOnlyOnce(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, invalidCart)
	s.queue(404, invalidCart)
	c := NewClient(s.URL, WithCartKey("t_expired"))

	_, err := c.Cart().Get(context.Background())
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) || cocartErr.ErrorCode != "cocart_invalid_cart" {
		t.Errorf("err = %v", err)
	}
	if len(s.all()) != 2 {
		t.Errorf("requests = %d, want 2", len(s.all()))
	}
}

func TestInvalidCartWithoutACartKeyIsNotRetried(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, invalidCart)
	if _, err := NewClient(s.URL).Cart().Get(context.Background()); err == nil {
		t.Error("expected an error")
	}
	if len(s.all()) != 1 {
		t.Errorf("requests = %d, want 1", len(s.all()))
	}
}

func TestInvalidCartWhenLoggedInIsNotRetried(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, invalidCart)
	storage := storageWith("cocart_cart_key", "t_expired")
	c := NewClient(s.URL, WithStorage(storage), WithBasicAuth("user", "pass"))

	if _, err := c.Cart().Get(context.Background()); err == nil {
		t.Error("expected an error")
	}
	if len(s.all()) != 1 || storedValue(storage, "cocart_cart_key") != "t_expired" {
		t.Errorf("requests = %d, stored = %q", len(s.all()), storedValue(storage, "cocart_cart_key"))
	}
}

func TestInvalidCartLeavesOtherErrorsAlone(t *testing.T) {
	s := newRecordingServer(t)
	s.queue(404, `{"code":"cocart_item_not_in_cart","message":"Item not in cart."}`)
	storage := storageWith("cocart_cart_key", "t_expired")
	c := NewClient(s.URL, WithStorage(storage))

	if _, err := c.Cart().GetItem(context.Background(), "abc"); err == nil {
		t.Error("expected an error")
	}
	if len(s.all()) != 1 || c.GetCartKey() != "t_expired" || storedValue(storage, "cocart_cart_key") != "t_expired" {
		t.Errorf("requests = %d, cart key = %q", len(s.all()), c.GetCartKey())
	}
}

// --- The cart key kept by the storage adapter is picked up on construction ---

func TestNewClientRestoresTheCartKeyFromStorage(t *testing.T) {
	c := NewClient("https://example.com", WithStorage(storageWith("cocart_cart_key", "t_stored")))
	if c.GetCartKey() != "t_stored" {
		t.Errorf("cart key = %q, want t_stored", c.GetCartKey())
	}
}

func TestNewClientReadsTheKeyNamedAfterTheStorageKey(t *testing.T) {
	c := NewClient("https://example.com", WithStorage(storageWith("shop_cart", "t_named")), WithStorageKey("shop_cart"))
	if c.GetCartKey() != "t_named" {
		t.Errorf("cart key = %q, want t_named", c.GetCartKey())
	}
}

func TestNewClientPrefersAnExplicitCartKey(t *testing.T) {
	c := NewClient("https://example.com", WithStorage(storageWith("cocart_cart_key", "t_stored")), WithCartKey("t_given"))
	if c.GetCartKey() != "t_given" {
		t.Errorf("cart key = %q, want t_given", c.GetCartKey())
	}
}

type brokenStorage struct{}

func (brokenStorage) Get(string) (string, error) { return "", errors.New("unreadable") }
func (brokenStorage) Set(string, string) error   { return nil }
func (brokenStorage) Delete(string) error        { return nil }

func TestNewClientStartsWithoutACartKeyWhenStorageFails(t *testing.T) {
	c := NewClient("https://example.com", WithStorage(brokenStorage{}))
	if c.GetCartKey() != "" {
		t.Errorf("cart key = %q, want none", c.GetCartKey())
	}
}

func TestFromRequestDoesNotReadSharedStorage(t *testing.T) {
	// A server may share one storage between customers: the cart key comes
	// from the request only.
	shared := storageWith("cocart_cart_key", "t_other_customer")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if c := FromRequest("https://example.com", r, WithStorage(shared)); c.GetCartKey() != "" {
		t.Errorf("cart key = %q, want none", c.GetCartKey())
	}
	r.Header.Set("X-Cart-Key", "t_this_customer")
	if c := FromRequest("https://example.com", r, WithStorage(shared)); c.GetCartKey() != "t_this_customer" {
		t.Errorf("cart key = %q, want t_this_customer", c.GetCartKey())
	}
}
