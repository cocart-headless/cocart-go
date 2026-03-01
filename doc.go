// Package cocart provides the official Go SDK for the CoCart REST API.
//
// CoCart is a headless WooCommerce cart plugin that provides a REST API
// for managing shopping carts, browsing products, and handling sessions.
//
// # Quick Start
//
//	client := cocart.NewClient("https://example.com")
//
//	// Get cart contents
//	cart, err := client.Cart().Get(ctx)
//
//	// Add item to cart
//	item, err := client.Cart().AddItem(ctx, 42, 1)
//
//	// Browse products
//	products, err := client.Products().All(ctx)
//
// # Authentication
//
// The SDK supports multiple authentication methods:
//
//	// Basic Auth
//	client := cocart.NewClient(url, cocart.WithBasicAuth("user", "pass"))
//
//	// JWT
//	client := cocart.NewClient(url, cocart.WithJWTToken("token"))
//
//	// WooCommerce Consumer Keys (admin)
//	client := cocart.NewClient(url, cocart.WithWooCommerceKeys("ck_...", "cs_..."))
//
// # Configuration
//
// Use functional options to configure the client:
//
//	client := cocart.NewClient(url,
//	    cocart.WithTimeout(15 * time.Second),
//	    cocart.WithMaxRetries(3),
//	    cocart.WithDebug(true),
//	)
package cocart
