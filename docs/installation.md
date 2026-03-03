# Configuration & Setup

For installation instructions and requirements, see the [README](../README.md#installation).

## Configuration Options

The `NewClient` function accepts functional options as variadic arguments. Only include the ones relevant to your setup — everything has sensible defaults.

```go
import (
	"time"
	cocart "github.com/cocart-headless/cocart-sdk-go"
)

client := cocart.NewClient("https://your-store.com",
	// Guest session
	cocart.WithCartKey("existing_cart_key"),

	// Basic Auth
	cocart.WithBasicAuth("customer@email.com", "password"),

	// JWT Auth
	cocart.WithJWTToken("your-jwt-token"),
	cocart.WithJWTRefreshToken("your-refresh-token"),

	// Admin (Sessions API)
	cocart.WithWooCommerceKeys("ck_xxxxx", "cs_xxxxx"),

	// HTTP settings
	cocart.WithTimeout(30 * time.Second),

	// REST API prefix (default: "wp-json")
	cocart.WithRESTPrefix("wp-json"),

	// API namespace (default: "cocart")
	cocart.WithNamespace("cocart"),

	// CoCart main plugin: MainPluginBasic (default) or MainPluginLegacy
	cocart.WithMainPlugin(cocart.MainPluginBasic),

	// Retry transient failures (429, 503, timeouts)
	cocart.WithMaxRetries(2),

	// Storage adapter (default: MemoryStorage)
	cocart.WithStorage(cocart.NewMemoryStorage()),
	cocart.WithStorageKey("cocart_cart_key"),

	// Custom auth header name (default: "Authorization")
	cocart.WithAuthHeaderName("X-Auth-Token"),

	// Transform every API response before returning
	cocart.WithResponseTransformer(func(r *cocart.Response) *cocart.Response {
		return r
	}),

	// Enable ETag conditional requests (default: true)
	cocart.WithETag(true),

	// Enable debug logging (default: false)
	cocart.WithDebug(true),

	// Custom HTTP client
	cocart.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
)
```

## Fluent Configuration

Setters return `*Client` for optional chaining. Each method configures one setting and returns the same client, so you can chain them with dots:

```go
client := cocart.NewClient("https://your-store.com").
	SetTimeout(60 * time.Second).
	SetMaxRetries(2).
	SetRESTPrefix("api").
	SetNamespace("mystore").
	AddHeader("X-Custom-Header", "value").
	SetAuthHeaderName("X-Auth-Token").
	SetETag(true).
	SetMainPlugin(cocart.MainPluginBasic).
	SetDebug(true)
```

This is equivalent to writing each call on its own line:

```go
client := cocart.NewClient("https://your-store.com")
client.SetTimeout(60 * time.Second)
client.SetMaxRetries(2)
// ... and so on
```

## White-Labelling / Custom REST Prefix

WordPress exposes its REST API at `/wp-json/` by default. The SDK builds URLs like `https://your-store.com/wp-json/cocart/v2/cart`. If your site or hosting changes this prefix, or if the CoCart plugin has been renamed (white-labelled), you can configure the SDK to match:

```go
// Custom REST prefix (site uses /api/ instead of /wp-json/)
client := cocart.NewClient("https://your-store.com",
	cocart.WithRESTPrefix("api"),
)
// Requests go to: https://your-store.com/api/cocart/v2/cart

// White-labelled namespace
client2 := cocart.NewClient("https://your-store.com",
	cocart.WithNamespace("mystore"),
)
// Requests go to: https://your-store.com/wp-json/mystore/v2/cart

// Both together
client3 := cocart.NewClient("https://your-store.com",
	cocart.WithRESTPrefix("api"),
	cocart.WithNamespace("mystore"),
)
// Requests go to: https://your-store.com/api/mystore/v2/cart
```

## Legacy Plugin Support

The SDK supports both **CoCart Basic** and the **legacy CoCart plugin** (`cart-rest-api-for-woocommerce` v4.x). By default, the SDK targets CoCart Basic.

To use the SDK with the legacy plugin, set `mainPlugin` to `MainPluginLegacy`:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithMainPlugin(cocart.MainPluginLegacy),
)

// Or use the fluent setter
client.SetMainPlugin(cocart.MainPluginLegacy)
```

### What changes in legacy mode

**Basic-only methods return an error immediately.** Methods that require CoCart Basic return a `*VersionError` before making any HTTP request, with a clear message indicating which method requires an upgrade:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithMainPlugin(cocart.MainPluginLegacy),
)

_, err := client.Products().FindBySlug(ctx, "blue-hoodie")

var vErr *cocart.VersionError
if errors.As(err, &vErr) {
	// "FindBySlug() requires CoCart Basic. Please upgrade..."
	fmt.Println(vErr.Message)
}
```

Basic-only methods include:

- `Cart().Create()`
- `Products().FindBySlug()`, `Variation()`, `Category()`, `Tag()`
- `Products().AttributeBySlug()`, `AttributeTermsBySlug()`, `AttributeTermBySlug()`
- `Products().Brands()`, `Brand()`, `ByBrand()`
- `Products().MyReviews()`

## Custom HTTP Client

For advanced use cases like custom TLS configuration, proxy settings, or transport middleware, pass your own `*http.Client`:

```go
transport := &http.Transport{
	MaxIdleConns:    10,
	IdleConnTimeout: 30 * time.Second,
}

client := cocart.NewClient("https://your-store.com",
	cocart.WithHTTPClient(&http.Client{
		Transport: transport,
	}),
)
```

## Event System

Register callbacks for request/response lifecycle events. These are useful for logging, metrics, or debugging:

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

## Debug Mode

Enable debug mode to log request/response details to stderr:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithDebug(true),
)
```
