# Sessions API

A **session** represents a single shopping cart — either a guest visitor's cart or a logged-in customer's cart. The server keeps track of all active sessions so that each visitor gets their own cart. This page covers two things:

1. **Admin Sessions Endpoint** — For store administrators to view and manage all active cart sessions.
2. **SessionManager** — A helper for your application to handle the guest-to-customer login flow.

## Admin Sessions Endpoint

The Sessions endpoint is for administrators to manage cart sessions server-side. It requires WooCommerce REST API credentials (see [Consumer Keys](authentication.md#consumer-keys-admin)).

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithWooCommerceKeys("ck_xxxxx", "cs_xxxxx"),
)
```

### List All Sessions

```go
response, err := client.Sessions().All(ctx, nil)

// With parameters
response, err := client.Sessions().All(ctx, &cocart.PaginationParams{PerPage: 50})
```

### Find a Session

```go
// By cart key
response, err := client.Sessions().Find(ctx, "guest_abc123")

// By customer ID
response, err := client.Sessions().BySession(ctx, 123)
```

### Get Session Items

```go
response, err := client.Sessions().GetItems(ctx, "guest_abc123")
```

### Delete a Session

```go
// By cart key
response, err := client.Sessions().Destroy(ctx, "guest_abc123")

// By customer ID
response, err := client.Sessions().DestroySession(ctx, 123)
```

---

## SessionManager

The `SessionManager` is a helper for applications that need to handle the guest-to-customer login flow. A visitor browses as a guest, adds items to a cart, then logs in — and their guest cart should transfer to their customer account. The `SessionManager` handles all of this automatically.

### Basic Setup

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithStorage(cocart.NewMemoryStorage()),
)
session := client.Session()
```

### Initialize a Cart

Creates a guest cart and persists the cart key:

```go
cartKey, err := session.InitializeCart(ctx)
fmt.Println(cartKey) // "guest_abc123..."
```

### Login with Basic Auth

```go
// Guest adds items first
client.Cart().AddItem(ctx, 123, 2)

// Login and merge guest cart into customer cart
response, err := session.Login(ctx, "customer@email.com", "password", true)

// Or login without merging (starts fresh customer cart)
response, err := session.Login(ctx, "customer@email.com", "password", false)
```

### Login with JWT

```go
// Guest adds items
client.Cart().AddItem(ctx, 123, 2)

// Login via JWT and merge cart
response, err := session.LoginWithJWT(ctx, "customer@email.com", "password", true)
```

### Login with Existing JWT Token

```go
response, err := session.LoginWithToken(ctx, "eyJ...")
```

### Logout

```go
err := session.Logout(ctx)

// Start a new guest session
session.InitializeCart(ctx)
```

### Session Status

```go
session.IsAuthenticated() // true if Basic Auth or JWT is set
session.IsGuest()         // true if no auth credentials
session.GetCartKey()      // current cart key or ""
```

---

## Storage Adapters

A **storage adapter** knows how to save and retrieve data. The SDK needs storage to persist cart keys and JWT tokens. Different environments may need different storage strategies.

All storage adapters implement the `Storage` interface:

```go
type Storage interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	Delete(key string) error
}
```

### MemoryStorage

Stores data in memory using a Go `map` protected by `sync.RWMutex`. This is the default. Data is lost when the process restarts. Best for short-lived operations, servers that handle sessions per-request, or testing.

```go
storage := cocart.NewMemoryStorage()
```

### Custom Storage

If the built-in `MemoryStorage` doesn't fit your needs, implement the `Storage` interface. For example, you could store data in Redis, a database, or a file:

```go
type RedisStorage struct {
	client *redis.Client
}

func (s *RedisStorage) Get(key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", cocart.ErrKeyNotFound
	}
	return val, err
}

func (s *RedisStorage) Set(key string, value string) error {
	return s.client.Set(ctx, key, value, 0).Err()
}

func (s *RedisStorage) Delete(key string) error {
	return s.client.Del(ctx, key).Err()
}
```

Use your custom storage:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithStorage(&RedisStorage{client: redisClient}),
)
```

When implementing custom storage, return `cocart.ErrKeyNotFound` from `Get()` when a key doesn't exist. This allows the SDK to distinguish between "key not found" and other errors.

---

## Cart Transfer on Login

This is one of the most important flows in headless e-commerce. A visitor shops as a guest, fills up their cart, then decides to log in. You want their guest cart items to carry over to their customer cart. Here's how the full flow works:

```go
client := cocart.NewClient("https://your-store.com",
	cocart.WithStorage(cocart.NewMemoryStorage()),
)
session := client.Session()

ctx := context.Background()

// 1. Initialize guest session
session.InitializeCart(ctx)

// 2. Guest browses and adds items
client.Cart().AddItem(ctx, 123, 2)
client.Cart().AddItem(ctx, 456, 1)

// 3. Guest decides to log in (merge cart = true)
session.LoginWithJWT(ctx, "customer@email.com", "password", true)

// 4. Guest cart items are now in the customer's cart
cart, _ := client.Cart().Get(ctx, nil)
items := cart.GetItems() // Contains items 123 and 456

// 5. Later, customer logs out
session.Logout(ctx)
session.InitializeCart(ctx) // Fresh guest session
```

See [Authentication](authentication.md) for more on JWT and Basic Auth setup.
