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

fmt.Println(response.Get("title", ""))
fmt.Println(response.Get("description", ""))
fmt.Println(response.Get("home_url", ""))
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

fmt.Println("Store:", info.Title)
fmt.Println("URL:", info.HomeURL)
fmt.Println("Description:", info.Description)
fmt.Println("CoCart:", info.Versions["core"], info.Versions["plus"])
```

## Field Filtering

Request only specific fields to reduce response size:

```go
response, err := client.Store().Info(ctx, map[string]string{
	"_fields": "title,home_url",
})
```

## StoreInfo Type

The `StoreInfo` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | The store's display name |
| `Description` | `string` | The store's tagline or description |
| `HomeURL` | `string` | The store's public URL |
| `Language` | `string` | The store's language, e.g. `en-US` |
| `GMTOffset` | `string` | The store's offset from GMT, in hours |
| `TimezoneString` | `string` | The store's timezone |
| `StoreAddress` | `ObjectMap[string]` | `address`, `address_2`, `city`, `country`, `postcode` |
| `Routes` | `ObjectMap[any]` | Available API routes, keyed by name |
| `Versions` | `ObjectMap[string]` | The CoCart versions: `core`, and `plus` with CoCart Plus |
