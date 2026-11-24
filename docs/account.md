# Account API

> [!IMPORTANT]
> The Account API is supported by this SDK, but not yet available in a
> released version of the CoCart plugin — it's coming soon. This
> documentation is ready for when it ships.

The Account API gives an authenticated customer access to their own profile, password, address, order history, downloads, reviews and subscriptions via the fixed `cocart/v2/my-account` route, plus methods that don't need a login: registering a new customer, looking up a guest order and requesting an order withdrawal.

- All other methods require the customer to be authenticated (Basic Auth or JWT) — see [Authentication](authentication.md).
- If the CoCart plugin providing a route isn't installed, a `cocart_plugin_required` error is returned.
- On a server, use one client per customer: a client holds one customer's login or cart key.

```go
account := client.Account()
```

## How Responses Look

Every account response is a document that says what it is in `object` (`customer`, `order`, `subscription`…). A request that changes something puts what it did first, in `result`: a `code` to branch on and a `message` for the customer. An order or subscription has `allowed_actions`: what the customer may do with it now, as a list of names (e.g. `pay`, `cancel`, `suspend`).

Amounts are minor-unit strings (`"4999"`), as in the cart. Methods that read a document take optional parameters as a `map[string]string`: `"formatted": "true"` for amounts ready to display (`"$49.99"`), and `"meta_keys"` (comma-joined, e.g. `"gift_message,_configuration"`) to narrow the order or customer `meta` to the keys you need.

## Register

Registers a new customer account. Unlike the other account methods, `Register` doesn't need the customer to be logged in: it's the entry point before authentication.

```go
client := cocart.NewClient("https://your-store.com")

response, err := client.Account().Register(ctx, cocart.RegisterCustomerInput{
	Email:    "jane@example.com",
	Username: "jane",              // optional if the store generates usernames from email
	Password: "a-strong-password", // optional if the store generates passwords
})

response.StatusCode                    // 201
response.GetString("result.code", "")  // "account_created"
response.GetInt("id", 0)               // 42
response.GetString("username", "")     // "jane"

// Log in next.
client.Login(ctx, "jane", "a-strong-password")
```

The response is the new customer, as `GetProfile` gives it. The username and password are sent as `requested_username` and `requested_password`, not `username` and `password`: CoCart treats a request with non-empty `username`/`password` fields as a Basic Auth attempt, which would reject the registration.

| Status | Code | Meaning |
|--------|------|---------|
| `400` | `cocart_invalid_email` | Email missing or invalid |
| `400` | `cocart_registration_invalid_username` | The username isn't valid |
| `400` | `cocart_registration_reserved_username` | The username is on the store's reserved list (e.g. `admin`) |
| `400` | `cocart_registration_missing_password` | The store needs a password and none was given |
| `400` | `cocart_registration_weak_password` | The password is too weak |
| `400` | `cocart_account_creation_failed` | WooCommerce couldn't create the account |
| `409` | `cocart_account_exists` | The email is already registered |
| `409` | `cocart_registration_username_exists` | The username is already taken |

## Profile

### Get Profile

The signed-in customer: name, email, addresses and what they've bought.

```go
customer, err := client.Account().GetProfile(ctx)

customer.GetString("first_name", "")           // "Jane"
customer.GetString("email", "")                // "jane@example.com"
customer.Get("pending_email")                  // a new email waiting to be confirmed, or nil
customer.GetInt("stats.orders_count", 0)       // 14
customer.GetString("stats.total_spent", "")    // "32000" (minor units)
customer.GetString("billing.formatted", "")    // the billing address on one line
customer.Get("recent_orders")                  // the 5 most recent orders, newest first: [{id, number, status, date_created, total, href}], [] when none

// Amounts ready to display, and only the meta keys you need
customer, err = client.Account().GetProfile(ctx, map[string]string{"formatted": "true", "meta_keys": "loyalty_tier"})
```

### Update Profile

Saves the account details. Send only what changes. It returns the customer with `account_updated` in `result`, and every field at fault is reported at once:

```go
customer, err := client.Account().UpdateProfile(ctx, map[string]any{
	"account_first_name": "Jane",
	"account_email":      "jane.doe@example.com",
})

var cocartErr *cocart.CoCartError
if errors.As(err, &cocartErr) {
	for field, message := range cocartErr.Fields() {
		showFieldError(field, message)
	}
}
```

The fields are `account_first_name`, `account_last_name`, `account_display_name` and `account_email`. When the store asks for it, an email change is confirmed by email first (`pending_email`), then with `VerifyEmail`.

### Change Password

```go
response, err := client.Account().ChangePassword(ctx, "old-password", "new-password", "new-password")
```

It's one `POST /my-account` with only the password fields (`password_current`, `password_1`, `password_2`); the rest of the account is left as it is. A wrong current password is a `*ValidationError` with code `cocart_account_incorrect_current_password`; passwords that don't match are `cocart_account_new_passwords_do_not_match`.

### Update Address

Saves the customer's billing and/or shipping address, and returns the customer. Fields are unprefixed:

```go
address := map[string]any{"first_name": "Jane", "address_1": "1 Main St", "city": "Paris", "postcode": "75001", "country": "FR"}
response, err := client.Account().UpdateAddress(ctx, map[string]any{
	"billing_address":  address,
	"shipping_address": address,
})
```

Both addresses are checked before either is saved; every field at fault is in `Fields()` (`cocart_invalid_address_fields`).

## Orders

### List Orders

A page of the customer's orders, newest first, each as a summary, with `pagination` (also in the `X-WP-Total` and `X-WP-TotalPages` headers):

```go
response, err := client.Account().GetOrders(ctx, nil)

orders, _ := response.Get("orders").([]any)
for _, o := range orders {
	order := o.(map[string]any)
	fmt.Println(order["id"], order["number"], order["total"]) // total in minor units: "2300"
	fmt.Println(order["status"].(map[string]any)["code"])     // "completed" (use this in code; "label" is for display)
	fmt.Println(order["allowed_actions"])                     // what the customer may do now: [pay cancel], or []
}
response.GetInt("pagination.total_items", 0)
response.GetTotalPages()

// Processing and on-hold orders from this year, 20 per page
response, err = client.Account().GetOrders(ctx, map[string]string{
	"status":   "processing,on-hold",
	"after":    "2026-01-01T00:00:00",
	"per_page": "20",
})
```

| Parameter | Value | Default |
|---|---|---|
| `page` | Page number | `1` |
| `per_page` | Results per page | `10` |
| `order` | `"ASC"` or `"DESC"` | `"DESC"` |
| `status` | One status, or several comma-joined (`"processing,on-hold"`) | every status |
| `after`, `before` | Only orders placed after/before this date (ISO 8601; without an offset it's UTC) | — |
| `subscription_order_type` | `"parent"`, `"renewal"`, `"switch"` or `"resubscribe"`: only orders with this relationship to a subscription | — |
| `formatted` | `"true"` for amounts formatted for display | `"false"` |
| `meta_keys` | Only these keys in each `meta`, comma-joined (`"gift_message,_configuration"`) | every allowed key |

### Get a Single Order

The whole order: what was bought, the totals as lines to display, the payment, the customer, the delivery (its shipping lines), the files it can download, notes, the parent order and what the customer may do with it now (`allowed_actions`).

```go
order, err := client.Account().GetOrder(ctx, 1042)

order.GetString("status.code", "")      // "completed"
order.Get("items")                      // each line with its product, quantity, price and totals
order.Get("items.0.details")            // [{label, value}] to print under the item: [{"label": "Size", "value": "L"}]
order.Get("total_lines")                // [{key, label, amount, detail}] in display order
order.GetString("payment.title", "")    // "Credit card"
order.Get("payment.needs_payment")      // true while it can still be paid
order.Get("delivery.shipping_lines")    // [{title, total, tax}]
order.Get("downloads")                  // the files this order can download
order.Get("parent_order")               // {id, number, href}, or nil
order.Get("allowed_actions")            // e.g. ["pay", "cancel"], or [] when there's nothing to do

order, err = client.Account().GetOrder(ctx, 1042, map[string]string{"formatted": "true"})
```

Someone else's order, or one that doesn't exist, is `404 cocart_order_not_found`.

### Get a Guest Order

For guest order look-up pages: no login, but the order ID and billing email must match. The order comes without its `key` (which would let the holder pay for it), and look-ups are rate limited (`429 cocart_guest_lookup_rate_limited`):

```go
response, err := client.Account().GetGuestOrder(ctx, 1042, "jane@example.com")
```

### Cancel an Order

Cancels one of the customer's orders where WooCommerce allows it (for example a pending or failed order), and returns the order with what happened in `result`:

```go
order, err := client.Account().CancelOrder(ctx, 1042)
order.GetString("status.code", "") // "cancelled"
```

An order that can't be cancelled is `409 cocart_order_cannot_be_cancelled`.

## Downloads

### All Downloads

Every file the customer can download, across all orders, as a list:

```go
response, err := client.Account().GetDownloads(ctx)

downloads, err := cocart.Unmarshal[[]map[string]any](response)
for _, file := range downloads {
	fmt.Println(file["name"], file["downloads_remaining"], file["href"]) // downloads_remaining is nil when unlimited
}
```

### Downloads for an Order

The files one order can download (`GET /my-account/orders/{id}/downloads`), in the same shape:

```go
response, err := client.Account().GetOrderDownloads(ctx, 1042)

// For a guest order look-up page: no login, but the order ID and billing email must match
response, err := client.Account().GetGuestOrderDownloads(ctx, 1042, "jane@example.com")
```

## Reviews

Returns the reviews the customer has submitted (`GET /products/reviews/mine`, the same as `Products().MyReviews()`):

```go
response, err := client.Account().GetReviews(ctx)
```

## Email Verification

```go
// Confirm a pending email change, with the key and user ID from the emailed link.
change, err := client.Account().VerifyEmail(ctx, key, userID)
change.GetString("email", "") // the address the account now has

// Prove ownership of the account email so WooCommerce links past guest orders to it (WooCommerce 11+).
sent, err := client.Account().SendOrderEmailVerification(ctx)
sent.GetInt("retry_after", 0) // seconds before another email can be asked for
client.Account().VerifyOrderEmail(ctx, key, userID) // returns the customer
```

An expired or wrong key is `400 cocart_invalid_or_expired_key`; an email already verified is `409 cocart_email_already_verified`.

## Subscriptions

Needs WooCommerce Subscriptions.

```go
list, err := client.Account().GetSubscriptions(ctx, map[string]string{"status": "active", "per_page": "10"})
one, err := client.Account().GetSubscription(ctx, 512)

client.Account().SuspendSubscription(ctx, 512)
client.Account().ReactivateSubscription(ctx, 512)
client.Account().CancelSubscription(ctx, 512)
client.Account().SetSubscriptionAutoRenew(ctx, 512, false)

// Resubscribe puts the ended subscription's items in the cart; check out as usual, or undo it.
client.Account().Resubscribe(ctx, 512)
client.Account().CancelResubscribe(ctx, 512)
```

`GetSubscriptions` takes `page`, `per_page`, `order`, `status` (one, or several comma-joined), `formatted` and `meta_keys`, and returns `subscriptions` with `pagination`. Each change returns the subscription with what happened in `result`. A change the subscription's state refuses is a `409` (e.g. `cocart_subscription_cannot_suspend`).

## Changing a Subscription's Payment Method

> Requires WooCommerce Subscriptions. The customer must be logged in.

A subscription can switch to a payment method the customer has saved, or to a new card. Use a gateway that can charge renewals: `SupportsSubscriptions` on a [saved payment method](checkout.md#saved-payment-methods) says so.

`UpdateSubscriptionPaymentMethod` can take payment like checkout, so it returns the same [`CheckoutResult`](checkout.md#reading-the-result), read from the subscription it returns (`Object` is `"subscription"`, with `SubscriptionID`). A new card may need 3D Secure.

**With a saved payment method:**

```go
methods, err := client.Checkout().GetSavedPaymentMethods(ctx, "")
// Show the methods with SupportsSubscriptions (method.Label: "Visa ending in 4242"); the customer picks one.

result, err := client.Account().UpdateSubscriptionPaymentMethod(ctx, 512, map[string]any{
	"payment_method": method.GatewayID,
	"token_id":       method.ID,
})
```

**With a new card**, create a payment method in the browser as in the [Quick Start](checkout.md#quick-start), with Elements in `mode: 'setup'` (nothing is charged now):

```go
result, err := client.Account().UpdateSubscriptionPaymentMethod(ctx, 512, map[string]any{
	"payment_method": "stripe",
	"payment_data":   map[string]any{"wc-stripe-payment-method": r.FormValue("payment_method_id")},
})
```

The new card is saved to the customer's account, since the renewals charge it.

**Reading the result:**

```go
switch {
case result.IsComplete():
	// Code is cocart.PaymentMethodChanged; result.Message says which method it now renews with.
	result.SubscriptionID                                 // 512
	result.Response.GetString("payment.title", "")        // "Visa ending in 4242"
case result.NeedsConfirmation():
	// Usually a SetupIntent (result.IntentType is "setup_intent"): confirm it in the browser,
	// then call UpdateSubscriptionPaymentMethod again with the same input.
case result.NeedsRedirect():
	// A bank that authorises a mandate off-site. When the customer comes back,
	// ConfirmReturn gives the subscription (result.Object is "subscription").
	http.Redirect(w, r, result.RedirectURL, http.StatusSeeOther)
case result.IsPending():
	// Not confirmed yet: GetSubscription(ctx, 512) shows the new method once it is.
default:
	fmt.Println(result.Message) // a declined card is a failed result, not an error
}
```

Confirm as in [3D Secure in the Page](checkout.md#3d-secure-in-the-page); `stripe.handleNextAction()` handles a SetupIntent too.

| Code (error) | HTTP | Meaning |
|---|---|---|
| `cocart_subscription_not_found` | 404 | No such subscription for this customer |
| `cocart_subscription_payment_method_change_unsupported` | 400 | The subscription's payment method can't be changed |
| `cocart_subscription_cannot_change_payment_method` | 409 | The subscription's state doesn't allow it now |
| `cocart_subscription_payment_token_required` | 400 | Pass a saved method's `token_id`, or a new card in `payment_data` |
| `cocart_invalid_payment_method` | 400 | The payment method isn't usable for this |
| `cocart_subscription_payment_token_needs_confirmation` | 400 | The saved method must be confirmed before it can pay renewals |
| `cocart_payment_in_progress` | 409 | Another change is still taking payment for this subscription: check it before trying again |

## Order Withdrawal

Submits a right-of-withdrawal request (WooCommerce's order withdrawal feature, WooCommerce 11.1+). No login needed; the response is the same whether or not the order matched, with `withdrawal_requested` in `result` and the email the acknowledgement went to:

```go
client.Account().RequestOrderWithdrawal(ctx, map[string]any{
	"first_name":         "Jane",
	"last_name":          "Doe",
	"email":              "jane@example.com",
	"email_confirmation": "jane@example.com",
	"order_number":       "1042",
	"withdrawal_type":    "full_order", // or "specific_items_only" with additional_details
})
```

Without the feature on, it's `404 cocart_order_withdrawal_unavailable`.

See [Error Handling](error-handling.md) for handling API errors.
