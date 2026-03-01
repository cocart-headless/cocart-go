package cocart_test

import (
	"context"
	"fmt"
	"time"

	"github.com/cocart-headless/cocart-sdk-go"
)

func ExampleNewClient() {
	client := cocart.NewClient("https://example.com")
	fmt.Println(client.GetStoreURL())
	// Output: https://example.com
}

func ExampleNewClient_withOptions() {
	_ = cocart.NewClient("https://example.com",
		cocart.WithBasicAuth("user", "pass"),
		cocart.WithTimeout(15*time.Second),
		cocart.WithMaxRetries(3),
		cocart.WithDebug(true),
	)
}

func ExampleClient_Cart() {
	client := cocart.NewClient("https://example.com")
	ctx := context.Background()

	// Get cart contents
	resp, err := client.Cart().Get(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Items: %d\n", resp.GetItemCount())
}

func ExampleClient_Products() {
	client := cocart.NewClient("https://example.com")
	ctx := context.Background()

	// Get all products
	resp, err := client.Products().All(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Status: %d\n", resp.StatusCode)
}

func ExampleUnmarshal() {
	client := cocart.NewClient("https://example.com")
	ctx := context.Background()

	resp, err := client.Products().Find(ctx, 42)
	if err != nil {
		return
	}

	product, err := cocart.Unmarshal[cocart.Product](resp)
	if err != nil {
		return
	}
	fmt.Printf("Product: %s\n", product.Name)
}

func ExampleCurrencyFormatter_Format() {
	f := cocart.NewCurrencyFormatter()
	info := cocart.CurrencyInfo{
		CurrencyCode:        "USD",
		CurrencySymbol:      "$",
		CurrencyMinorUnit:   2,
		CurrencyDecimalSep:  ".",
		CurrencyThousandSep: ",",
		CurrencyPrefix:      "$",
		CurrencySuffix:      "",
	}

	result := f.Format(4599, info)
	fmt.Println(result)
	// Output: $45.99
}

func ExampleCurrencyFormatter_FormatDecimal() {
	f := cocart.NewCurrencyFormatter()
	info := cocart.CurrencyInfo{CurrencyMinorUnit: 2}

	result := f.FormatDecimal(4599, info)
	fmt.Println(result)
	// Output: 45.99
}

func ExampleResponse_Get() {
	resp := &cocart.Response{
		StatusCode: 200,
		Headers:    make(map[string][]string),
		Body:       []byte(`{"items":[{"name":"Widget"}],"totals":{"total":"9.99"}}`),
	}

	fmt.Println(resp.Get("totals.total"))
	fmt.Println(resp.Get("items.0.name"))
	fmt.Println(resp.Get("missing", "default"))
	// Output:
	// 9.99
	// Widget
	// default
}

func ExampleResponse_GetString() {
	resp := &cocart.Response{
		StatusCode: 200,
		Headers:    make(map[string][]string),
		Body:       []byte(`{"name":"Widget","count":5}`),
	}

	fmt.Println(resp.GetString("name", "unknown"))
	fmt.Println(resp.GetInt("count", 0))
	// Output:
	// Widget
	// 5
}

func ExampleClient_Session() {
	client := cocart.NewClient("https://example.com",
		cocart.WithStorage(cocart.NewMemoryStorage()),
	)
	session := client.Session()

	fmt.Println("Guest:", session.IsGuest())
	fmt.Println("Authenticated:", session.IsAuthenticated())
	// Output:
	// Guest: true
	// Authenticated: false
}

func ExampleClient_OnRequest() {
	client := cocart.NewClient("https://example.com")

	// Log all outgoing requests
	unsubscribe := client.OnRequest(func(e cocart.RequestEvent) {
		fmt.Printf("-> %s %s\n", e.Method, e.URL)
	})

	// Later, to stop listening:
	_ = unsubscribe
}

func ExampleClient_OnResponse() {
	client := cocart.NewClient("https://example.com")

	client.OnResponse(func(e cocart.ResponseEvent) {
		fmt.Printf("<- %d %s (%v)\n", e.Status, e.URL, e.Duration)
	})
}
