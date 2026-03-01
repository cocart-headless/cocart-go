// Example: Basic CoCart SDK usage
//
// This example demonstrates creating a client, browsing products,
// adding items to a cart, and reading cart data.
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
		log.Fatal("Failed to get products:", err)
	}
	fmt.Println("Products loaded, status:", products.StatusCode)

	// Add an item to the cart (guest session created automatically)
	response, err := client.Cart().AddItem(ctx, 123, 2)
	if err != nil {
		log.Fatal("Failed to add item:", err)
	}
	fmt.Println("Added item, key:", response.Get("item_key", ""))

	// The SDK automatically captures the cart key from the response
	fmt.Println("Cart key:", client.GetCartKey())

	// Get the full cart
	cart, err := client.Cart().Get(ctx, nil)
	if err != nil {
		log.Fatal("Failed to get cart:", err)
	}

	// Use helper methods to access cart data
	fmt.Println("Item count:", cart.GetItemCount())
	fmt.Println("Total:", cart.Get("totals.total", "0"))

	// Use dot-notation for nested data
	fmt.Println("First item name:", cart.Get("items.0.name", ""))
	fmt.Println("Currency:", cart.Get("currency.currency_code", ""))

	// Check cart state
	if cart.HasItems() {
		fmt.Println("Cart has items!")
	}

	// Format prices using CurrencyFormatter
	formatter := cocart.NewCurrencyFormatter()
	currency := cart.GetCurrency()
	fmt.Println("Formatted total:", formatter.Format(4599, currency))

	// Clear the cart
	_, err = client.Cart().Clear(ctx)
	if err != nil {
		log.Fatal("Failed to clear cart:", err)
	}
	fmt.Println("Cart cleared!")
}
