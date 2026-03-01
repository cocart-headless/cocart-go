// Example: Browsing Products with CoCart SDK
//
// This example demonstrates product listing, filtering,
// searching, and paginated iteration using Go 1.23 range-over-func.
package main

import (
	"context"
	"fmt"
	"log"

	cocart "github.com/cocart-headless/cocart-sdk-go"
)

func main() {
	ctx := context.Background()

	client := cocart.NewClient("https://your-store.com")

	// --- Basic Listing ---

	// List first 12 products
	response, err := client.Products().All(ctx, &cocart.ProductListParams{
		PerPage: 12,
	})
	if err != nil {
		log.Fatal("Failed to list products:", err)
	}
	fmt.Printf("Got %d total products across %d pages\n",
		response.GetTotalResults(), response.GetTotalPages())

	// --- Filtering ---

	// By category
	electronics, err := client.Products().ByCategory(ctx, "electronics", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Electronics:", electronics.StatusCode)

	// Featured products
	featured, err := client.Products().Featured(ctx, &cocart.ProductListParams{PerPage: 4})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Featured:", featured.StatusCode)

	// Products on sale
	onSale, err := client.Products().OnSale(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("On sale:", onSale.StatusCode)

	// Search
	results, err := client.Products().Search(ctx, "wireless headphones", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Search results:", results.StatusCode)

	// Combined filters
	filtered, err := client.Products().All(ctx, &cocart.ProductListParams{
		Category: "clothing",
		OnSale:   true,
		MinPrice: "20",
		MaxPrice: "100",
		OrderBy:  "popularity",
		Order:    "desc",
		PerPage:  12,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Filtered:", filtered.StatusCode)

	// --- Single Product ---

	product, err := client.Products().Find(ctx, 123)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Product name:", product.Get("name", ""))
	fmt.Println("Product price:", product.Get("price", ""))

	// --- Pagination with range-over-func (Go 1.23+) ---

	fmt.Println("\n--- Paginating all products ---")
	paginator := client.Products().AllPaginated(&cocart.ProductListParams{PerPage: 20})

	pageNum := 0
	for resp, err := range paginator.All(ctx) {
		if err != nil {
			log.Fatal("Pagination error:", err)
		}
		pageNum++
		fmt.Printf("Page %d: status %d, total pages: %d\n",
			pageNum, resp.StatusCode, resp.GetTotalPages())
	}
	fmt.Printf("Iterated through %d pages\n", pageNum)

	// Or collect all pages into a slice
	pages, err := client.Products().AllPaginated(nil).Collect(ctx)
	if err != nil {
		log.Fatal("Collect error:", err)
	}
	fmt.Printf("Collected %d pages\n", len(pages))

	// --- Categories & Attributes ---

	categories, err := client.Products().Categories(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Categories:", categories.StatusCode)

	attributes, err := client.Products().Attributes(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Attributes:", attributes.StatusCode)

	// --- Reviews ---

	reviews, err := client.Products().Reviews(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Reviews:", reviews.StatusCode)
}
