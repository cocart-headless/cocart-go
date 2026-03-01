# CoCart Go SDK

Official Go SDK for the [CoCart](https://cocartapi.com) REST API.

[![Tests](https://img.shields.io/github/actions/workflow/status/cocart-headless/cocart-go/tests.yml?label=tests&style=for-the-badge&labelColor=000000)](https://github.com/cocart-headless/cocart-go/actions/workflows/tests.yml)
[![License](https://img.shields.io/github/license/jayanratna/resend-php?color=9cf&style=for-the-badge&labelColor=000000)](https://github.com/cocart-headless/cocart-go/blob/main/LICENSE)

> [!IMPORTANT]
> This SDK is still in development and not yet ready for production use. Provide feedback if you experience a bug.

## TODO to complete the SDK

* [ ] Add SDK docs to documentation site
* [ ] Add support for Cart API extras
* [ ] Add Checkout API support
* [ ] Add Customers Account API support

---

## Features

- Zero external dependencies — uses only the Go standard library
- Functional options pattern for clean, flexible configuration
- `context.Context` on all IO methods for cancellation and timeouts
- Generic `Unmarshal[T]()` for typed response deserialization
- Client-side input validation (catches errors before network requests)
- Currency formatting and timezone utilities
- Event system for request/response lifecycle hooks
- Response transformer for custom processing
- Configurable auth header name (for proxies that strip `Authorization`)
- JWT authentication with auto-refresh
- ETag conditional requests for reduced bandwidth
- Retry with exponential backoff and `Retry-After` support
- Paginated iteration using Go 1.23 range-over-func (`iter.Seq2`)
- Legacy CoCart plugin support with version-aware endpoint guards

## Requirements

- **Go 1.23+** — Required for `iter.Seq2` (range-over-func) support used by the paginator.
- **CoCart plugin** installed on your WooCommerce store — This is the WordPress plugin that provides the REST API endpoints the SDK communicates with.
- [CoCart JWT Authentication](https://wordpress.org/plugins/cocart-jwt-authentication/) plugin for JWT features (optional) — Only needed if you want to use JSON Web Token authentication (explained in the [Authentication](docs/authentication.md) guide).

## Installation

```bash
go get github.com/cocart-headless/cocart-sdk-go
```

**Zero external dependencies** — the SDK uses only the Go standard library, keeping your dependency tree clean.

## Quick Start

An **SDK** (Software Development Kit) is a library that provides ready-made functions for talking to a specific service — in this case, the CoCart REST API on your WooCommerce store. Instead of writing raw HTTP requests yourself, you call simple methods like `client.Cart().AddItem(ctx, 123, 2)` and the SDK handles the details for you.

```go
package main

import (
	"context"
	"fmt"
	"log"

	cocart "github.com/cocart-headless/cocart-sdk-go"
)

func main() {
	ctx := context.Background()

	// Create a client pointing to your WooCommerce store
	client := cocart.NewClient("https://your-store.com")

	// Browse products (no auth required)
	products, err := client.Products().All(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Products:", string(products.Body))

	// Add to cart (guest session created automatically)
	response, err := client.Cart().AddItem(ctx, 123, 2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added item:", response.Get("item_key", ""))

	// Get cart
	cart, err := client.Cart().Get(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Items:", cart.GetItems())
	fmt.Println("Total:", cart.Get("totals.total", ""))
}
```

## Documentation

| Guide | Description |
|-------|-------------|
| [Configuration & Setup](docs/installation.md) | Options, fluent config, white-labelling |
| [Authentication](docs/authentication.md) | Guest, Basic Auth, JWT, consumer keys |
| [Cart API](docs/cart.md) | Add, update, remove items, coupons, shipping, fees |
| [Products API](docs/products.md) | List, filter, search, categories, tags, brands |
| [Store API](docs/store.md) | Store info, typed deserialization, field filtering |
| [Sessions API](docs/sessions.md) | Admin sessions, SessionManager, storage adapters |
| [Error Handling](docs/error-handling.md) | Error hierarchy, catching errors, common scenarios |
| [Utilities](docs/utilities.md) | Currency formatter, timezone helper, response transformer |

## Features

### Functional Options

Go's functional options pattern provides clean, flexible configuration. Only include the options you need — everything has sensible defaults:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithBasicAuth("customer@email.com", "password"),
	cocart.WithTimeout(30 * time.Second),
	cocart.WithMaxRetries(2),
	cocart.WithDebug(true),
)
```

### Fluent Setters

Setters return the `*Client` for optional chaining:

```go
client := cocart.NewClient("https://your-store.com").
	SetTimeout(60 * time.Second).
	SetMaxRetries(2).
	AddHeader("X-Custom", "value")
```

### Dot-Notation Response Access

Access nested data in API responses using a simple string path with dots — no need to manually traverse `map[string]any`:

```go
cart, _ := client.Cart().Get(ctx, nil)
cart.Get("totals.total", "0")          // Reach into nested objects
cart.Get("currency.currency_code", "") // No manual nil checks needed
cart.Get("items.0.name", "")           // Access array items by index
```

### Generic Typed Responses

Deserialize responses into your own Go structs:

```go
type MyCart struct {
	Items     []Item `json:"items"`
	ItemCount int    `json:"item_count"`
}

cart, err := cocart.Unmarshal[MyCart](response)
```

### Paginated Iteration (Go 1.23+)

Iterate through all pages using Go 1.23 range-over-func:

```go
paginator := client.Products().AllPaginated(nil)

for resp, err := range paginator.All(ctx) {
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Page:", string(resp.Body))
}
```

### Currency Formatting

```go
fmt := cocart.NewCurrencyFormatter()
currency := response.GetCurrency()

fmt.Format(4599, currency)        // "$45.99"
fmt.FormatDecimal(4599, currency) // "45.99"
```

### Client-Side Validation

Invalid inputs are caught before making a network request:

```go
_, err := client.Cart().AddItem(ctx, -1, 0)
// err is *ValidationError: "Product ID must be a positive integer"
```

### Event System

```go
client.OnRequest(func(e cocart.RequestEvent) {
	fmt.Printf("%s %s\n", e.Method, e.URL)
})
client.OnResponse(func(e cocart.ResponseEvent) {
	fmt.Printf("%d in %dms\n", e.Status, e.Duration.Milliseconds())
})
client.OnError(func(e cocart.ErrorEvent) {
	fmt.Println("Error:", e.Err)
})
```

### JWT with Auto-Refresh

**JWT (JSON Web Token)** authentication with automatic token refresh on 401 responses:

```go
// Login acquires JWT tokens
_, err := client.JWT().Login(ctx, "customer@email.com", "password")

// Tokens are automatically refreshed on 401 if a refresh token is available
cart, err := client.Cart().Get(ctx, nil)
```

### Error Type Checking

Use Go's `errors.As()` to check error types:

```go
var authErr *cocart.AuthenticationError
if errors.As(err, &authErr) {
	fmt.Println("Auth failed:", authErr.Message)
	fmt.Println("HTTP code:", authErr.HTTPCode)
}

var valErr *cocart.ValidationError
if errors.As(err, &valErr) {
	fmt.Println("Validation:", valErr.Message)
}
```

## Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithCartKey(key)` | Set an existing guest cart key | — |
| `WithBasicAuth(user, pass)` | Basic authentication | — |
| `WithJWTToken(token)` | JWT access token | — |
| `WithJWTRefreshToken(token)` | JWT refresh token | — |
| `WithWooCommerceKeys(key, secret)` | WooCommerce consumer keys (admin) | — |
| `WithTimeout(duration)` | HTTP request timeout | 30s |
| `WithRESTPrefix(prefix)` | WordPress REST API prefix | `"wp-json"` |
| `WithNamespace(ns)` | API namespace (white-labelling) | `"cocart"` |
| `WithHeaders(map)` | Custom headers for every request | — |
| `WithStorage(s)` | Storage adapter for cart keys/tokens | `MemoryStorage` |
| `WithStorageKey(key)` | Storage key name for cart key | `"cocart_cart_key"` |
| `WithMaxRetries(n)` | Max retries for transient failures | `0` |
| `WithDebug(bool)` | Enable debug logging | `false` |
| `WithAuthHeaderName(name)` | Custom auth header name | `"Authorization"` |
| `WithResponseTransformer(fn)` | Transform every response | — |
| `WithETag(bool)` | Enable ETag conditional requests | `true` |
| `WithMainPlugin(plugin)` | CoCart plugin variant | `MainPluginBasic` |
| `WithHTTPClient(hc)` | Custom `*http.Client` | — |

## Examples

| Example | Description |
|---------|-------------|
| [Basic Usage](examples/basic/) | Creating a client, browsing products, adding to cart |
| [JWT Authentication](examples/jwt-auth/) | JWT login, token management, guest-to-customer cart transfer |
| [Products Browse](examples/products-browse/) | Filtering, searching, pagination with range-over-func |
| [Headless Store](examples/headless-store/) | Complete end-to-end headless storefront flow |

## License

MIT
