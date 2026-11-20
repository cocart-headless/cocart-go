package cocart

import (
	"bytes"
	"encoding/json"
)

// MainPlugin identifies which CoCart plugin variant is installed.
type MainPlugin string

const (
	// MainPluginStarter is CoCart Starter (the default).
	MainPluginStarter MainPlugin = "starter"
	// MainPluginLegacy is the CoCart Community plugin (4.x).
	MainPluginLegacy MainPlugin = "legacy"
)

// StockStatus represents a product's stock status.
type StockStatus string

const (
	StockInStock     StockStatus = "instock"
	StockOutOfStock  StockStatus = "outofstock"
	StockOnBackorder StockStatus = "onbackorder"
)

// SortOrder represents sort direction.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ProductOrderBy represents the sort orders CoCart accepts for products.
// Sort by price with OrderByPriceAsc or OrderByPriceDesc: CoCart rejects
// "price". OrderByRelevance and OrderByRand need CoCart Starter.
type ProductOrderBy string

const (
	OrderByDate          ProductOrderBy = "date"
	OrderByID            ProductOrderBy = "id"
	OrderByMenuOrder     ProductOrderBy = "menu_order"
	OrderByInclude       ProductOrderBy = "include"
	OrderByTitle         ProductOrderBy = "title"
	OrderBySlug          ProductOrderBy = "slug"
	OrderByName          ProductOrderBy = "name"
	OrderByAlphabetical  ProductOrderBy = "alphabetical"
	OrderByReverseAlpha  ProductOrderBy = "reverse_alpha"
	OrderByPopularity    ProductOrderBy = "popularity"
	OrderBySales         ProductOrderBy = "sales"
	OrderByRating        ProductOrderBy = "rating"
	OrderByReviewCount   ProductOrderBy = "review_count"
	OrderByStock         ProductOrderBy = "by_stock"
	OrderByOnSaleFirst   ProductOrderBy = "on_sale_first"
	OrderByFeaturedFirst ProductOrderBy = "featured_first"
	OrderByPriceAsc      ProductOrderBy = "price_asc"
	OrderByPriceDesc     ProductOrderBy = "price_desc"
	OrderByRelevance     ProductOrderBy = "relevance"
	OrderByRand          ProductOrderBy = "rand"
)

// CurrencyInfo contains the currency fields sent beside amounts (the cart's
// currency, a product's prices.currency).
type CurrencyInfo struct {
	CurrencyCode        string `json:"currency_code"`
	CurrencySymbol      string `json:"currency_symbol"`
	CurrencySymbolPos   string `json:"currency_symbol_pos"`
	CurrencyMinorUnit   int    `json:"currency_minor_unit"`
	CurrencyDecimalSep  string `json:"currency_decimal_separator"`
	CurrencyThousandSep string `json:"currency_thousand_separator"`
	CurrencyPrefix      string `json:"currency_prefix"`
	CurrencySuffix      string `json:"currency_suffix"`
}

// --- Cart ---

// CartItemQuantity describes quantity constraints for a cart item.
type CartItemQuantity struct {
	Value      int  `json:"value"`
	Minimum    int  `json:"minimum"`
	Maximum    int  `json:"maximum"`
	MultipleOf int  `json:"multiple_of"`
	Editable   bool `json:"editable"`
}

// CartItemTotals contains a cart item's totals in minor-unit strings.
type CartItemTotals struct {
	Subtotal    string `json:"subtotal"`
	SubtotalTax string `json:"subtotal_tax"`
	Total       string `json:"total"`
	Tax         string `json:"tax"`
}

// CartItemMeta contains metadata for a cart item.
type CartItemMeta struct {
	ProductType string            `json:"product_type"`
	SKU         string            `json:"sku"`
	Dimensions  map[string]string `json:"dimensions"`
	// Weight is sent as a string, e.g. "0.5".
	Weight FlexString `json:"weight"`
	// Variation holds a variation's attributes.
	Variation ObjectMap[string] `json:"variation,omitempty"`
}

// CartItemImage represents a cart item image.
//
// Deprecated: cart items carry featured_image as a URL string, not an image
// object; this shape was never returned. Use CartItem.FeaturedImage.
type CartItemImage struct {
	ID        int    `json:"id"`
	Src       string `json:"src"`
	Thumbnail string `json:"thumbnail"`
	Name      string `json:"name"`
	Alt       string `json:"alt"`
}

// CartItem represents a single item in the cart. Prices and totals are
// minor-unit strings ("1800" for 18.00).
type CartItem struct {
	ItemKey      string           `json:"item_key"`
	ID           int              `json:"id"`
	Name         string           `json:"name"`
	Title        string           `json:"title"`
	Price        string           `json:"price"`
	RegularPrice string           `json:"regular_price"`
	OnSale       bool             `json:"on_sale"`
	Quantity     CartItemQuantity `json:"quantity"`
	Totals       CartItemTotals   `json:"totals"`
	Slug         string           `json:"slug"`
	Meta         CartItemMeta     `json:"meta"`
	Backorders   string           `json:"backorders"`
	CartItemData ObjectMap[any]   `json:"cart_item_data"`
	// FeaturedImage is the image URL.
	FeaturedImage string `json:"featured_image"`
	// Variation holds the chosen attributes of a variation, e.g. {"Color": "Blue"}.
	Variation ObjectMap[string] `json:"variation"`
	// Extensions holds data plugins add to the item (see docs/extension-data.md).
	Extensions ObjectMap[any] `json:"extensions"`
}

// CartTotals contains the cart's totals in minor-unit strings, as on GET /cart.
type CartTotals struct {
	Subtotal      string `json:"subtotal"`
	SubtotalTax   string `json:"subtotal_tax"`
	FeeTotal      string `json:"fee_total"`
	FeeTax        string `json:"fee_tax"`
	DiscountTotal string `json:"discount_total"`
	DiscountTax   string `json:"discount_tax"`
	ShippingTotal string `json:"shipping_total"`
	ShippingTax   string `json:"shipping_tax"`
	Total         string `json:"total"`
	TotalTax      string `json:"total_tax"`
}

// CartTotalsBreakdown contains the totals GET /cart/totals returns: the full
// breakdown WooCommerce calculates, in minor-unit strings, with the taxes
// keyed by tax rate ID.
type CartTotalsBreakdown struct {
	Subtotal          string            `json:"subtotal"`
	SubtotalTax       string            `json:"subtotal_tax"`
	ShippingTotal     string            `json:"shipping_total"`
	ShippingTax       string            `json:"shipping_tax"`
	ShippingTaxes     ObjectMap[string] `json:"shipping_taxes"`
	DiscountTotal     string            `json:"discount_total"`
	DiscountTax       string            `json:"discount_tax"`
	CartContentsTotal string            `json:"cart_contents_total"`
	CartContentsTax   string            `json:"cart_contents_tax"`
	CartContentsTaxes ObjectMap[string] `json:"cart_contents_taxes"`
	FeeTotal          string            `json:"fee_total"`
	FeeTax            string            `json:"fee_tax"`
	FeeTaxes          ObjectMap[string] `json:"fee_taxes"`
	Total             string            `json:"total"`
	TotalTax          string            `json:"total_tax"`
}

// CustomerAddress represents an unprefixed address.
//
// Deprecated: the cart's addresses use prefixed field names; use
// CartBillingAddress or CartShippingAddress. (CartEndpoint.UpdateCustomer
// still takes unprefixed fields.)
type CustomerAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address_1"`
	Address2  string `json:"address_2"`
	City      string `json:"city"`
	State     string `json:"state"`
	Postcode  string `json:"postcode"`
	Country   string `json:"country"`
	Phone     string `json:"phone,omitempty"`
	Email     string `json:"email,omitempty"`
}

// CartBillingAddress is the cart's billing address. CoCart prefixes the
// fields: billing_first_name, billing_email, …
type CartBillingAddress struct {
	FirstName string `json:"billing_first_name"`
	LastName  string `json:"billing_last_name"`
	Company   string `json:"billing_company"`
	Country   string `json:"billing_country"`
	Address1  string `json:"billing_address_1"`
	Address2  string `json:"billing_address_2"`
	Postcode  string `json:"billing_postcode"`
	City      string `json:"billing_city"`
	State     string `json:"billing_state"`
	Phone     string `json:"billing_phone"`
	Email     string `json:"billing_email"`
}

// CartShippingAddress is the cart's shipping address. CoCart prefixes the
// fields: shipping_first_name, shipping_city, …
type CartShippingAddress struct {
	FirstName string `json:"shipping_first_name"`
	LastName  string `json:"shipping_last_name"`
	Company   string `json:"shipping_company"`
	Country   string `json:"shipping_country"`
	Address1  string `json:"shipping_address_1"`
	Address2  string `json:"shipping_address_2"`
	Postcode  string `json:"shipping_postcode"`
	City      string `json:"shipping_city"`
	State     string `json:"shipping_state"`
	Phone     string `json:"shipping_phone"`
}

// CartCustomer contains the cart customer's addresses.
type CartCustomer struct {
	BillingAddress  CartBillingAddress  `json:"billing_address"`
	ShippingAddress CartShippingAddress `json:"shipping_address"`
}

// CartCoupon represents an applied coupon. Saving is a negative minor-unit
// amount ("-180").
type CartCoupon struct {
	Coupon       string `json:"coupon"`
	Label        string `json:"label"`
	Saving       string `json:"saving"`
	SavingHTML   string `json:"saving_html"`
	DiscountType string `json:"discount_type"`
}

// CartFee represents a cart fee: its name and the fee amount in minor units.
// The cart keys fees by fee ID; [Response.GetFees] lists them.
type CartFee struct {
	Name string `json:"name"`
	Fee  string `json:"fee"`
}

// CartTax represents a normalized cart tax line, as returned by
// [Response.GetTaxes]. One entry per tax rate when the store's tax display
// setting is itemized (Key is WooCommerce's composite rate code, e.g.
// "US-US-1"), or a single synthetic entry keyed "total" when it isn't.
type CartTax struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Price string `json:"price"`
}

// CartTaxes is the cart's tax lines. CoCart Starter 5 sends a list; the
// community CoCart plugin and older versions send an object keyed by tax rate
// code, which is converted to the same list.
type CartTaxes []CartTax

// UnmarshalJSON accepts a list of tax lines or an object keyed by tax rate code.
func (t *CartTaxes) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var legacy map[string]CartTax
		if err := json.Unmarshal(trimmed, &legacy); err != nil {
			return err
		}
		out := make(CartTaxes, 0, len(legacy))
		for key, tax := range legacy {
			tax.Key = key
			out = append(out, tax)
		}
		*t = out
		return nil
	}
	var list []CartTax
	if err := json.Unmarshal(trimmed, &list); err != nil {
		return err
	}
	*t = list
	return nil
}

// ShippingRate represents a shipping rate option. Cost and Taxes are
// minor-unit strings.
type ShippingRate struct {
	Key          string         `json:"key"`
	MethodID     string         `json:"method_id"`
	InstanceID   int            `json:"instance_id"`
	Label        string         `json:"label"`
	Cost         string         `json:"cost"`
	HTML         string         `json:"html"`
	Taxes        string         `json:"taxes"`
	ChosenMethod bool           `json:"chosen_method"`
	MetaData     ObjectMap[any] `json:"meta_data"`
}

// ShippingPackage represents a shipping package. Its rates are keyed by rate
// ID (e.g. "flat_rate:1").
type ShippingPackage struct {
	PackageName          string                  `json:"package_name"`
	Rates                ObjectMap[ShippingRate] `json:"rates"`
	PackageDetails       string                  `json:"package_details"`
	Index                int                     `json:"index"`
	ChosenMethod         string                  `json:"chosen_method"`
	FormattedDestination string                  `json:"formatted_destination"`
}

// CartShipping is the cart's shipping field: the package count, whether
// shipping has been calculated, and the packages keyed by package ID.
type CartShipping struct {
	TotalPackages         int                        `json:"total_packages"`
	ShowPackageDetails    bool                       `json:"show_package_details"`
	HasCalculatedShipping bool                       `json:"has_calculated_shipping"`
	Packages              ObjectMap[ShippingPackage] `json:"packages"`
}

// RemovedItem is an item removed from the cart that can be restored.
type RemovedItem struct {
	Key       string `json:"key"`
	ProductID int    `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Name      string `json:"name"`
}

// CrossSellProduct represents a cross-sell product recommendation.
type CrossSellProduct struct {
	ID           int    `json:"id"`
	ProductName  string `json:"product_name"`
	ProductTitle string `json:"product_title"`
	Price        string `json:"price"`
	RegularPrice string `json:"regular_price"`
	SalePrice    string `json:"sale_price"`
	ProductImage string `json:"product_image"`
}

// CartResponse represents the full cart (GET /cart, and the cart returned by
// cart actions).
type CartResponse struct {
	CartHash      string             `json:"cart_hash"`
	CartKey       string             `json:"cart_key"`
	Currency      CurrencyInfo       `json:"currency"`
	Customer      CartCustomer       `json:"customer"`
	Items         []CartItem         `json:"items"`
	ItemCount     int                `json:"item_count"`
	ItemsWeight   FlexString         `json:"items_weight"`
	Coupons       []CartCoupon       `json:"coupons"`
	NeedsPayment  bool               `json:"needs_payment"`
	NeedsShipping bool               `json:"needs_shipping"`
	Shipping      CartShipping       `json:"shipping"`
	Fees          ObjectMap[CartFee] `json:"fees"`
	Taxes         CartTaxes          `json:"taxes"`
	Totals        CartTotals         `json:"totals"`
	RemovedItems  []RemovedItem      `json:"removed_items"`
	CrossSells    []CrossSellProduct `json:"cross_sells"`
	// Notices are keyed by type (success, error, notice, info).
	Notices ObjectMap[[]string] `json:"notices"`
	// Extensions holds data plugins add to the cart (see docs/extension-data.md).
	Extensions ObjectMap[any] `json:"extensions"`
	Meta       ObjectMap[any] `json:"meta"`
}

// --- Products ---

// ProductImage represents a product image. Src maps image size names to URLs
// (thumbnail, woocommerce_single, full, …).
type ProductImage struct {
	ID       int               `json:"id"`
	Src      ObjectMap[string] `json:"src"`
	Name     string            `json:"name"`
	Alt      string            `json:"alt"`
	Position int               `json:"position"`
	Featured bool              `json:"featured"`
}

// ProductCategory is a category as embedded in a product.
type ProductCategory struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	RestURL string `json:"rest_url"`
}

// ProductCategoryDetail is a category as GET /products/categories returns it.
type ProductCategoryDetail struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	Count       int            `json:"count"`
	MenuOrder   int            `json:"menu_order"`
	ParentID    int            `json:"parent_id"`
	Display     string         `json:"display"`
	Image       ObjectMap[any] `json:"image"`
}

// ProductTag is a tag as embedded in a product.
type ProductTag struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	RestURL string `json:"rest_url"`
}

// ProductBrand is a brand as embedded in a product.
type ProductBrand struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Count       int    `json:"count"`
	RestURL     string `json:"rest_url"`
}

// ProductAttribute represents a product attribute as listed by
// GET /products/attributes.
type ProductAttribute struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	Position  int      `json:"position"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
	Options   []string `json:"options"`
}

// ProductAttributeDetail is one of a product's attributes. A product's
// attributes are keyed by attribute (attribute_pa_color), and Options maps
// each term's slug to its name.
type ProductAttributeDetail struct {
	ID                 int               `json:"id"`
	Name               string            `json:"name"`
	Position           int               `json:"position"`
	IsAttributeVisible bool              `json:"is_attribute_visible"`
	UsedForVariation   bool              `json:"used_for_variation"`
	Options            ObjectMap[string] `json:"options"`
}

// ProductDefaultAttribute is a variable product's default choice for one attribute.
type ProductDefaultAttribute struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Option string `json:"option"`
}

// ProductPriceRange represents the lowest and highest price of a variable
// product, in minor units. Empty for other products.
type ProductPriceRange struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// UnmarshalJSON accepts an object, or the [] CoCart 4.x sends when there's no range.
func (p *ProductPriceRange) UnmarshalJSON(b []byte) error {
	type plain ProductPriceRange
	return decodeObjectOrEmpty(b, (*plain)(p))
}

// ProductDateOnSale holds a scheduled sale's dates; empty when none is set.
type ProductDateOnSale struct {
	From    string `json:"from"`
	FromGMT string `json:"from_gmt"`
	To      string `json:"to"`
	ToGMT   string `json:"to_gmt"`
}

// ProductPrices contains a product's prices in minor-unit strings ("1800" for 18.00).
type ProductPrices struct {
	Price        string             `json:"price"`
	RegularPrice string             `json:"regular_price"`
	SalePrice    string             `json:"sale_price"`
	PriceRange   *ProductPriceRange `json:"price_range,omitempty"`
	OnSale       bool               `json:"on_sale"`
	DateOnSale   ProductDateOnSale  `json:"date_on_sale"`
	Currency     CurrencyInfo       `json:"currency"`
}

// ProductStock contains stock information.
type ProductStock struct {
	IsInStock         bool        `json:"is_in_stock"`
	StockQuantity     *int        `json:"stock_quantity"`
	StockStatus       StockStatus `json:"stock_status"`
	Backorders        string      `json:"backorders"`
	BackordersAllowed bool        `json:"backorders_allowed"`
	Backordered       bool        `json:"backordered"`
	LowStockAmount    FlexString  `json:"low_stock_amount"`
}

// ProductConditions contains purchasability conditions.
//
// Deprecated: CoCart doesn't send a conditions field. Use
// Product.AddToCart.IsPurchasable and Product.HiddenConditions.
type ProductConditions struct {
	IsPurchasable bool `json:"is_purchasable"`
}

// ProductHiddenConditions contains a product's settings that affect buying it.
type ProductHiddenConditions struct {
	Virtual          bool `json:"virtual"`
	Downloadable     bool `json:"downloadable"`
	ManageStock      bool `json:"manage_stock"`
	SoldIndividually bool `json:"sold_individually"`
	ReviewsAllowed   bool `json:"reviews_allowed"`
	ShippingRequired bool `json:"shipping_required"`
}

// PurchaseQuantity holds the quantity limits for adding a product to the cart.
type PurchaseQuantity struct {
	Minimum    int `json:"minimum,omitempty"`
	Maximum    int `json:"maximum,omitempty"`
	MultipleOf int `json:"multiple_of,omitempty"`
}

// UnmarshalJSON accepts an object, or the [] CoCart 4.x sends when it's empty.
func (q *PurchaseQuantity) UnmarshalJSON(b []byte) error {
	type plain PurchaseQuantity
	return decodeObjectOrEmpty(b, (*plain)(q))
}

// ProductAddToCart describes how to add a product to the cart. RestURL is a
// ready-made add-to-cart request, empty when the product needs options first.
type ProductAddToCart struct {
	Text             string           `json:"text"`
	Description      string           `json:"description"`
	HasOptions       bool             `json:"has_options"`
	IsPurchasable    bool             `json:"is_purchasable"`
	PurchaseQuantity PurchaseQuantity `json:"purchase_quantity"`
	RestURL          string           `json:"rest_url"`
}

// ProductDates holds a product's creation and modification dates.
type ProductDates struct {
	Created     string `json:"created"`
	CreatedGMT  string `json:"created_gmt"`
	Modified    string `json:"modified"`
	ModifiedGMT string `json:"modified_gmt"`
}

// ProductWeight holds a product's weight and its unit.
type ProductWeight struct {
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

// ProductDimensions holds a product's dimensions and their unit.
type ProductDimensions struct {
	Length string `json:"length"`
	Width  string `json:"width"`
	Height string `json:"height"`
	Unit   string `json:"unit"`
}

// ProductMetaData is a product meta entry.
type ProductMetaData struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// ConnectedProduct is a related product, upsell or cross-sell embedded in a product.
type ConnectedProduct struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Permalink string `json:"permalink"`
	Price     string `json:"price"`
	AddToCart struct {
		Text        string `json:"text"`
		Description string `json:"description"`
		RestURL     string `json:"rest_url"`
	} `json:"add_to_cart"`
	RestURL string `json:"rest_url"`
}

// ProductVariationSummary is a variation as embedded in a variable product.
type ProductVariationSummary struct {
	ID          int    `json:"id"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	// Attributes maps each attribute to the variation's term slug, e.g.
	// {"attribute_pa_color": "blue"}.
	Attributes ObjectMap[string] `json:"attributes"`
	// FeaturedImage maps image size names to URLs.
	FeaturedImage ObjectMap[string] `json:"featured_image"`
	Prices        ProductPrices     `json:"prices"`
	Stock         ProductStock      `json:"stock"`
	AddToCart     ProductAddToCart  `json:"add_to_cart"`
}

// Product represents a single product.
type Product struct {
	ID               int                     `json:"id"`
	ParentID         int                     `json:"parent_id"`
	Name             string                  `json:"name"`
	Type             string                  `json:"type"`
	Slug             string                  `json:"slug"`
	Permalink        string                  `json:"permalink"`
	SKU              string                  `json:"sku"`
	Description      string                  `json:"description"`
	ShortDescription string                  `json:"short_description"`
	Dates            ProductDates            `json:"dates"`
	Featured         bool                    `json:"featured"`
	Prices           ProductPrices           `json:"prices"`
	HiddenConditions ProductHiddenConditions `json:"hidden_conditions"`
	AverageRating    string                  `json:"average_rating"`
	ReviewCount      int                     `json:"review_count"`
	RatingCount      int                     `json:"rating_count"`
	RatedOutOf       string                  `json:"rated_out_of"`
	Images           []ProductImage          `json:"images"`
	Categories       []ProductCategory       `json:"categories"`
	Tags             []ProductTag            `json:"tags"`
	// Attributes are keyed by attribute (attribute_pa_color).
	Attributes        ObjectMap[ProductAttributeDetail]  `json:"attributes"`
	DefaultAttributes ObjectMap[ProductDefaultAttribute] `json:"default_attributes"`
	Variations        []ProductVariationSummary          `json:"variations"`
	GroupedProducts   []int                              `json:"grouped_products"`
	Stock             ProductStock                       `json:"stock"`
	Weight            ProductWeight                      `json:"weight"`
	Dimensions        ProductDimensions                  `json:"dimensions"`
	Reviews           []ProductReview                    `json:"reviews"`
	Related           []ConnectedProduct                 `json:"related"`
	Upsells           []ConnectedProduct                 `json:"upsells"`
	CrossSells        []ConnectedProduct                 `json:"cross_sells"`
	TotalSales        FlexString                         `json:"total_sales"`
	ExternalURL       string                             `json:"external_url"`
	ButtonText        string                             `json:"button_text"`
	AddToCart         ProductAddToCart                   `json:"add_to_cart"`
	MetaData          []ProductMetaData                  `json:"meta_data"`
	GlobalUniqueID    string                             `json:"global_unique_id"`
	Brands            []ProductBrand                     `json:"brands"`
	// Extensions holds data plugins add to the product (see docs/extension-data.md).
	Extensions ObjectMap[any] `json:"extensions"`
}

// ProductsResponse is the product list GET /products returns.
type ProductsResponse struct {
	Products      []Product `json:"products"`
	Page          int       `json:"page"`
	TotalPages    int       `json:"total_pages"`
	TotalProducts int       `json:"total_products"`
}

// ProductVariationAttribute is one of a variation's attributes. Option maps
// the chosen term's slug to its name.
type ProductVariationAttribute struct {
	ID     int               `json:"id"`
	Name   string            `json:"name"`
	Option ObjectMap[string] `json:"option"`
}

// ProductVariation represents a product variation (GET /products/{id}/variations).
type ProductVariation struct {
	ID               int                                  `json:"id"`
	ParentID         int                                  `json:"parent_id"`
	Name             string                               `json:"name"`
	Slug             string                               `json:"slug"`
	Permalink        string                               `json:"permalink"`
	SKU              string                               `json:"sku"`
	Description      string                               `json:"description"`
	Dates            ProductDates                         `json:"dates"`
	Featured         bool                                 `json:"featured"`
	Prices           ProductPrices                        `json:"prices"`
	HiddenConditions ProductHiddenConditions              `json:"hidden_conditions"`
	Images           []ProductImage                       `json:"images"`
	Categories       []ProductCategory                    `json:"categories"`
	Tags             []ProductTag                         `json:"tags"`
	Attributes       ObjectMap[ProductVariationAttribute] `json:"attributes"`
	Stock            ProductStock                         `json:"stock"`
	Weight           ProductWeight                        `json:"weight"`
	Dimensions       ProductDimensions                    `json:"dimensions"`
	TotalSales       FlexString                           `json:"total_sales"`
	AddToCart        ProductAddToCart                     `json:"add_to_cart"`
	MetaData         []ProductMetaData                    `json:"meta_data"`
	GlobalUniqueID   string                               `json:"global_unique_id"`
	Brands           []ProductBrand                       `json:"brands"`
}

// ProductReview represents a product review.
type ProductReview struct {
	ID             int    `json:"id"`
	DateCreated    string `json:"date_created"`
	DateCreatedGMT string `json:"date_created_gmt"`
	ProductID      int    `json:"product_id"`
	Status         string `json:"status"`
	Reviewer       string `json:"reviewer"`
	ReviewerEmail  string `json:"reviewer_email"`
	Review         string `json:"review"`
	Rating         int    `json:"rating"`
	Verified       bool   `json:"verified"`
}

// --- Store and sessions ---

// StoreInfo represents the store information response (GET /store).
type StoreInfo struct {
	Title          string            `json:"title"`
	Description    string            `json:"description"`
	HomeURL        string            `json:"home_url"`
	Language       string            `json:"language"`
	GMTOffset      string            `json:"gmt_offset"`
	TimezoneString string            `json:"timezone_string"`
	StoreAddress   ObjectMap[string] `json:"store_address"`
	Routes         ObjectMap[any]    `json:"routes"`
	// Versions holds the plugin versions: core, and plus with CoCart Plus.
	Versions ObjectMap[string] `json:"versions"`
}

// SessionItem represents a session in the admin sessions list.
type SessionItem struct {
	CartID         int    `json:"cart_id"`
	CartKey        string `json:"cart_key"`
	CustomersName  string `json:"customers_name"`
	CustomersEmail string `json:"customers_email"`
	CartCreated    string `json:"cart_created"`
	CartExpiry     string `json:"cart_expiry"`
	CartSource     string `json:"cart_source"`
	Link           string `json:"link"`
}

// RateLimitInfo holds the RateLimit-* headers CoCart Plus sends when rate
// limiting is on. A field is nil when its header wasn't sent.
type RateLimitInfo struct {
	// Limit is the number of requests allowed in the window.
	Limit *int
	// Remaining is the number of requests left in the window.
	Remaining *int
	// Reset is the Unix time the window resets.
	Reset *int
	// RetryAfter is the number of seconds to wait after a 429.
	RetryAfter *int
}

// --- Parameters ---

// PaginationParams contains common pagination parameters.
type PaginationParams struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// ProductListParams contains parameters for listing products.
type ProductListParams struct {
	Page        int            `url:"page,omitempty"`
	PerPage     int            `url:"per_page,omitempty"`
	Search      string         `url:"search,omitempty"`
	OrderBy     ProductOrderBy `url:"orderby,omitempty"`
	Order       SortOrder      `url:"order,omitempty"`
	Category    string         `url:"category,omitempty"`
	Tag         string         `url:"tag,omitempty"`
	Brand       string         `url:"brand,omitempty"`
	Featured    bool           `url:"featured,omitempty"`
	OnSale      bool           `url:"on_sale,omitempty"`
	StockStatus StockStatus    `url:"stock_status,omitempty"`
	MinPrice    string         `url:"min_price,omitempty"`
	MaxPrice    string         `url:"max_price,omitempty"`
	Fields      string         `url:"_fields,omitempty"`
}

// ProductParams contains parameters for getting a single product.
type ProductParams struct {
	Fields string `url:"_fields,omitempty"`
}

// ProductReviewInput is a review to post with [ProductsEndpoint.CreateReview].
// Guests give Reviewer and ReviewerEmail; stores that only allow verified
// owners check the email against past orders.
type ProductReviewInput struct {
	ProductID int    `json:"product_id"`
	Review    string `json:"review"`
	// Rating is 1–5.
	Rating        int    `json:"rating,omitempty"`
	Reviewer      string `json:"reviewer,omitempty"`
	ReviewerEmail string `json:"reviewer_email,omitempty"`
}

// ProductShippingRatesParams contains the options for [ProductsEndpoint.ShippingRates].
type ProductShippingRatesParams struct {
	// Quantity to price shipping for. Default 1.
	Quantity int
	// Variation holds a variable product's attributes, e.g. {"attribute_pa_color": "blue"}.
	Variation map[string]string
	// ItemData holds custom item data, as sent when adding the item.
	ItemData map[string]string
}

// CartGetParams contains parameters for getting the cart.
type CartGetParams struct {
	Fields  string `url:"_fields,omitempty"`
	CartKey string `url:"cart_key,omitempty"`
}

// CartItemData contains data for adding an item to cart.
type CartItemData struct {
	ID        string            `json:"id"`
	Quantity  string            `json:"quantity"`
	Variation map[string]string `json:"variation,omitempty"`
	ItemData  map[string]any    `json:"item_data,omitempty"`
}

// BatchRequestItem represents a single sub-request queued for [Client.Batch].
type BatchRequestItem struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   any    `json:"body,omitempty"`
}

// authCredentials holds basic auth credentials (internal).
type authCredentials struct {
	username string
	password string
}

// structToParams converts a struct with `url` tags to a map[string]string.
// Fields tagged with `omitempty` are skipped when they have zero values.
func structToParams(v any) map[string]string {
	return structToParamsReflect(v)
}
