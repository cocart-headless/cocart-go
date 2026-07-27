package cocart

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Response wraps an HTTP response from the CoCart API.
type Response struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Headers contains the response headers.
	Headers http.Header
	// Body is the raw response body.
	Body []byte

	parsed any
	once   sync.Once
}

// Unmarshal deserializes the response body into the given type T.
func Unmarshal[T any](r *Response) (T, error) {
	var result T
	err := json.Unmarshal(r.Body, &result)
	return result, err
}

// ToObject parses the body as a generic map.
func (r *Response) ToObject() map[string]any {
	r.once.Do(func() {
		var data map[string]any
		if err := json.Unmarshal(r.Body, &data); err != nil {
			data = make(map[string]any)
		}
		r.parsed = data
	})
	if m, ok := r.parsed.(map[string]any); ok {
		return m
	}
	return make(map[string]any)
}

// ToJSON returns the response body as a JSON string.
// Pass true for pretty-printed output. Defaults to compact JSON.
func (r *Response) ToJSON(pretty ...bool) string {
	obj := r.ToObject()
	if len(pretty) > 0 && pretty[0] {
		b, _ := json.MarshalIndent(obj, "", "  ")
		return string(b)
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

// IsSuccessful returns true for 2xx status codes.
func (r *Response) IsSuccessful() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError returns true for 4xx/5xx status codes.
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// Get retrieves a nested value using dot notation (e.g., "items.0.name").
// Returns defaultValue if the key is not found.
func (r *Response) Get(key string, defaultValue ...any) any {
	data := r.ToObject()
	keys := strings.Split(key, ".")
	var current any = data

	for _, k := range keys {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[k]
			if !ok {
				return getDefault(defaultValue)
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(k)
			if err != nil || idx < 0 || idx >= len(v) {
				return getDefault(defaultValue)
			}
			current = v[idx]
		default:
			return getDefault(defaultValue)
		}
	}

	return current
}

// GetString retrieves a string value using dot notation.
// Returns defaultValue if the key is not found or the value is not a string.
func (r *Response) GetString(key string, defaultValue string) string {
	v := r.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return defaultValue
}

// GetInt retrieves an int value using dot notation.
// Handles JSON numbers (float64) automatically.
// Returns defaultValue if the key is not found or cannot be converted.
func (r *Response) GetInt(key string, defaultValue int) int {
	v := r.Get(key)
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return defaultValue
}

// GetFloat retrieves a float64 value using dot notation.
// Returns defaultValue if the key is not found or cannot be converted.
func (r *Response) GetFloat(key string, defaultValue float64) float64 {
	v := r.Get(key)
	if f, ok := v.(float64); ok {
		return f
	}
	return defaultValue
}

// Has checks if a key exists in the response data using dot notation.
func (r *Response) Has(key string) bool {
	data := r.ToObject()
	keys := strings.Split(key, ".")
	var current any = data

	for _, k := range keys {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[k]
			if !ok {
				return false
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(k)
			if err != nil || idx < 0 || idx >= len(v) {
				return false
			}
			current = v[idx]
		default:
			return false
		}
	}

	return true
}

// GetHeader returns a response header value (case-insensitive).
func (r *Response) GetHeader(name string) string {
	return r.Headers.Get(name)
}

// --- Cart helpers ---

// GetCartKey returns the cart key from the Cart-Key response header.
func (r *Response) GetCartKey() string {
	return r.GetHeader("Cart-Key")
}

// GetCartHash returns the cart hash from the response data.
func (r *Response) GetCartHash() string {
	if v, ok := r.Get("cart_hash").(string); ok {
		return v
	}
	return ""
}

// GetItems returns cart items from the response data.
func (r *Response) GetItems() []CartItem {
	items, _ := unmarshalField[[]CartItem](r, "items")
	return items
}

// GetTotals returns cart totals from the response data.
func (r *Response) GetTotals() CartTotals {
	totals, _ := unmarshalField[CartTotals](r, "totals")
	return totals
}

// GetItemCount returns the item count from the response data.
func (r *Response) GetItemCount() int {
	if v, ok := r.Get("item_count").(float64); ok {
		return int(v)
	}
	return 0
}

// HasItems returns true if the cart has items.
func (r *Response) HasItems() bool {
	return r.GetItemCount() > 0
}

// IsEmpty returns true if the cart is empty.
func (r *Response) IsEmpty() bool {
	return r.GetItemCount() == 0
}

// GetNotices returns notices from the response data.
func (r *Response) GetNotices() []any {
	if v, ok := r.Get("notices").([]any); ok {
		return v
	}
	return nil
}

// GetCoupons returns applied coupons from the response data.
func (r *Response) GetCoupons() []CartCoupon {
	coupons, _ := unmarshalField[[]CartCoupon](r, "coupons")
	return coupons
}

// HasCoupons returns true if the cart has coupons applied.
func (r *Response) HasCoupons() bool {
	return len(r.GetCoupons()) > 0
}

// GetCustomer returns customer details from the response data.
func (r *Response) GetCustomer() CartCustomer {
	customer, _ := unmarshalField[CartCustomer](r, "customer")
	return customer
}

// GetCurrency returns currency information from the response data.
func (r *Response) GetCurrency() CurrencyInfo {
	currency, _ := unmarshalField[CurrencyInfo](r, "currency")
	return currency
}

// GetShippingMethods returns shipping packages from the response data.
func (r *Response) GetShippingMethods() []ShippingPackage {
	shipping, _ := unmarshalField[[]ShippingPackage](r, "shipping")
	return shipping
}

// GetFees returns cart fees from the response data.
func (r *Response) GetFees() []CartFee {
	fees, _ := unmarshalField[[]CartFee](r, "fees")
	return fees
}

// GetTaxes returns cart tax lines from the response data, normalized to a
// flat slice — returned as-is when the "taxes" field is already an array,
// or converted when it's a legacy object keyed by tax rate code (e.g.
// {"US-US-1": {"name": ..., "price": ...}}), which some CoCart plugin
// versions return instead. Callers never need to branch on which shape the
// server sent.
func (r *Response) GetTaxes() []CartTax {
	raw := r.Get("taxes")

	switch v := raw.(type) {
	case []any:
		taxes := make([]CartTax, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			taxes = append(taxes, CartTax{
				Key:   getStringField(m, "key"),
				Name:  getStringField(m, "name"),
				Price: getStringField(m, "price"),
			})
		}
		return taxes
	case map[string]any:
		taxes := make([]CartTax, 0, len(v))
		for key, tax := range v {
			m, ok := tax.(map[string]any)
			if !ok {
				continue
			}
			taxes = append(taxes, CartTax{
				Key:   key,
				Name:  getStringField(m, "name"),
				Price: getStringField(m, "price"),
			})
		}
		return taxes
	default:
		return nil
	}
}

// HasTaxes returns true if the cart has any tax lines.
func (r *Response) HasTaxes() bool {
	return len(r.GetTaxes()) > 0
}

// GetCrossSells returns cross-sell products from the response data.
func (r *Response) GetCrossSells() []CrossSellProduct {
	crossSells, _ := unmarshalField[[]CrossSellProduct](r, "cross_sells")
	return crossSells
}

// --- Pagination helpers ---

// GetTotalResults returns the total number of results from X-WP-Total header.
func (r *Response) GetTotalResults() int {
	v := r.GetHeader("X-WP-Total")
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

// GetTotalPages returns the total number of pages from X-WP-TotalPages header.
func (r *Response) GetTotalPages() int {
	v := r.GetHeader("X-WP-TotalPages")
	if v == "" {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

// --- Cache helpers ---

// GetETag returns the ETag header value.
func (r *Response) GetETag() string {
	return r.GetHeader("ETag")
}

// IsNotModified returns true if the response is 304 Not Modified.
func (r *Response) IsNotModified() bool {
	return r.StatusCode == 304
}

// GetCacheStatus returns the CoCart-Cache header value (HIT, MISS, or SKIP).
func (r *Response) GetCacheStatus() string {
	return r.GetHeader("CoCart-Cache")
}

// --- Error helpers ---

// GetErrorCode returns the API error code from an error response.
func (r *Response) GetErrorCode() string {
	if !r.IsError() {
		return ""
	}
	if v, ok := r.Get("code").(string); ok {
		return v
	}
	return ""
}

// GetErrorMessage returns the error message from an error response.
func (r *Response) GetErrorMessage() string {
	if !r.IsError() {
		return ""
	}
	if v, ok := r.Get("message").(string); ok {
		return v
	}
	return ""
}

// --- Internal helpers ---

// getStringField reads a string value from a generic map, returning "" if
// the key is missing or not a string.
func getStringField(m map[string]any, key string) string {
	if s, ok := m[key].(string); ok {
		return s
	}
	return ""
}

func getDefault(defaultValue []any) any {
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return nil
}

// unmarshalField extracts a field from the parsed response and unmarshals it into T.
func unmarshalField[T any](r *Response, field string) (T, error) {
	var result T
	data := r.ToObject()
	v, ok := data[field]
	if !ok {
		return result, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}
