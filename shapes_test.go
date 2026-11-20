package cocart

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// readFixture reads a response recorded from CoCart Starter 5 (testdata/).
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return b
}

func decodeFixture[T any](t *testing.T, name string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(readFixture(t, name), &v); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return v
}

func TestShapesCart(t *testing.T) {
	cart := decodeFixture[CartResponse](t, "cart.json")

	if len(cart.Items) != 2 || cart.Items[0].ID != 154 || cart.Items[0].Price != "1800" || cart.Items[0].RegularPrice != "2000" {
		t.Fatalf("items = %+v", cart.Items)
	}
	if cart.Items[0].Meta.Weight != "0.0" {
		t.Errorf("meta.weight = %q, want 0.0", cart.Items[0].Meta.Weight)
	}
	if cart.Items[1].Variation["Color"] != "Blue" {
		t.Errorf("item variation = %v", cart.Items[1].Variation)
	}
	if cart.ItemsWeight != "0" {
		t.Errorf("items_weight = %q", cart.ItemsWeight)
	}
	if cart.Customer.BillingAddress.Country != "US" || cart.Customer.ShippingAddress.State != "CA" {
		t.Errorf("customer = %+v", cart.Customer)
	}
	if cart.Currency.CurrencySymbolPos != "currency_prefix" {
		t.Errorf("currency = %+v", cart.Currency)
	}
	if len(cart.Coupons) != 1 || cart.Coupons[0].DiscountType != "percent" || cart.Coupons[0].Saving != "-630" {
		t.Errorf("coupons = %+v", cart.Coupons)
	}
	pkg, ok := cart.Shipping.Packages["default"]
	if !ok || !cart.Shipping.HasCalculatedShipping || pkg.PackageName != "Shipment" {
		t.Fatalf("shipping = %+v", cart.Shipping)
	}
	rate := pkg.Rates["flat_rate:1"]
	if rate.Cost != "500" || rate.Taxes != "50" || !rate.ChosenMethod || rate.InstanceID != 1 {
		t.Errorf("rate = %+v", rate)
	}
	if len(cart.Taxes) != 1 || cart.Taxes[0].Key != "US-US-1" || cart.Taxes[0].Price != "617" {
		t.Errorf("taxes = %+v", cart.Taxes)
	}
	if cart.Totals.Total != "6787" {
		t.Errorf("totals = %+v", cart.Totals)
	}
	// CoCart Starter 5 sends empty maps as {}.
	if len(cart.Fees) != 0 || len(cart.Notices) != 0 || len(cart.Extensions) != 0 {
		t.Errorf("fees = %v, notices = %v, extensions = %v", cart.Fees, cart.Notices, cart.Extensions)
	}
}

func TestShapesCartTotalsAndItems(t *testing.T) {
	totals := decodeFixture[CartTotalsBreakdown](t, "cart-totals.json")
	if totals.CartContentsTotal != "5670" || totals.ShippingTaxes["4"] != "50" || len(totals.FeeTaxes) != 0 {
		t.Errorf("totals = %+v", totals)
	}

	// GET /cart/items keys the items by item key.
	items := decodeFixture[ObjectMap[CartItem]](t, "cart-items.json")
	if items["1d7f7abc18fcb43975065399b0d1e48e"].Name != "Beanie with Logo" || len(items) != 2 {
		t.Errorf("items = %v", items)
	}
}

func TestShapesProducts(t *testing.T) {
	simple := decodeFixture[Product](t, "product-simple.json")
	if simple.ID != 154 || simple.Prices.Price != "1800" || simple.Prices.Currency.CurrencyCode != "USD" {
		t.Errorf("simple product = %+v", simple.Prices)
	}
	if simple.Images[0].Src["thumbnail"] == "" || simple.Images[0].Src["full"] == "" || !simple.Images[0].Featured {
		t.Errorf("image src = %v", simple.Images[0].Src)
	}
	if simple.Attributes["attribute_pa_color"].Options["red"] != "Red" {
		t.Errorf("attributes = %+v", simple.Attributes)
	}
	if simple.TotalSales != "25" || simple.Stock.StockQuantity != nil || !simple.Stock.IsInStock {
		t.Errorf("total_sales = %q, stock = %+v", simple.TotalSales, simple.Stock)
	}
	if simple.AddToCart.PurchaseQuantity.Maximum != 9999 || simple.Related[0].Price != "400" {
		t.Errorf("add_to_cart = %+v, related = %+v", simple.AddToCart, simple.Related)
	}

	variable := decodeFixture[Product](t, "product-variable.json")
	if variable.Prices.PriceRange == nil || variable.Prices.PriceRange.From != "4200" || variable.Prices.PriceRange.To != "4500" {
		t.Errorf("price_range = %+v", variable.Prices.PriceRange)
	}
	if len(variable.Variations) == 0 || variable.Variations[0].Attributes["attribute_pa_color"] != "blue" || variable.Variations[0].FeaturedImage["full"] == "" {
		t.Errorf("variations = %+v", variable.Variations)
	}
	if !variable.Attributes["attribute_logo"].UsedForVariation {
		t.Errorf("attributes = %+v", variable.Attributes)
	}

	list := decodeFixture[ProductsResponse](t, "products.json")
	if len(list.Products) == 0 || list.TotalProducts == 0 || list.TotalPages == 0 {
		t.Errorf("products = %d, total = %d, pages = %d", len(list.Products), list.TotalProducts, list.TotalPages)
	}
}

func TestShapesVariations(t *testing.T) {
	variation := decodeFixture[ProductVariation](t, "product-variation.json")
	if variation.ParentID != 134 || variation.Attributes["attribute_pa_color"].Option["blue"] != "Blue" {
		t.Errorf("variation = %+v", variation.Attributes)
	}
	// A variation's total_sales is a string; a product's is a number.
	if variation.TotalSales != "0" {
		t.Errorf("total_sales = %q", variation.TotalSales)
	}

	variations := decodeFixture[[]ProductVariation](t, "product-variations.json")
	if len(variations) == 0 || variations[0].ID == 0 {
		t.Errorf("variations = %+v", variations)
	}
}

func TestShapesStoreCategoriesTagsReviews(t *testing.T) {
	store := decodeFixture[StoreInfo](t, "store.json")
	if store.HomeURL != "https://next-wp.instawp.xyz" || store.Language != "en-US" || store.Routes["cart"] == nil {
		t.Errorf("store = %+v", store)
	}

	categories := decodeFixture[[]ProductCategoryDetail](t, "categories.json")
	if len(categories) == 0 || categories[0].Name != "Accessories" || categories[0].ParentID != 17 {
		t.Errorf("categories = %+v", categories)
	}

	decodeFixture[[]ProductTag](t, "tags.json")
	decodeFixture[[]ProductReview](t, "reviews.json")
}

func TestShapesEmptyMapsFromCoCart4(t *testing.T) {
	// CoCart 4.x sent empty maps as [].
	var cart CartResponse
	body := `{"fees":[],"notices":[],"extensions":[],"meta":[],"items":[{"cart_item_data":[],"variation":[],"extensions":[]}],"shipping":{"packages":[]}}`
	if err := json.Unmarshal([]byte(body), &cart); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cart.Fees) != 0 || len(cart.Notices) != 0 || len(cart.Shipping.Packages) != 0 {
		t.Errorf("cart = %+v", cart)
	}

	var product Product
	if err := json.Unmarshal([]byte(`{"prices":{"price_range":[]},"attributes":[],"default_attributes":[],"add_to_cart":{"purchase_quantity":[]}}`), &product); err != nil {
		t.Fatalf("decode product: %v", err)
	}
	if product.Prices.PriceRange == nil || *product.Prices.PriceRange != (ProductPriceRange{}) {
		t.Errorf("price_range = %+v", product.Prices.PriceRange)
	}
}

func TestShapesLegacyTaxesObject(t *testing.T) {
	var cart CartResponse
	if err := json.Unmarshal([]byte(`{"taxes":{"US-US-1":{"name":"State Tax","price":"500"}}}`), &cart); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := CartTaxes{{Key: "US-US-1", Name: "State Tax", Price: "500"}}
	if !reflect.DeepEqual(cart.Taxes, want) {
		t.Errorf("taxes = %+v, want %+v", cart.Taxes, want)
	}
}

func TestFlexString(t *testing.T) {
	tests := []struct {
		json string
		want FlexString
	}{
		{`"0.5"`, "0.5"},
		{`25`, "25"},
		{`1.5`, "1.5"},
		{`null`, ""},
		{`true`, "true"},
	}
	for _, tt := range tests {
		var v FlexString
		if err := json.Unmarshal([]byte(tt.json), &v); err != nil {
			t.Errorf("%s: %v", tt.json, err)
			continue
		}
		if v != tt.want {
			t.Errorf("%s = %q, want %q", tt.json, v, tt.want)
		}
	}
	var v FlexString
	if err := json.Unmarshal([]byte(`{}`), &v); err == nil {
		t.Error("an object should not decode into FlexString")
	}
}

func TestObjectMapNonEmptyArrayIsKeyedByIndex(t *testing.T) {
	var m ObjectMap[string]
	if err := json.Unmarshal([]byte(`["a","b"]`), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(m, ObjectMap[string]{"0": "a", "1": "b"}) {
		t.Errorf("map = %v", m)
	}
}

// --- Response helpers on the recorded cart ---

func recordedCart(t *testing.T) *Response {
	return &Response{StatusCode: 200, Headers: http.Header{}, Body: readFixture(t, "cart.json")}
}

func TestResponseGetShippingMethodsListsThePackages(t *testing.T) {
	packages := recordedCart(t).GetShippingMethods()
	if len(packages) != 1 || packages[0].PackageName != "Shipment" {
		t.Fatalf("packages = %+v", packages)
	}
	if _, ok := packages[0].Rates["free_shipping:2"]; !ok {
		t.Errorf("rates = %v", packages[0].Rates)
	}
}

func TestResponseGetShippingReturnsTheWholeField(t *testing.T) {
	shipping := recordedCart(t).GetShipping()
	if shipping == nil || !shipping.HasCalculatedShipping || !shipping.ShowPackageDetails {
		t.Fatalf("shipping = %+v", shipping)
	}
	if _, ok := shipping.Packages["default"]; !ok {
		t.Errorf("packages = %v", shipping.Packages)
	}
}

func TestResponseGetShippingMethodsAcceptsAListAndIsEmptyWithoutShipping(t *testing.T) {
	list := &Response{Body: []byte(`{"shipping":[{"package_name":"A"}]}`)}
	if packages := list.GetShippingMethods(); len(packages) != 1 || packages[0].PackageName != "A" {
		t.Errorf("packages = %+v", packages)
	}
	none := &Response{Body: []byte(`{"items":[]}`)}
	if len(none.GetShippingMethods()) != 0 || none.GetShipping() != nil {
		t.Error("expected no shipping")
	}
}

func TestResponseGetShippingMethodsKeepsThePackageOrder(t *testing.T) {
	r := &Response{Body: []byte(`{"shipping":{"packages":{"vendor_9":{"package_name":"B"},"vendor_2":{"package_name":"A"},"vendor_5":{"package_name":"C"}}}}`)}
	var names []string
	for _, p := range r.GetShippingMethods() {
		names = append(names, p.PackageName)
	}
	if !reflect.DeepEqual(names, []string{"B", "A", "C"}) {
		t.Errorf("order = %v, want B, A, C", names)
	}
}

func TestResponseGetFeesListsFeesKeyedByFeeID(t *testing.T) {
	r := &Response{Body: []byte(`{"fees":{"handling":{"name":"Handling","fee":"150"},"gift-wrap":{"name":"Gift wrap","fee":"500"}}}`)}
	want := []CartFee{{Name: "Handling", Fee: "150"}, {Name: "Gift wrap", Fee: "500"}}
	if fees := r.GetFees(); !reflect.DeepEqual(fees, want) {
		t.Errorf("fees = %+v, want %+v", fees, want)
	}
	for _, body := range []string{`{"fees":{}}`, `{"fees":[]}`, `{}`} {
		if fees := (&Response{Body: []byte(body)}).GetFees(); len(fees) != 0 {
			t.Errorf("%s: fees = %+v", body, fees)
		}
	}
	if fees := recordedCart(t).GetFees(); len(fees) != 0 {
		t.Errorf("recorded cart fees = %+v", fees)
	}
}

func TestResponseGetNoticesKeyedByType(t *testing.T) {
	r := &Response{Body: []byte(`{"notices":{"success":["Item added"],"notice":["Quantity reduced"]}}`)}
	want := map[string][]string{"success": {"Item added"}, "notice": {"Quantity reduced"}}
	if notices := r.GetNotices(); !reflect.DeepEqual(notices, want) {
		t.Errorf("notices = %v, want %v", notices, want)
	}
	// CoCart Starter 5 sends {}, CoCart 4.x [].
	for _, body := range []string{`{"notices":{}}`, `{"notices":[]}`, `{}`} {
		notices := (&Response{Body: []byte(body)}).GetNotices()
		if notices == nil || len(notices) != 0 {
			t.Errorf("%s: notices = %#v, want an empty map", body, notices)
		}
	}
	if len(recordedCart(t).GetNotices()) != 0 {
		t.Error("recorded cart has no notices")
	}
}

func TestResponseGetExtensions(t *testing.T) {
	r := &Response{Body: []byte(`{"extensions":{"recurring_totals":{"abc":{"totals":{"period":"/ month"}}}}}`)}
	recurring, ok := r.GetExtensions()["recurring_totals"].(map[string]any)
	if !ok || recurring["abc"] == nil {
		t.Errorf("extensions = %v", r.GetExtensions())
	}
	for _, body := range []string{`{"extensions":[]}`, `{}`} {
		if ext := (&Response{Body: []byte(body)}).GetExtensions(); ext == nil || len(ext) != 0 {
			t.Errorf("%s: extensions = %#v", body, ext)
		}
	}
}

func TestResponseGetItemsOnTheRecordedCart(t *testing.T) {
	r := recordedCart(t)
	items := r.GetItems()
	if len(items) != 2 || items[1].ID != 157 {
		t.Fatalf("items = %+v", items)
	}
	if r.GetCustomer().BillingAddress.Country != "US" {
		t.Errorf("customer = %+v", r.GetCustomer())
	}
}

func TestResponseGetRateLimit(t *testing.T) {
	h := http.Header{}
	h.Set("RateLimit-Limit", "25")
	h.Set("ratelimit-remaining", "24")
	h.Set("RateLimit-Reset", "1791108883")
	got := (&Response{Headers: h}).GetRateLimit()
	if got.Limit == nil || *got.Limit != 25 || *got.Remaining != 24 || *got.Reset != 1791108883 || got.RetryAfter != nil {
		t.Errorf("rate limit = %+v", got)
	}

	h = http.Header{}
	h.Set("RateLimit-Retry-After", "30")
	h.Set("RateLimit-Limit", "abc")
	got = (&Response{Headers: h}).GetRateLimit()
	if got.RetryAfter == nil || *got.RetryAfter != 30 || got.Limit != nil {
		t.Errorf("rate limit = %+v", got)
	}

	none := (&Response{Headers: http.Header{}}).GetRateLimit()
	if none != (RateLimitInfo{}) {
		t.Errorf("rate limit without headers = %+v", none)
	}
}
