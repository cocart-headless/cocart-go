# Store API

The Store API returns information about your WooCommerce store — its name, description, URL, and available API routes. This is useful for verifying connectivity, displaying store branding, or discovering available endpoints.

```go
store := client.Store()
```

## Get Store Info

```go
response, err := client.Store().Info(ctx)
if err != nil {
	log.Fatal(err)
}

fmt.Println(response.Get("store_name", ""))
fmt.Println(response.Get("store_description", ""))
fmt.Println(response.Get("store_url", ""))
```

## Typed Deserialization

Use `Unmarshal[T]` to deserialize the response into a `StoreInfo` struct:

```go
response, err := client.Store().Info(ctx)
if err != nil {
	log.Fatal(err)
}

info, err := cocart.Unmarshal[cocart.StoreInfo](response)
if err != nil {
	log.Fatal(err)
}

fmt.Println("Store:", info.StoreName)
fmt.Println("URL:", info.StoreURL)
fmt.Println("Description:", info.StoreDescription)
```

## Field Filtering

Request only specific fields to reduce response size:

```go
response, err := client.Store().Info(ctx, map[string]string{
	"_fields": "store_name,store_url",
})
```

## StoreInfo Type

The `StoreInfo` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `StoreName` | `string` | The store's display name |
| `StoreDescription` | `string` | The store's tagline or description |
| `StoreURL` | `string` | The store's public URL |
| `Routes` | `map[string]any` | Available API routes and their details |
