# Products API

The Products API lets you browse your store's catalog — listing products, searching, filtering by category or price, and reading product details. It is publicly accessible and does not require authentication, just like a customer browsing your store's shelves.

```go
products := client.Products()
```

All product methods take `context.Context` as their first parameter.

## List Products

```go
response, err := client.Products().All(ctx, nil)
response, err := client.Products().All(ctx, &cocart.ProductListParams{
	PerPage: 20,
	Page:    1,
})

// The list is under `products`, with the page count alongside
list, err := cocart.Unmarshal[cocart.ProductsResponse](response)
for _, product := range list.Products {
	fmt.Println(product.Name, product.Prices.Price) // price in minor units, e.g. "1800"
}
fmt.Println("Pages:", list.TotalPages)
```

Product responses carry a lot for headless pages (related products, embedded variations with price and stock, upsells, cross-sells, stock, extension meta). For a listing that needs less, ask for just those fields; CoCart then skips building the rest:

```go
response, err := client.Products().All(ctx, &cocart.ProductListParams{
	PerPage: 20,
	Fields:  "id,name,slug,prices,images,add_to_cart",
})
```

Prices are minor-unit strings under `prices` (`"1800"` for 18.00; see `CurrencyFormatter` in [Utilities](utilities.md)), an image's `Src` maps size names to URLs (`thumbnail`, `medium`, `full`…), and `Attributes` is keyed by attribute (`attribute_pa_color`).

## Parameters Reference

All list methods accept an optional `*ProductListParams` struct with these fields:

| Field | Type | Description |
|-------|------|-------------|
| `Page` | `int` | Page number (default: 1) |
| `PerPage` | `int` | Items per page (default: 10, max: 100) |
| `Search` | `string` | Search term |
| `Category` | `string` | Filter by category slug |
| `Tag` | `string` | Filter by tag slug |
| `Featured` | `bool` | Show only featured products |
| `OnSale` | `bool` | Show only products on sale |
| `MinPrice` | `string` | Minimum price |
| `MaxPrice` | `string` | Maximum price |
| `StockStatus` | `StockStatus` | Stock status (`StockInStock`, `StockOutOfStock`, `StockOnBackorder`) |
| `OrderBy` | `ProductOrderBy` | Sort order: `OrderByDate`, `OrderByID`, `OrderByMenuOrder`, `OrderByInclude`, `OrderByTitle`, `OrderBySlug`, `OrderByName`, `OrderByAlphabetical`, `OrderByReverseAlpha`, `OrderByPopularity`, `OrderBySales`, `OrderByRating`, `OrderByReviewCount`, `OrderByStock`, `OrderByOnSaleFirst`, `OrderByFeaturedFirst`, `OrderByPriceAsc`, `OrderByPriceDesc`; `OrderByRelevance` and `OrderByRand` need CoCart Starter |
| `Order` | `SortOrder` | Sort direction (`SortAsc`, `SortDesc`) |

## Filtering

### By Category

```go
response, err := client.Products().ByCategory(ctx, "electronics", nil)

// With additional parameters
response, err := client.Products().ByCategory(ctx, "electronics", &cocart.ProductListParams{
	PerPage: 20,
	OrderBy: cocart.OrderByPriceAsc,
	Order:   cocart.SortAsc,
})
```

### By Tag

```go
response, err := client.Products().ByTag(ctx, "new-arrival", nil)
```

### Featured Products

```go
response, err := client.Products().Featured(ctx, nil)
response, err := client.Products().Featured(ctx, &cocart.ProductListParams{PerPage: 4})
```

### Products on Sale

```go
response, err := client.Products().OnSale(ctx, nil)
```

### By Price Range

```go
// Products between $10 and $50
response, err := client.Products().ByPriceRange(ctx, "10", "50")

// Products under $25
response, err := client.Products().ByPriceRange(ctx, "", "25")

// Products over $100
response, err := client.Products().ByPriceRange(ctx, "100", "")
```

### Search

```go
response, err := client.Products().Search(ctx, "wireless headphones", nil)

// Search within a category
response, err := client.Products().Search(ctx, "headphones", &cocart.ProductListParams{
	Category: "electronics",
})
```

### Combining Filters

```go
response, err := client.Products().All(ctx, &cocart.ProductListParams{
	Category:    "clothing",
	OnSale:      true,
	MinPrice:    "20",
	MaxPrice:    "100",
	OrderBy:     cocart.OrderByPopularity,
	Order:       cocart.SortDesc,
	PerPage:     12,
})
```

### By Stock Status

```go
response, err := client.Products().ByStockStatus(ctx, cocart.StockInStock)
response, err := client.Products().ByStockStatus(ctx, cocart.StockOutOfStock)
response, err := client.Products().ByStockStatus(ctx, cocart.StockOnBackorder)
```

## Pagination & Sorting

**Pagination** is how the API delivers large sets of results in smaller chunks. If your store has 500 products, you request page 1 (products 1-20), then page 2 (products 21-40), and so on. **Sorting** controls the order results come back in.

### Paginate Helper

```go
// Page 1, 12 products per page
response, err := client.Products().Paginate(ctx, 1, 12, nil)

// Page 2
response, err := client.Products().Paginate(ctx, 2, 12, nil)
```

### Sort Helper

```go
// Cheapest first (CoCart rejects plain "price": use OrderByPriceAsc or OrderByPriceDesc)
response, err := client.Products().SortBy(ctx, cocart.OrderByPriceAsc, cocart.SortAsc)

// Most expensive first
response, err := client.Products().SortBy(ctx, cocart.OrderByPriceDesc, cocart.SortDesc)

// Newest first
response, err := client.Products().SortBy(ctx, cocart.OrderByDate, cocart.SortDesc)

// Most popular
response, err := client.Products().SortBy(ctx, cocart.OrderByPopularity, cocart.SortDesc)
```

### Paginated Iterator (Go 1.23+)

`AllPaginated()` returns a `*Paginator` that you can iterate with Go 1.23's range-over-func. It automatically fetches the next page when the current one is done, and stops when there are no more pages:

```go
paginator := client.Products().AllPaginated(&cocart.ProductListParams{PerPage: 20})

for resp, err := range paginator.All(ctx) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Page data:", string(resp.Body))
}

// Or collect all pages into a slice
pages, err := paginator.Collect(ctx)
```

## Single Product

### By ID or SKU

`Find()` accepts either the numeric product/variation ID or the product's SKU — either one works the same way.

```go
response, err := client.Products().Find(ctx, 123)

// Or by SKU
response, err := client.Products().Find(ctx, "PCT-2024")

name := response.Get("name", "")
price := response.Get("prices.price", "") // minor units, e.g. "1800"
description := response.Get("description", "")

// Or as a typed struct
product, err := cocart.Unmarshal[cocart.Product](response)
fmt.Println(product.Prices.Price, product.Images[0].Src["thumbnail"])
```

### By Slug

> Requires CoCart Starter.

```go
response, err := client.Products().FindBySlug(ctx, "blue-hoodie")
```

## Variations

**Variations** are the specific versions of a variable product. For example, a T-shirt product might have variations for "Red / Small", "Red / Large", "Blue / Small", etc.

### List All Variations

```go
response, err := client.Products().Variations(ctx, 123, nil)
variations, err := cocart.Unmarshal[[]cocart.ProductVariation](response)
```

### Get a Specific Variation

> Requires CoCart Starter.

```go
response, err := client.Products().Variation(ctx, 123, 456)
```

## Categories

### List All Categories

```go
response, err := client.Products().Categories(ctx, nil)
response, err := client.Products().Categories(ctx, &cocart.PaginationParams{PerPage: 50})
```

### Get a Single Category

> Requires CoCart Starter.

```go
response, err := client.Products().Category(ctx, 15)
```

## Tags

### List All Tags

```go
response, err := client.Products().Tags(ctx, nil)
```

### Get a Single Tag

> Requires CoCart Starter.

```go
response, err := client.Products().Tag(ctx, 8)
```

## Attributes

**Attributes** are the properties that define product variations — things like "Color", "Size", or "Material". Each attribute has **terms** (the specific values), such as "Red", "Blue", "Green" for a "Color" attribute.

### List All Attributes

```go
response, err := client.Products().Attributes(ctx, nil)
```

### Get a Single Attribute

```go
response, err := client.Products().Attribute(ctx, 1)
```

### Get Attribute Terms

```go
// Get all terms for attribute ID 1 (e.g., all colors)
response, err := client.Products().AttributeTerms(ctx, 1, nil)
```

### Get a Single Attribute Term

```go
// Get term ID 5 for attribute ID 1
response, err := client.Products().AttributeTerm(ctx, 1, 5)
```

### Slug-Based Attribute Lookups

> Requires CoCart Starter.

```go
// Get attribute by slug
response, err := client.Products().AttributeBySlug(ctx, "color")

// Get terms for attribute by slug
response, err := client.Products().AttributeTermsBySlug(ctx, "color", nil)

// Get a specific term by slug for an attribute by slug
response, err := client.Products().AttributeTermBySlug(ctx, "color", "red")
```

## Brands

> Requires CoCart Starter.

### List All Brands

```go
response, err := client.Products().Brands(ctx, nil)
response, err := client.Products().Brands(ctx, &cocart.PaginationParams{PerPage: 50})
```

### Get a Single Brand

```go
response, err := client.Products().Brand(ctx, 5)
```

### Filter Products by Brand

```go
response, err := client.Products().ByBrand(ctx, "nike", nil)

// With additional parameters
response, err := client.Products().ByBrand(ctx, "nike", &cocart.ProductListParams{
	PerPage: 20,
	OrderBy: cocart.OrderByPriceAsc,
	Order:   cocart.SortAsc,
})
```

## Reviews

### List All Reviews

```go
response, err := client.Products().Reviews(ctx, nil)
```

### Reviews for a Specific Product

```go
response, err := client.Products().ProductReviews(ctx, 123, nil)
```

### Post a Review

```go
response, err := client.Products().CreateReview(ctx, cocart.ProductReviewInput{
	ProductID:     123,
	Review:        "Warm and well made.",
	Rating:        5,
	Reviewer:      "Jane", // guests
	ReviewerEmail: "jane@example.com",
})
```

### My Reviews

> Requires CoCart Starter.

Get reviews written by the authenticated user:

```go
response, err := client.Products().MyReviews(ctx, nil)
```

## Shipping Rates for a Product

> Requires the CoCart Plus plugin.

Shipping rates for a product before it's added to the cart, priced for the store's (or the customer's) location. Pass a variation ID, or the parent with the variation's attributes:

```go
response, err := client.Products().ShippingRates(ctx, 154, cocart.ProductShippingRatesParams{Quantity: 2})
response, err := client.Products().ShippingRates(ctx, 134, cocart.ProductShippingRatesParams{
	Variation: map[string]string{"attribute_pa_color": "blue", "attribute_logo": "Yes"},
})

packages, _ := response.Get("packages").(map[string]any)
for _, pkg := range packages {
	rates, _ := pkg.(map[string]any)["rates"].(map[string]any)
	for id, rate := range rates {
		fmt.Println(id, rate.(map[string]any)["cost"]) // cost in minor units
	}
}
```

The request is sent without the cart key: the route prices a simulated cart and refuses one with `400 cocart_calculate_product_shipping_rates_cart_access_denied`. External and grouped products can't be priced (also a `400`).

## SEO (CoCart SEO Pack)

> Requires the CoCart SEO Pack plugin.

Get SEO metadata, Open Graph tags, Twitter Card data, and Schema.org structured data for products.

### By Product ID

```go
response, err := client.Products().SEO(ctx, 123)

fmt.Println(response.Get("provider", ""))          // "yoast", "rankmath", etc.
fmt.Println(response.Get("meta_data.title", ""))   // SEO title
fmt.Println(response.Get("open_graph.og:title", ""))
```

### By Product Slug

```go
response, err := client.Products().SEOBySlug(ctx, "blue-hoodie")
```

## Working with Responses

All methods return a `*Response`:

```go
response, err := client.Products().All(ctx, &cocart.ProductListParams{PerPage: 5})

// Check success
if response.IsSuccessful() {
	fmt.Println("Products:", response.Get("total_products", 0))
}

// Access nested data with dot notation
response, err := client.Products().Find(ctx, 123)
fmt.Println(response.Get("name", ""))
fmt.Println(response.Get("prices.price", ""))
fmt.Println(response.Get("categories.0.name", ""))

// Pagination helpers
fmt.Println("Total results:", response.GetTotalResults())
fmt.Println("Total pages:", response.GetTotalPages())
```

See [Error Handling](error-handling.md) for handling API errors.
