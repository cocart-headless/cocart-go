# Cart API

The Cart API handles all shopping cart operations — adding items, updating quantities, applying coupons, managing shipping, and more. This is the core of any headless WooCommerce storefront.

**How cart sessions work:**

- **Guest customers** — The first request creates a new guest session. The server returns a `Cart-Key` (a unique identifier like `guest_abc123`) which the SDK extracts and stores automatically. All subsequent requests use this key so the server knows which cart belongs to which visitor.
- **Authenticated customers** — The server identifies the cart by the WordPress user account. No cart key is needed because the server already knows who you are from your authentication credentials.

To access cart methods, call `client.Cart()`:

```go
cart := client.Cart()
```

All cart methods take `context.Context` as their first parameter for cancellation and timeout support.

## Create Cart

Create a new guest cart session without adding items. Only available for non-authenticated (guest) users.

> Requires CoCart Basic. Returns `*VersionError` if using legacy plugin.

```go
response, err := client.Cart().Create(ctx)
fmt.Println(response.Get("cart_key", "")) // "guest_abc123..."
```

## Get Cart

```go
response, err := client.Cart().Get(ctx, nil)

// With parameters
response, err := client.Cart().Get(ctx, &cocart.CartGetParams{
	Fields: "items,totals",
})
```

## Client-Side Validation

The SDK checks your inputs _before_ sending anything to the server. If you pass an invalid product ID (like `-1`) or a quantity of `0`, the SDK returns an error immediately without making a network request.

```go
_, err := client.Cart().AddItem(ctx, -1, 0)

var valErr *cocart.ValidationError
if errors.As(err, &valErr) {
	// valErr.Message => "Product ID must be a positive integer"
	// No network request was made
}
```

Validation rules:

- **Product ID** — Must be a positive integer (`AddItem`, `AddVariation`)
- **Quantity** — Must be a positive number (`AddItem`, `AddVariation`, `UpdateItem`)

You can also use the validation functions directly:

```go
err := cocart.ValidateProductID(123)          // nil
err := cocart.ValidateProductID(-1)           // *ValidationError
err := cocart.ValidateQuantity(2)             // nil
err := cocart.ValidateQuantity(0)             // *ValidationError
err := cocart.ValidateEmail("user@example.com") // nil
err := cocart.ValidateEmail("not-an-email")   // *ValidationError
```

## Adding Items

### Add a Simple Product

```go
// Product ID 123, quantity 2
response, err := client.Cart().AddItem(ctx, 123, 2)

// Shorthand alias
response, err := client.Cart().Add(ctx, 123, 2)
```

### Add with Options

```go
response, err := client.Cart().AddItem(ctx, 123, 1, map[string]any{
	"item_data": map[string]any{
		"gift_message": "Happy Birthday!",
		"engraving":    "John",
	},
	"email":       "customer@email.com",
	"return_item": true,
})
```

### Add a Variable Product

A **variable product** is a product with options like size or color. In WooCommerce, these are called "variations." When adding a variable product, you specify which variation the customer chose:

```go
response, err := client.Cart().AddVariation(ctx, 456, 1, map[string]string{
	"attribute_pa_color": "blue",
	"attribute_pa_size":  "large",
})
```

### Add Multiple Items at Once

```go
response, err := client.Cart().AddItems(ctx, []cocart.CartItemData{
	{ID: "123", Quantity: "2"},
	{ID: "456", Quantity: "1"},
	{ID: "789", Quantity: "3"},
})
```

## Updating Items

Every item in the cart has a unique **item key** — a string like `abc123def456...` that identifies that specific item. You receive item keys in cart responses (in the `item_key` field of each item).

```go
// Update quantity to 5
response, err := client.Cart().UpdateItem(ctx, "abc123def456...", 5)
```

### Update Multiple Items at Once

```go
response, err := client.Cart().UpdateItems(ctx, map[string]int{
	"abc123def456...": 3,
	"def789ghi012...": 1,
})
```

## Removing & Restoring Items

### Remove an Item

```go
response, err := client.Cart().RemoveItem(ctx, "abc123def456...")
```

### Remove Multiple Items at Once

```go
response, err := client.Cart().RemoveItems(ctx, []string{
	"abc123def456...",
	"def789ghi012...",
})
```

### Restore a Removed Item

```go
response, err := client.Cart().RestoreItem(ctx, "abc123def456...")
```

### Get Removed Items

```go
response, err := client.Cart().GetRemovedItems(ctx)
```

## Cart Management

### Clear Cart

```go
response, err := client.Cart().Clear(ctx)
```

### Calculate Totals

```go
response, err := client.Cart().Calculate(ctx)
```

### Update Cart

```go
response, err := client.Cart().Update(ctx, map[string]any{
	"customer_note": "Please gift wrap.",
})
```

## Totals & Counts

### Get Totals

```go
// Raw values
response, err := client.Cart().GetTotals(ctx, false)

// Formatted with currency (HTML)
response, err := client.Cart().GetTotals(ctx, true)
```

### Get Item Count

```go
response, err := client.Cart().GetItemCount(ctx)
```

### Get Cart Items

Get only the items in the cart (lighter than fetching the full cart):

```go
response, err := client.Cart().GetItems(ctx)
```

### Get a Single Cart Item

```go
response, err := client.Cart().GetItem(ctx, "abc123def456...")
```

## Coupons

> Requires the CoCart Plus plugin.

### Apply a Coupon

```go
response, err := client.Cart().ApplyCoupon(ctx, "SUMMER20")
```

### Remove a Coupon

```go
response, err := client.Cart().RemoveCoupon(ctx, "SUMMER20")
```

### Get Applied Coupons

```go
response, err := client.Cart().GetCoupons(ctx)
```

### Validate Applied Coupons

```go
response, err := client.Cart().CheckCoupons(ctx)
```

## Customer Details

### Update Customer

```go
// Update billing address
response, err := client.Cart().UpdateCustomer(ctx,
	map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "john@example.com",
		"phone":      "+1234567890",
		"address_1":  "123 Main St",
		"city":       "New York",
		"state":      "NY",
		"postcode":   "10001",
		"country":    "US",
	},
	nil, // no shipping update
)

// Update shipping address
response, err := client.Cart().UpdateCustomer(ctx,
	nil, // no billing update
	map[string]string{
		"first_name": "John",
		"last_name":  "Doe",
		"address_1":  "456 Oak Ave",
		"city":       "Los Angeles",
		"state":      "CA",
		"postcode":   "90001",
		"country":    "US",
	},
)

// Update both at once
response, err := client.Cart().UpdateCustomer(ctx,
	map[string]string{"email": "john@example.com"},
	map[string]string{"address_1": "456 Oak Ave"},
)
```

### Get Customer Details

```go
response, err := client.Cart().GetCustomer(ctx)
```

## Shipping

### Get Available Shipping Methods

```go
response, err := client.Cart().GetShippingMethods(ctx)
```

### Set Shipping Method

> Requires the CoCart Plus plugin.

```go
response, err := client.Cart().SetShippingMethod(ctx, "flat_rate:1")
```

### Calculate Shipping

```go
response, err := client.Cart().CalculateShipping(ctx, map[string]string{
	"country":  "US",
	"state":    "CA",
	"postcode": "90001",
	"city":     "Los Angeles",
})
```

## Fees

> Requires the CoCart Plus plugin.

### Get Cart Fees

```go
response, err := client.Cart().GetFees(ctx)
```

### Add a Fee

```go
// Non-taxable fee
response, err := client.Cart().AddFee(ctx, "Rush Processing", 9.99, false)

// Taxable fee
response, err := client.Cart().AddFee(ctx, "Gift Wrapping", 4.99, true)
```

### Remove All Fees

```go
response, err := client.Cart().RemoveFees(ctx)
```

## Cross-Sells

**Cross-sells** are product recommendations based on what's currently in the cart. These are configured in WooCommerce's product settings.

```go
response, err := client.Cart().GetCrossSells(ctx)
```

## ETag / Conditional Requests

**ETag** (Entity Tag) is a caching mechanism. When the server responds, it includes an `ETag` header — a unique fingerprint of the data. On the next request, the SDK automatically sends this fingerprint back via `If-None-Match`. If the data hasn't changed, the server responds with `304 Not Modified` (no body), saving bandwidth.

ETag support is **enabled by default**.

```go
// First request: full response with ETag header
response, err := client.Cart().Get(ctx, nil)

// Second request: sends If-None-Match automatically
response2, err := client.Cart().Get(ctx, nil)
if response2.IsNotModified() {
	fmt.Println("Cart has not changed")
}
```

### Disable ETag

```go
// Via constructor
client := cocart.NewClient("https://your-store.com",
	cocart.WithETag(false),
)

// At runtime
client.SetETag(false)
```

### Clear ETag Cache

```go
client.ClearETagCache()
```

## Working with Responses

Every cart method returns a `*Response` that wraps the server's reply. Use helper methods to access common data. The `Get()` method supports **dot notation** — a way to reach nested values using dots:

```go
response, err := client.Cart().Get(ctx, nil)

// Cart items
items := response.GetItems()

// Cart totals
totals := response.GetTotals()

// Item count
count := response.GetItemCount()

// Cart key (from headers)
cartKey := response.GetCartKey()

// Cart hash
hash := response.GetCartHash()

// Notices
notices := response.GetNotices()

// Dot-notation access
subtotal := response.Get("totals.subtotal", "0")
firstItemName := response.Get("items.0.name", "")

// Check if key exists
if response.Has("totals.discount_total") {
	fmt.Println("Discount applied!")
}

// Full data
data := response.ToJSON()
```

See [Error Handling](error-handling.md) for handling API errors.
