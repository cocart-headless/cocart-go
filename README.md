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

## Requirements

- **Go 1.23+** — Required for `iter.Seq2` (range-over-func) support used by the paginator.
- **CoCart plugin** installed on your WooCommerce store — This is the WordPress plugin that provides the REST API endpoints the SDK communicates with.
- [CoCart JWT Authentication](https://wordpress.org/plugins/cocart-jwt-authentication/) plugin for JWT features (optional) — Only needed if you want to use JSON Web Token authentication (explained in the [Authentication](docs/authentication.md) guide).

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

## CoCart Channels

We have different channels at your disposal where you can find information about the CoCart project, discuss it and get involved:

[![Twitter: cocartapi](https://img.shields.io/twitter/follow/cocartapi?style=social)](https://twitter.com/cocartapi) [![CoCart GitHub Stars](https://img.shields.io/github/stars/cocart-headless/cocart-go?style=social)](https://github.com/cocart-headless/cocart-go)

<ul>
  <li>📖 <strong>Documentation</strong>: this is the place to learn how to use CoCart API. <a href="https://cocartapi.com/docs/?utm_medium=gh&utm_source=github&utm_campaign=readme&utm_content=cocart">Get started!</a></li>
  <li>👪 <strong>Community</strong>: use our Discord chat room to share any doubts, feedback and meet great people. This is your place too to share <a href="https://cocartapi.com/community/?utm_medium=gh&utm_source=github&utm_campaign=readme&utm_content=cocart">how are you planning to use CoCart!</a></li>
  <li>🐞 <strong>GitHub</strong>: we use GitHub for bugs and pull requests, doubts are solved with the community.</li>
  <li>🐦 <strong>Social media</strong>: a more informal place to interact with CoCart users, reach out to us on <a href="https://twitter.com/cocartapi">X/Twitter.</a></li>
</ul>

## Credits

Website [cocartapi.com](https://cocartapi.com/?ref=github) &nbsp;&middot;&nbsp;
GitHub [@cocart-headless](https://github.com/cocart-headless) &nbsp;&middot;&nbsp;
X/Twitter [@cocartapi](https://twitter.com/cocartapi) &nbsp;&middot;&nbsp;
[Facebook](https://www.facebook.com/cocartforwc/) &nbsp;&middot;&nbsp;
[Instagram](https://www.instagram.com/cocartheadless/)

## License

MIT
