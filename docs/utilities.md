# Utilities

The SDK includes standalone utility types for common tasks in headless WooCommerce projects. These are optional — you can use them when you need them, and they don't affect the core SDK behavior.

## Currency Formatter

### Why do prices come back as integers?

CoCart returns prices as **smallest-unit integers** — that means cents for USD, pence for GBP, or the smallest denomination for any currency. For example, `4599` means $45.99 (4599 cents). This is an industry-standard practice because floating-point numbers (like `45.99`) can cause rounding errors in calculations, while integers are always exact.

The `CurrencyFormatter` converts these integers into human-readable price strings using the `CurrencyInfo` metadata from API responses.

```go
import cocart "github.com/cocart-headless/cocart-sdk-go"

formatter := cocart.NewCurrencyFormatter()
```

### Formatting Prices

Use the `CurrencyInfo` from a cart response to format amounts:

```go
response, _ := client.Cart().Get(ctx, nil)
currency := response.GetCurrency()

formatter.Format(4599, currency)  // "$45.99"
formatter.Format(100, currency)   // "$1.00"
formatter.Format(0, currency)     // "$0.00"
```

### Decimal String (No Symbol)

```go
formatter.FormatDecimal(4599, currency) // "45.99"
```

### Different Currencies

The formatter respects the currency metadata from the API response:

```go
// Euro (2 decimal places, comma decimal, suffix symbol)
eur := cocart.CurrencyInfo{
	CurrencyCode:        "EUR",
	CurrencySymbol:      "€",
	CurrencyMinorUnit:   2,
	CurrencyDecimalSep:  ",",
	CurrencyThousandSep: ".",
	CurrencyPrefix:      "",
	CurrencySuffix:      " €",
}
formatter.Format(1299, eur)  // "12,99 €"

// Japanese Yen (0 decimal places)
jpy := cocart.CurrencyInfo{
	CurrencyCode:        "JPY",
	CurrencySymbol:      "¥",
	CurrencyMinorUnit:   0,
	CurrencyThousandSep: ",",
	CurrencyPrefix:      "¥",
}
formatter.Format(1500, jpy)  // "¥1,500"

// Large number with thousand separators
formatter.Format(123456789, cocart.CurrencyInfo{
	CurrencyCode:        "USD",
	CurrencyMinorUnit:   2,
	CurrencyDecimalSep:  ".",
	CurrencyThousandSep: ",",
	CurrencyPrefix:      "$",
})
// "$1,234,567.89"
```

---

## Timezone Helper

### Why do timezones matter?

Your WooCommerce store has a configured timezone (e.g., `America/New_York` or `UTC`), and dates in API responses use that timezone. But your customer might be in a completely different timezone. The `TimezoneHelper` handles these conversions using Go's `time.LoadLocation` from the standard library, which understands timezone rules including daylight saving time.

```go
tz := cocart.NewTimezoneHelper()
```

### Detect User Timezone

```go
timezone := tz.DetectTimezone()
// "America/New_York", "Europe/London", "Asia/Tokyo", etc.
```

### Convert Between Timezones

```go
result, err := tz.Convert("2025-01-15T10:00:00", "UTC", "America/New_York")
// "2025-01-15T05:00:00" (EST is UTC-5)

result, err = tz.Convert("2025-06-15T10:00:00", "UTC", "America/New_York")
// "2025-06-15T06:00:00" (EDT is UTC-4, daylight saving)
```

### Convert Store Time to Local Time

Shorthand for converting a store date to the user's local timezone:

```go
// Store is in UTC, convert to user's local timezone
result, err := tz.ToLocal("2025-01-15T10:00:00", "UTC")
```

---

## Response Transformer

A **response transformer** is a function that the SDK calls on every API response before returning it to your code. Think of it as middleware — it receives the response, you can inspect or modify it, and then you return it.

Common uses:

- **Logging** — Log every response status for debugging.
- **Metrics** — Track response times or error rates.
- **Data enrichment** — Add computed fields or format values before your application sees them.

### Via Constructor

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithResponseTransformer(func(r *cocart.Response) *cocart.Response {
		fmt.Printf("[%d] Response received\n", r.StatusCode)
		return r
	}),
)
```

### Via Fluent Setter

```go
client := cocart.NewClient("https://your-store.com").
	SetResponseTransformer(func(r *cocart.Response) *cocart.Response {
		// Add timing metadata, format currencies, etc.
		return r
	})
```

### Clearing the Transformer

```go
client.SetResponseTransformer(nil)
```

### Example: Logging All Responses

```go
client.SetResponseTransformer(func(r *cocart.Response) *cocart.Response {
	fmt.Printf("Status: %d\n", r.StatusCode)
	fmt.Printf("Items: %d\n", r.GetItemCount())
	return r
})
```
