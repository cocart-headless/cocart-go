package cocart

import (
	"net/http"
	"testing"
)

func TestResponseIsSuccessful(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{301, false},
		{400, false},
		{404, false},
		{500, false},
	}
	for _, tt := range tests {
		r := &Response{StatusCode: tt.code}
		if r.IsSuccessful() != tt.want {
			t.Errorf("IsSuccessful(%d) = %v, want %v", tt.code, r.IsSuccessful(), tt.want)
		}
	}
}

func TestResponseIsError(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{304, false},
		{400, true},
		{401, true},
		{500, true},
	}
	for _, tt := range tests {
		r := &Response{StatusCode: tt.code}
		if r.IsError() != tt.want {
			t.Errorf("IsError(%d) = %v, want %v", tt.code, r.IsError(), tt.want)
		}
	}
}

func TestResponseGet(t *testing.T) {
	body := `{"items":[{"name":"Widget","price":"9.99"}],"totals":{"total":"9.99"},"item_count":1}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	// Simple key
	if count, ok := r.Get("item_count").(float64); !ok || int(count) != 1 {
		t.Errorf("Get('item_count') = %v", r.Get("item_count"))
	}

	// Nested key
	if total := r.Get("totals.total"); total != "9.99" {
		t.Errorf("Get('totals.total') = %v", total)
	}

	// Array index
	if name := r.Get("items.0.name"); name != "Widget" {
		t.Errorf("Get('items.0.name') = %v", name)
	}

	// Missing key with default
	if val := r.Get("missing", "default"); val != "default" {
		t.Errorf("Get('missing', 'default') = %v", val)
	}

	// Missing key without default
	if val := r.Get("missing"); val != nil {
		t.Errorf("Get('missing') = %v, want nil", val)
	}
}

func TestResponseHas(t *testing.T) {
	body := `{"items":[{"name":"Widget"}],"totals":{"total":"9.99"}}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	if !r.Has("items") {
		t.Error("Has('items') should be true")
	}
	if !r.Has("totals.total") {
		t.Error("Has('totals.total') should be true")
	}
	if !r.Has("items.0.name") {
		t.Error("Has('items.0.name') should be true")
	}
	if r.Has("missing") {
		t.Error("Has('missing') should be false")
	}
	if r.Has("items.5.name") {
		t.Error("Has('items.5.name') should be false")
	}
}

func TestResponseCartHelpers(t *testing.T) {
	body := `{
		"items": [{"item_key":"abc","id":1,"name":"Widget","title":"Widget","price":"999","quantity":{"value":2,"minimum":1,"maximum":10,"multiple_of":1,"editable":true},"totals":{"subtotal":"1998","subtotal_tax":"0","total":"1998","tax":"0"},"slug":"widget","meta":{"product_type":"simple","sku":"WDG-1","dimensions":{},"weight":0},"backorders":"no","cart_item_data":{},"featured_image":""}],
		"item_count": 2,
		"coupons": [{"coupon":"SAVE10","label":"Save 10%","saving":"200","saving_html":"$2.00"}],
		"totals": {"subtotal":"1998","total":"1798"},
		"currency": {"currency_code":"USD","currency_symbol":"$","currency_minor_unit":2},
		"customer": {"billing_address":{"first_name":"John"},"shipping_address":{"first_name":"John"}},
		"fees": [],
		"shipping": [],
		"cross_sells": [],
		"notices": ["test notice"]
	}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	if r.GetItemCount() != 2 {
		t.Errorf("GetItemCount() = %d, want 2", r.GetItemCount())
	}
	if !r.HasItems() {
		t.Error("HasItems() should be true")
	}
	if r.IsEmpty() {
		t.Error("IsEmpty() should be false")
	}

	items := r.GetItems()
	if len(items) != 1 {
		t.Fatalf("GetItems() len = %d, want 1", len(items))
	}
	if items[0].Name != "Widget" {
		t.Errorf("item name = %s, want Widget", items[0].Name)
	}
	qty := items[0].Quantity
	if qty.Minimum != 1 || qty.Maximum != 10 || qty.MultipleOf != 1 || !qty.Editable {
		t.Errorf("item quantity = %+v, want {Value:2 Minimum:1 Maximum:10 MultipleOf:1 Editable:true}", qty)
	}

	totals := r.GetTotals()
	if totals.Total != "1798" {
		t.Errorf("totals.Total = %s, want 1798", totals.Total)
	}

	coupons := r.GetCoupons()
	if len(coupons) != 1 || coupons[0].Coupon != "SAVE10" {
		t.Error("GetCoupons() failed")
	}
	if !r.HasCoupons() {
		t.Error("HasCoupons() should be true")
	}

	currency := r.GetCurrency()
	if currency.CurrencyCode != "USD" {
		t.Errorf("currency code = %s, want USD", currency.CurrencyCode)
	}

	notices := r.GetNotices()
	if len(notices) != 1 {
		t.Errorf("notices len = %d, want 1", len(notices))
	}
}

func TestResponsePaginationHelpers(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-WP-Total", "100")
	headers.Set("X-WP-TotalPages", "10")

	r := &Response{StatusCode: 200, Headers: headers, Body: []byte(`{}`)}

	if r.GetTotalResults() != 100 {
		t.Errorf("GetTotalResults() = %d, want 100", r.GetTotalResults())
	}
	if r.GetTotalPages() != 10 {
		t.Errorf("GetTotalPages() = %d, want 10", r.GetTotalPages())
	}
}

func TestResponseCacheHelpers(t *testing.T) {
	headers := http.Header{}
	headers.Set("ETag", `"abc123"`)
	headers.Set("CoCart-Cache", "HIT")

	r := &Response{StatusCode: 200, Headers: headers, Body: []byte(`{}`)}

	if r.GetETag() != `"abc123"` {
		t.Errorf("GetETag() = %s", r.GetETag())
	}
	if r.GetCacheStatus() != "HIT" {
		t.Errorf("GetCacheStatus() = %s", r.GetCacheStatus())
	}
	if r.IsNotModified() {
		t.Error("IsNotModified() should be false for 200")
	}

	r304 := &Response{StatusCode: 304, Headers: http.Header{}, Body: nil}
	if !r304.IsNotModified() {
		t.Error("IsNotModified() should be true for 304")
	}
}

func TestResponseErrorHelpers(t *testing.T) {
	r := &Response{StatusCode: 400, Headers: http.Header{}, Body: []byte(`{"code":"cocart_invalid","message":"Invalid product"}`)}

	if r.GetErrorCode() != "cocart_invalid" {
		t.Errorf("GetErrorCode() = %s", r.GetErrorCode())
	}
	if r.GetErrorMessage() != "Invalid product" {
		t.Errorf("GetErrorMessage() = %s", r.GetErrorMessage())
	}

	// Non-error response should return empty strings
	ok := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(`{"code":"ok"}`)}
	if ok.GetErrorCode() != "" {
		t.Errorf("GetErrorCode() on success = %s", ok.GetErrorCode())
	}
}

func TestResponseGetTaxesArrayShape(t *testing.T) {
	body := `{"taxes":[{"key":"US-US-1","name":"State Tax","price":"1.00"}]}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	taxes := r.GetTaxes()
	if len(taxes) != 1 {
		t.Fatalf("expected 1 tax line, got %d", len(taxes))
	}
	if taxes[0].Key != "US-US-1" || taxes[0].Name != "State Tax" || taxes[0].Price != "1.00" {
		t.Errorf("unexpected tax line: %+v", taxes[0])
	}
	if !r.HasTaxes() {
		t.Error("HasTaxes() should be true")
	}
}

func TestResponseGetTaxesLegacyObjectShape(t *testing.T) {
	body := `{"taxes":{"US-US-1":{"name":"State Tax","price":"1.00"}}}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	taxes := r.GetTaxes()
	if len(taxes) != 1 {
		t.Fatalf("expected 1 tax line, got %d", len(taxes))
	}
	if taxes[0].Key != "US-US-1" || taxes[0].Name != "State Tax" || taxes[0].Price != "1.00" {
		t.Errorf("unexpected tax line: %+v", taxes[0])
	}
	if !r.HasTaxes() {
		t.Error("HasTaxes() should be true")
	}
}

func TestResponseGetTaxesEmpty(t *testing.T) {
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(`{"taxes":[]}`)}
	if r.HasTaxes() {
		t.Error("HasTaxes() should be false for empty taxes")
	}

	r2 := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(`{}`)}
	if r2.HasTaxes() {
		t.Error("HasTaxes() should be false when taxes is missing")
	}
}

func TestUnmarshal(t *testing.T) {
	body := `{"id":42,"name":"Widget","slug":"widget"}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	product, err := Unmarshal[Product](r)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if product.ID != 42 {
		t.Errorf("product.ID = %d, want 42", product.ID)
	}
	if product.Name != "Widget" {
		t.Errorf("product.Name = %s, want Widget", product.Name)
	}
}

func TestResponseTypedGetters(t *testing.T) {
	body := `{"name":"Widget","count":5,"price":29.99,"active":true}`
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(body)}

	// GetString
	if s := r.GetString("name", ""); s != "Widget" {
		t.Errorf("GetString('name') = %s, want Widget", s)
	}
	if s := r.GetString("missing", "default"); s != "default" {
		t.Errorf("GetString('missing') = %s, want default", s)
	}
	if s := r.GetString("count", "fallback"); s != "fallback" {
		t.Errorf("GetString('count') should return default for non-string, got %s", s)
	}

	// GetInt
	if n := r.GetInt("count", 0); n != 5 {
		t.Errorf("GetInt('count') = %d, want 5", n)
	}
	if n := r.GetInt("missing", 42); n != 42 {
		t.Errorf("GetInt('missing') = %d, want 42", n)
	}
	if n := r.GetInt("name", -1); n != -1 {
		t.Errorf("GetInt('name') should return default for string, got %d", n)
	}

	// GetFloat
	if f := r.GetFloat("price", 0); f != 29.99 {
		t.Errorf("GetFloat('price') = %f, want 29.99", f)
	}
	if f := r.GetFloat("missing", 1.5); f != 1.5 {
		t.Errorf("GetFloat('missing') = %f, want 1.5", f)
	}
}

func TestResponseToJSONVariadic(t *testing.T) {
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(`{"key":"value"}`)}

	// No arg defaults to compact
	compact := r.ToJSON()
	if compact != `{"key":"value"}` {
		t.Errorf("ToJSON() = %s", compact)
	}

	// Explicit false
	compact2 := r.ToJSON(false)
	if compact2 != compact {
		t.Errorf("ToJSON(false) != ToJSON()")
	}

	// True for pretty
	pretty := r.ToJSON(true)
	if pretty == compact {
		t.Error("ToJSON(true) should be pretty-printed")
	}
}

func TestResponseToJSON(t *testing.T) {
	r := &Response{StatusCode: 200, Headers: http.Header{}, Body: []byte(`{"key":"value"}`)}

	compact := r.ToJSON(false)
	if compact != `{"key":"value"}` {
		t.Errorf("ToJSON(false) = %s", compact)
	}

	pretty := r.ToJSON(true)
	if pretty == compact {
		t.Error("ToJSON(true) should be pretty-printed")
	}
}
