# CoCart Go SDK

Official Go SDK for the CoCart REST API.

- **Module:** `github.com/cocart-headless/cocart-sdk-go`
- **Version:** 1.0.0
- **Distribution:** Go modules (go.mod / go.sum)
- **License:** MIT
- **Requires:** Go 1.23+
- **Zero external dependencies** — standard library only

---

## Commands

```bash
go build ./...                           # build
go test -race -v ./...                   # run all tests
go test -run TestCartGet -v ./...        # run a single test by name
go test -cover ./...                     # run tests with coverage
go vet ./...                             # static analysis
```

---

## Tech Stack

| | |
|---|---|
| Language | Go 1.23+ |
| Tests | `testing` package (standard library) |
| HTTP | `net/http` (standard library) |
| External deps | none |
| Build | Go toolchain only |

Go 1.23 is required for `iter.Seq2` / range-over-func support.

---

## Project Structure

All source files live in the root package `cocart`:

```
cocart.go               # Client type, request pipeline, header building
cart.go                 # CartEndpoint
products.go             # ProductsEndpoint
sessions_endpoint.go    # SessionsEndpoint
jwt.go                  # JWTManager
session.go              # session lifecycle
response.go             # Response wrapper
errors.go               # error types
options.go              # functional options pattern (WithTimeout, WithAuth, …)
validation.go           # input validators
params.go               # query parameter helpers
paginator.go            # Paginator and async iterator
currency.go             # CurrencyFormatter
timezone.go             # TimezoneHelper
storage.go              # CoCartStorage interface + implementations
doc.go                  # package-level documentation
*_test.go               # tests colocated with source
examples/               # runnable examples
docs/                   # documentation
```

---

## Code Style

- **Exported identifiers:** `PascalCase` (`Client`, `NewClient`, `AddItem`)
- **Unexported identifiers:** `camelCase` (`buildHeaders`, `extractCartKey`)
- **Constants:** `PascalCase` (`Version`, `APIVersion`, `MainPluginBasic`)
- **Config pattern:** functional options (`WithTimeout(30*time.Second)`)
- Tests are colocated with source (`cart.go` / `cart_test.go`)
- Use `httptest` for HTTP mocking in tests — no real network calls

---

## Git

- **Commit style:** Imperative, capital first letter — `Add X`, `Added X`, `Fix X`
- **Co-author footer:** `Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>`

---

## Testing

| | |
|---|---|
| Framework | `testing` (stdlib) |
| Location | colocated `*_test.go` files |
| Mocking | `net/http/httptest` |
| Race detection | always run with `-race` |
| Pattern | table-driven tests |

Run a specific test: `go test -run TestCartGet -v ./...`. Use `-run` with a regex to match test function names.
