package cocart

// MainPlugin identifies which CoCart plugin variant is installed.
type MainPlugin string

const (
	MainPluginBasic  MainPlugin = "basic"
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

// ProductOrderBy represents valid product sort fields.
type ProductOrderBy string

const (
	OrderByDate       ProductOrderBy = "date"
	OrderByID         ProductOrderBy = "id"
	OrderByInclude    ProductOrderBy = "include"
	OrderByTitle      ProductOrderBy = "title"
	OrderBySlug       ProductOrderBy = "slug"
	OrderByPrice      ProductOrderBy = "price"
	OrderByPopularity ProductOrderBy = "popularity"
	OrderByRating     ProductOrderBy = "rating"
)

// CurrencyInfo contains currency metadata from the API.
type CurrencyInfo struct {
	CurrencyCode          string `json:"currency_code"`
	CurrencySymbol        string `json:"currency_symbol"`
	CurrencyMinorUnit     int    `json:"currency_minor_unit"`
	CurrencyDecimalSep    string `json:"currency_decimal_separator"`
	CurrencyThousandSep   string `json:"currency_thousand_separator"`
	CurrencyPrefix        string `json:"currency_prefix"`
	CurrencySuffix        string `json:"currency_suffix"`
}

// CartItemQuantity describes quantity constraints for a cart item.
type CartItemQuantity struct {
	Value      int  `json:"value"`
	Minimum    int  `json:"minimum"`
	Maximum    int  `json:"maximum"`
	MultipleOf int  `json:"multiple_of"`
	Editable   bool `json:"editable"`
}

// CartItemTotals contains price totals for a single cart item.
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
	Weight      float64           `json:"weight"`
}

// CartItemImage represents a cart item image.
type CartItemImage struct {
	ID        int    `json:"id"`
	Src       string `json:"src"`
	Thumbnail string `json:"thumbnail"`
	Name      string `json:"name"`
	Alt       string `json:"alt"`
}

// CartItem represents a single item in the cart.
type CartItem struct {
	ItemKey       string            `json:"item_key"`
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	Title         string            `json:"title"`
	Price         string            `json:"price"`
	Quantity      CartItemQuantity  `json:"quantity"`
	Totals        CartItemTotals    `json:"totals"`
	Slug          string            `json:"slug"`
	Meta          CartItemMeta      `json:"meta"`
	Backorders    string            `json:"backorders"`
	CartItemData  map[string]any    `json:"cart_item_data"`
	FeaturedImage string            `json:"featured_image"`
}

// CartTotals contains the cart's price totals.
type CartTotals struct {
	Subtotal     string `json:"subtotal"`
	SubtotalTax  string `json:"subtotal_tax"`
	FeeTotal     string `json:"fee_total"`
	FeeTax       string `json:"fee_tax"`
	DiscountTotal string `json:"discount_total"`
	DiscountTax  string `json:"discount_tax"`
	ShippingTotal string `json:"shipping_total"`
	ShippingTax  string `json:"shipping_tax"`
	Total        string `json:"total"`
	TotalTax     string `json:"total_tax"`
}

// CustomerAddress represents a billing or shipping address.
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

// CartCustomer contains the cart customer's addresses.
type CartCustomer struct {
	BillingAddress  CustomerAddress `json:"billing_address"`
	ShippingAddress CustomerAddress `json:"shipping_address"`
}

// CartCoupon represents an applied coupon.
type CartCoupon struct {
	Coupon     string `json:"coupon"`
	Label      string `json:"label"`
	Saving     string `json:"saving"`
	SavingHTML string `json:"saving_html"`
}

// CartFee represents a cart fee.
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

// ShippingRate represents a shipping rate option.
type ShippingRate struct {
	Key        string         `json:"key"`
	MethodID   string         `json:"method_id"`
	InstanceID int            `json:"instance_id"`
	Label      string         `json:"label"`
	Cost       string         `json:"cost"`
	Tax        string         `json:"tax"`
	MetaData   map[string]any `json:"meta_data"`
}

// ShippingPackage represents a shipping package with available rates.
type ShippingPackage struct {
	PackageName  string                  `json:"package_name"`
	Rates        map[string]ShippingRate `json:"rates"`
	ChosenMethod string                  `json:"chosen_method"`
}

// CrossSellProduct represents a cross-sell product recommendation.
type CrossSellProduct struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Price         string `json:"price"`
	RegularPrice  string `json:"regular_price"`
	SalePrice     string `json:"sale_price"`
	FeaturedImage string `json:"featured_image"`
}

// CartResponse represents the full cart API response.
type CartResponse struct {
	CartHash      string            `json:"cart_hash"`
	CartKey       string            `json:"cart_key"`
	Currency      CurrencyInfo      `json:"currency"`
	Customer      CartCustomer      `json:"customer"`
	Items         []CartItem        `json:"items"`
	ItemCount     int               `json:"item_count"`
	ItemsWeight   float64           `json:"items_weight"`
	Coupons       []CartCoupon      `json:"coupons"`
	NeedsPayment  bool              `json:"needs_payment"`
	NeedsShipping bool              `json:"needs_shipping"`
	Shipping      []ShippingPackage `json:"shipping"`
	Fees          []CartFee         `json:"fees"`
	Taxes         map[string]any    `json:"taxes"`
	Totals        CartTotals        `json:"totals"`
	RemovedItems  []CartItem        `json:"removed_items"`
	CrossSells    []CrossSellProduct `json:"cross_sells"`
	Notices       []any             `json:"notices"`
}

// ProductImage represents a product image.
type ProductImage struct {
	ID        int    `json:"id"`
	Src       string `json:"src"`
	Thumbnail string `json:"thumbnail"`
	Name      string `json:"name"`
	Alt       string `json:"alt"`
	Position  int    `json:"position"`
}

// ProductCategory represents a product category reference.
type ProductCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ProductTag represents a product tag reference.
type ProductTag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ProductAttribute represents a product attribute.
type ProductAttribute struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Slug      string   `json:"slug"`
	Position  int      `json:"position"`
	Visible   bool     `json:"visible"`
	Variation bool     `json:"variation"`
	Options   []string `json:"options"`
}

// ProductPriceRange represents min/max price for variable products.
type ProductPriceRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ProductPrices contains product pricing information.
type ProductPrices struct {
	Price        string             `json:"price"`
	RegularPrice string            `json:"regular_price"`
	SalePrice    string            `json:"sale_price"`
	PriceRange   *ProductPriceRange `json:"price_range,omitempty"`
	OnSale       bool              `json:"on_sale"`
}

// ProductStock contains stock information.
type ProductStock struct {
	StockQuantity *int        `json:"stock_quantity"`
	StockStatus   StockStatus `json:"stock_status"`
}

// ProductConditions contains purchasability conditions.
type ProductConditions struct {
	IsPurchasable bool `json:"is_purchasable"`
}

// Product represents a single product.
type Product struct {
	ID               int                `json:"id"`
	Name             string             `json:"name"`
	Slug             string             `json:"slug"`
	Permalink        string             `json:"permalink"`
	Type             string             `json:"type"`
	Description      string             `json:"description"`
	ShortDescription string             `json:"short_description"`
	SKU              string             `json:"sku"`
	Prices           ProductPrices      `json:"prices"`
	Images           []ProductImage     `json:"images"`
	Categories       []ProductCategory  `json:"categories"`
	Tags             []ProductTag       `json:"tags"`
	Attributes       []ProductAttribute `json:"attributes"`
	Stock            ProductStock       `json:"stock"`
	Conditions       ProductConditions  `json:"conditions"`
	Featured         bool               `json:"featured"`
	AverageRating    string             `json:"average_rating"`
	ReviewCount      int                `json:"review_count"`
}

// ProductVariation represents a product variation.
type ProductVariation struct {
	ID          int               `json:"id"`
	SKU         string            `json:"sku"`
	Description string            `json:"description"`
	Prices      ProductPrices     `json:"prices"`
	Attributes  map[string]string `json:"attributes"`
	Stock       ProductStock      `json:"stock"`
	Images      []ProductImage    `json:"images"`
}

// ProductReview represents a product review.
type ProductReview struct {
	ID            int    `json:"id"`
	ProductID     int    `json:"product_id"`
	Reviewer      string `json:"reviewer"`
	ReviewerEmail string `json:"reviewer_email"`
	Review        string `json:"review"`
	Rating        int    `json:"rating"`
	Verified      bool   `json:"verified"`
	DateCreated   string `json:"date_created"`
}

// StoreInfo represents the store information response.
type StoreInfo struct {
	StoreName        string         `json:"store_name"`
	StoreDescription string         `json:"store_description"`
	StoreURL         string         `json:"store_url"`
	Routes           map[string]any `json:"routes"`
}

// SessionItem represents a session in the admin sessions list.
type SessionItem struct {
	SessionKey   string         `json:"session_key"`
	SessionValue map[string]any `json:"session_value"`
	SessionExpiry string        `json:"session_expiry"`
}

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
