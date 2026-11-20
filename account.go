package cocart

import (
	"context"
	"fmt"
	"net/http"
)

const (
	accountBase     = "cocart/v2/my-account"
	registerRoute   = "cocart/v2/register"
	withdrawalRoute = "cocart/v2/order-withdrawal"
)

// AccountEndpoint handles the customer's account: profile, address, orders,
// downloads, reviews, subscriptions and email verification (CoCart Plus).
//
// All methods require the customer to be authenticated (Basic Auth or JWT),
// except Register, GetGuestOrder, GetGuestOrderDownloads and
// RequestOrderWithdrawal.
//
// Note: the CoCart Account API is not yet available in a released version of
// the CoCart plugin — this endpoint is implemented and ready for when it
// ships.
type AccountEndpoint struct {
	endpoint
}

func (e *AccountEndpoint) rawPath(sub string) string {
	if sub == "" {
		return accountBase
	}
	return accountBase + "/" + sub
}

func (e *AccountEndpoint) doRaw(ctx context.Context, method, route string, params map[string]string, data any) (*Response, error) {
	resp, err := e.client.RequestRaw(ctx, method, route, params, data)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

func (e *AccountEndpoint) doRawGet(ctx context.Context, sub string, params map[string]string) (*Response, error) {
	return e.doRaw(ctx, http.MethodGet, e.rawPath(sub), params, nil)
}

func (e *AccountEndpoint) doRawPost(ctx context.Context, sub string, data any) (*Response, error) {
	return e.doRaw(ctx, http.MethodPost, e.rawPath(sub), nil, data)
}

// --- Registration (CoCart Plus) ---

// RegisterCustomerInput is a new customer for [AccountEndpoint.Register].
// Username and Password are optional when the store generates them.
type RegisterCustomerInput struct {
	Email    string
	Username string
	Password string
}

// Register registers a new customer. It doesn't need the customer logged in.
//
// The username and password are sent as requested_username and
// requested_password: CoCart treats non-empty username and password fields
// as a Basic Auth attempt.
func (e *AccountEndpoint) Register(ctx context.Context, customer RegisterCustomerInput) (*Response, error) {
	data := map[string]any{"email": customer.Email}
	if customer.Username != "" {
		data["requested_username"] = customer.Username
	}
	if customer.Password != "" {
		data["requested_password"] = customer.Password
	}
	return e.doRaw(ctx, http.MethodPost, registerRoute, nil, data)
}

// --- Profile ---

// GetProfile returns the signed-in customer: name, email, addresses, order
// stats and actions. params optionally takes formatted and meta_keys (a
// []string).
func (e *AccountEndpoint) GetProfile(ctx context.Context, params ...map[string]any) (*Response, error) {
	return e.doRawGet(ctx, "", documentQuery(firstParams(params)))
}

// UpdateProfile updates the authenticated user's profile fields
// (account_first_name, account_last_name, account_display_name, account_email).
func (e *AccountEndpoint) UpdateProfile(ctx context.Context, data map[string]any) (*Response, error) {
	return e.doRawPost(ctx, "", data)
}

// ChangePassword changes the authenticated user's password (POST
// /my-account). It returns the customer with account_updated in result. A
// wrong current password is a *ValidationError
// (cocart_account_incorrect_current_password).
func (e *AccountEndpoint) ChangePassword(ctx context.Context, current, password, confirm string) (*Response, error) {
	return e.doRawPost(ctx, "", map[string]any{
		"password_current": current,
		"password_1":       password,
		"password_2":       confirm,
	})
}

// UpdateAddress saves the customer's billing and/or shipping address
// (billing_address, shipping_address), each with unprefixed fields
// (first_name, address_1, …).
func (e *AccountEndpoint) UpdateAddress(ctx context.Context, data map[string]any) (*Response, error) {
	return e.doRawPost(ctx, "address", data)
}

// --- Orders ---

// GetOrders returns the customer's orders, newest first, with pagination
// (also in X-WP-Total and X-WP-TotalPages). params optionally takes page,
// per_page, order ("ASC" or "DESC"), status (one, or a []string of
// statuses), after and before (ISO 8601 dates), subscription_order_type
// ("parent", "renewal", "switch" or "resubscribe"), formatted and meta_keys
// (a []string). Pass nil for none.
func (e *AccountEndpoint) GetOrders(ctx context.Context, params map[string]any) (*Response, error) {
	return e.doRawGet(ctx, "orders", documentQuery(params))
}

// GetOrder returns one of the customer's orders. Someone else's order is 404
// cocart_order_not_found. params optionally takes formatted and meta_keys.
func (e *AccountEndpoint) GetOrder(ctx context.Context, id int, params ...map[string]any) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d", id), documentQuery(firstParams(params)))
}

// GetGuestOrder returns a guest's order, looked up with its billing email.
// It comes without its key. Lookups are rate limited (429
// cocart_guest_lookup_rate_limited). params optionally takes formatted and
// meta_keys.
func (e *AccountEndpoint) GetGuestOrder(ctx context.Context, id int, email string, params ...map[string]any) (*Response, error) {
	query := queryOf(firstParams(params))
	query["email"] = email
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d", id), query)
}

// CancelOrder cancels one of the customer's orders where WooCommerce allows
// it (for example a pending or failed order).
func (e *AccountEndpoint) CancelOrder(ctx context.Context, id int) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("orders/%d/cancel", id), nil)
}

// --- Downloads ---

// GetOrderDownloads returns the files one of the customer's orders can
// download, as sent: {"downloads": [...]}.
func (e *AccountEndpoint) GetOrderDownloads(ctx context.Context, id int) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d/downloads", id), nil)
}

// GetGuestOrderDownloads returns the files a guest's order can download,
// looked up with its billing email, as sent: {"downloads": [...]}.
func (e *AccountEndpoint) GetGuestOrderDownloads(ctx context.Context, id int, email string) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d/downloads", id), map[string]string{"email": email})
}

// GetDownloads returns every file the customer can download: {"downloads": [...]}.
func (e *AccountEndpoint) GetDownloads(ctx context.Context) (*Response, error) {
	return e.doRawGet(ctx, "downloads", nil)
}

// --- Reviews ---

// GetReviews returns the customer's product reviews (GET
// /products/reviews/mine), the same as [ProductsEndpoint.MyReviews].
func (e *AccountEndpoint) GetReviews(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	return e.client.Products().MyReviews(ctx, params...)
}

// --- Email verification (CoCart Plus) ---

// VerifyEmail confirms a pending email change, with the key and user ID from
// the emailed link.
func (e *AccountEndpoint) VerifyEmail(ctx context.Context, key string, userID int) (*Response, error) {
	return e.doRawGet(ctx, "verify-email", map[string]string{"key": key, "user_id": fmt.Sprintf("%d", userID)})
}

// SendOrderEmailVerification emails the customer a link proving they own the
// account email, so WooCommerce links their past guest orders to it
// (WooCommerce 11+). The response's status is "sent", or "throttled" (once a
// minute).
func (e *AccountEndpoint) SendOrderEmailVerification(ctx context.Context) (*Response, error) {
	return e.doRawPost(ctx, "verify-order-email", nil)
}

// VerifyOrderEmail verifies the email with the key and user ID from the
// link [AccountEndpoint.SendOrderEmailVerification] sent.
func (e *AccountEndpoint) VerifyOrderEmail(ctx context.Context, key string, userID int) (*Response, error) {
	return e.doRawGet(ctx, "verify-order-email", map[string]string{"key": key, "user_id": fmt.Sprintf("%d", userID)})
}

// --- Subscriptions (CoCart Plus with WooCommerce Subscriptions) ---

// GetSubscriptions returns the customer's subscriptions, with pagination.
// params optionally takes page, per_page, order ("ASC" or "DESC"), status
// (one, or a []string of statuses), formatted and meta_keys (a []string).
// Pass nil for none.
func (e *AccountEndpoint) GetSubscriptions(ctx context.Context, params map[string]any) (*Response, error) {
	return e.doRawGet(ctx, "subscriptions", documentQuery(params))
}

// GetSubscription returns one subscription. Pass the email to view it as a guest.
func (e *AccountEndpoint) GetSubscription(ctx context.Context, id int, email ...string) (*Response, error) {
	var params map[string]string
	if len(email) > 0 && email[0] != "" {
		params = map[string]string{"email": email[0]}
	}
	return e.doRawGet(ctx, fmt.Sprintf("subscriptions/%d", id), params)
}

// SuspendSubscription suspends (puts on hold) a subscription.
func (e *AccountEndpoint) SuspendSubscription(ctx context.Context, id int) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/suspend", id), nil)
}

// CancelSubscription cancels a subscription.
func (e *AccountEndpoint) CancelSubscription(ctx context.Context, id int) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/cancel", id), nil)
}

// ReactivateSubscription reactivates a suspended subscription.
func (e *AccountEndpoint) ReactivateSubscription(ctx context.Context, id int) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/reactivate", id), nil)
}

// SetSubscriptionAutoRenew turns automatic renewal on or off.
func (e *AccountEndpoint) SetSubscriptionAutoRenew(ctx context.Context, id int, enabled bool) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/auto-renew", id), map[string]any{"enabled": enabled})
}

// UpdateSubscriptionPaymentMethod changes the card a subscription renews
// with: payment_method (gateway ID, e.g. "stripe"), and token_id (a saved
// card's ID) or payment_data (a new card, e.g.
// {"wc-stripe-payment-method": "pm_…"}).
//
// A new card may need 3D Secure, so this returns a CheckoutResult, read as at
// checkout: IsComplete once changed (PaymentMethodChanged, with
// SubscriptionID); on NeedsConfirmation, confirm ClientSecret with the
// gateway's SDK and call again with the same input; on NeedsRedirect, send
// the customer to RedirectURL. A declined card is a failed result, not an
// error. Another change still running is 409 cocart_payment_in_progress.
func (e *AccountEndpoint) UpdateSubscriptionPaymentMethod(ctx context.Context, id int, data map[string]any) (*CheckoutResult, error) {
	resp, err := e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/payment-method", id), withPaymentDataPairs(data))
	if err != nil {
		return nil, err
	}
	return checkoutResultFromResponse(resp), nil
}

// Resubscribe puts an ended subscription's items in the cart to subscribe
// again (checkout follows as usual).
func (e *AccountEndpoint) Resubscribe(ctx context.Context, id int) (*Response, error) {
	return e.doRawPost(ctx, fmt.Sprintf("subscriptions/%d/resubscribe", id), nil)
}

// CancelResubscribe undoes Resubscribe: it restores the cart as it was before.
func (e *AccountEndpoint) CancelResubscribe(ctx context.Context, id int) (*Response, error) {
	return e.doRaw(ctx, http.MethodDelete, e.rawPath(fmt.Sprintf("subscriptions/%d/resubscribe", id)), nil, nil)
}

// --- Order withdrawal (CoCart Plus) ---

// RequestOrderWithdrawal submits an order withdrawal (right of withdrawal)
// request. It doesn't need the customer logged in. data takes first_name,
// last_name, email, email_confirmation, order_number, and optionally
// withdrawal_type ("full_order" or "specific_items_only") and
// additional_details (required for specific_items_only).
func (e *AccountEndpoint) RequestOrderWithdrawal(ctx context.Context, data map[string]any) (*Response, error) {
	return e.doRaw(ctx, http.MethodPost, withdrawalRoute, nil, data)
}

// documentQuery returns the query for optional parameters (formatted,
// meta_keys and any others given), or nil when there are none. A list is
// comma-joined, and a boolean is "true" or "false".
func documentQuery(params map[string]any) map[string]string {
	query := queryOf(params)
	if len(query) == 0 {
		return nil
	}
	return query
}
