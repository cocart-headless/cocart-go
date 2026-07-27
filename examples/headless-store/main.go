// Example: Complete Headless Storefront with CoCart SDK
//
// This example demonstrates a full end-to-end headless WooCommerce
// storefront flow: browsing products, managing a cart, handling
// authentication, and preparing for checkout.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cocart "github.com/cocart-headless/cocart-sdk-go"
)

func main() {
	ctx := context.Background()

	// --- 1. Create client with storage and retry ---

	client := cocart.NewClient("https://your-store.com",
		cocart.WithStorage(cocart.NewMemoryStorage()),
		cocart.WithTimeout(15*time.Second),
		cocart.WithMaxRetries(2),
	)

	// --- 2. Get store info ---

	storeInfo, err := client.Store().Info(ctx)
	if err != nil {
		log.Fatal("Failed to get store info:", err)
	}
	fmt.Println("Store:", storeInfo.Get("store_name", ""))
	fmt.Println("URL:", storeInfo.Get("store_url", ""))

	// --- 3. Browse featured products ---

	featured, err := client.Products().Featured(ctx, &cocart.ProductListParams{PerPage: 4})
	if err != nil {
		log.Fatal("Failed to get featured products:", err)
	}
	fmt.Printf("\nFeatured products (%d total):\n", featured.GetTotalResults())

	// --- 4. Filter by category ---

	electronics, err := client.Products().ByCategory(ctx, "electronics", &cocart.ProductListParams{
		PerPage: 10,
		OrderBy: cocart.OrderByPopularity,
		Order:   cocart.SortDesc,
	})
	if err != nil {
		log.Fatal("Failed to browse electronics:", err)
	}
	fmt.Printf("\nElectronics: %d products\n", electronics.GetTotalResults())

	// --- 5. Search products ---

	results, err := client.Products().Search(ctx, "wireless headphones", &cocart.ProductListParams{
		MinPrice: "20",
		MaxPrice: "200",
	})
	if err != nil {
		log.Fatal("Failed to search:", err)
	}
	fmt.Printf("Search results: %d products\n", results.GetTotalResults())

	// --- 6. View single product + variations ---

	product, err := client.Products().Find(ctx, 123)
	if err != nil {
		log.Fatal("Failed to get product:", err)
	}
	fmt.Printf("\nProduct: %s\n", product.Get("name", ""))
	fmt.Printf("Price: %s\n", product.Get("prices.price", ""))
	fmt.Printf("In stock: %s\n", product.Get("stock.stock_status", ""))

	variations, err := client.Products().Variations(ctx, 123)
	if err != nil {
		log.Fatal("Failed to get variations:", err)
	}
	fmt.Printf("Variations: %d\n", variations.GetTotalResults())

	// --- 7. Initialize guest cart session ---

	session := client.Session()
	cartKey, err := session.InitializeCart(ctx)
	if err != nil {
		log.Fatal("Failed to initialize cart:", err)
	}
	fmt.Printf("\nGuest cart key: %s\n", cartKey)

	// --- 8. Add items (simple + variable product) ---

	// Add a simple product
	_, err = client.Cart().AddItem(ctx, 123, 2)
	if err != nil {
		log.Fatal("Failed to add item:", err)
	}
	fmt.Println("Added simple product (qty 2)")

	// Add a variable product with attributes
	_, err = client.Cart().AddVariation(ctx, 456, 1, map[string]string{
		"attribute_pa_color": "blue",
		"attribute_pa_size":  "large",
	})
	if err != nil {
		log.Fatal("Failed to add variation:", err)
	}
	fmt.Println("Added variable product (Blue, Large)")

	// --- 9. Apply coupon ---

	_, err = client.Cart().ApplyCoupon(ctx, "WELCOME10")
	if err != nil {
		fmt.Println("Coupon not applied:", err)
	} else {
		fmt.Println("Applied coupon: WELCOME10")
	}

	// --- 10. Update customer billing/shipping ---

	_, err = client.Cart().UpdateCustomer(ctx,
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
		map[string]string{
			"first_name": "John",
			"last_name":  "Doe",
			"address_1":  "123 Main St",
			"city":       "New York",
			"state":      "NY",
			"postcode":   "10001",
			"country":    "US",
		},
	)
	if err != nil {
		log.Fatal("Failed to update customer:", err)
	}
	fmt.Println("\nCustomer details updated")

	// --- 11. Calculate shipping ---
	//
	// There is no address-taking shipping-calculation endpoint in the CoCart
	// REST API (CalculateShipping is deprecated for this reason). Shipping
	// was already recalculated as a side effect of UpdateCustomer() above,
	// which set the destination address — Calculate() just recalculates
	// totals against whatever address is already on the cart.

	_, err = client.Cart().Calculate(ctx)
	if err != nil {
		fmt.Println("Shipping calculation:", err)
	}

	shippingResp, err := client.Cart().GetShippingMethods(ctx)
	if err != nil {
		log.Fatal("Failed to get shipping methods:", err)
	}
	shipping := shippingResp.GetShippingMethods()
	fmt.Printf("Shipping packages: %d\n", len(shipping))

	// --- 12. Cart summary with formatted prices ---

	cart, err := client.Cart().Get(ctx, nil)
	if err != nil {
		log.Fatal("Failed to get cart:", err)
	}

	fmt.Println("\n--- Cart Summary ---")
	fmt.Printf("Items: %d\n", cart.GetItemCount())

	items := cart.GetItems()
	for _, item := range items {
		fmt.Printf("  - %s (qty: %d) — %s\n", item.Name, item.Quantity.Value, item.Totals.Total)
	}

	totals := cart.GetTotals()
	fmt.Printf("Subtotal: %s\n", totals.Subtotal)
	fmt.Printf("Shipping: %s\n", totals.ShippingTotal)
	fmt.Printf("Discount: %s\n", totals.DiscountTotal)
	fmt.Printf("Tax: %s\n", totals.TotalTax)
	fmt.Printf("Total: %s\n", totals.Total)

	// Format prices using CurrencyFormatter
	formatter := cocart.NewCurrencyFormatter()
	currency := cart.GetCurrency()
	fmt.Printf("\nFormatted total: %s\n", formatter.Format(4599, currency))

	if cart.HasCoupons() {
		coupons := cart.GetCoupons()
		for _, c := range coupons {
			fmt.Printf("Coupon: %s — Saving: %s\n", c.Coupon, c.Saving)
		}
	}

	// --- 13. Login and transfer cart ---

	fmt.Println("\n--- Login & Cart Transfer ---")
	loginResp, err := session.LoginWithJWT(ctx, "customer@email.com", "password", true)
	if err != nil {
		fmt.Println("Login skipped (JWT plugin not available):", err)
	} else {
		fmt.Println("Logged in as:", loginResp.Get("display_name", ""))
		fmt.Println("Guest cart items transferred to customer account")

		// Verify cart after transfer
		customerCart, err := client.Cart().Get(ctx, nil)
		if err != nil {
			log.Fatal("Failed to get customer cart:", err)
		}
		fmt.Printf("Customer cart items: %d\n", customerCart.GetItemCount())
	}
}
