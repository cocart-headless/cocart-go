package cocart

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	checkoutRoute = "cocart/v2/checkout"
	checkoutAPI   = "cocart/v2"
)

// PaymentResultCode is what a call that takes payment did: the response's
// result.code (CoCart Plus 2.0). It's a string type, so a value a store or
// gateway adds comes through as it is.
type PaymentResultCode string

// Result codes CoCart Plus reports.
const (
	// PaymentOrderPlaced: paid, nothing to pay, or held for review or offline payment.
	PaymentOrderPlaced PaymentResultCode = "order_placed"
	// PaymentActionRequired: the customer must do something first, e.g. 3D Secure.
	PaymentActionRequired PaymentResultCode = "action_required"
	// PaymentPending: the gateway hasn't confirmed the payment yet.
	PaymentPending PaymentResultCode = "payment_pending"
	// PaymentFailed: the gateway declined the payment.
	PaymentFailed PaymentResultCode = "payment_failed"
	// PaymentMethodChanged: a subscription's payment method changed.
	PaymentMethodChanged PaymentResultCode = "payment_method_changed"
)

// CheckoutResult is the outcome of a call that takes payment: placing an
// order, paying for an unpaid order, returning from an off-site step, or
// changing a subscription's payment method (CoCart Plus 2.0). It's read from
// the response's result.
//
// A declined card is a result (IsFailed), not an error. Errors are kept for
// requests the store refuses before it tries to take payment.
//
// Printing a result (%v, %+v, %#v) never shows ClientSecret or OrderKey.
type CheckoutResult struct {
	// Code is result.code: PaymentOrderPlaced, PaymentActionRequired,
	// PaymentPending, PaymentFailed, or PaymentMethodChanged for a
	// subscription. A response without one reads as PaymentFailed, never as
	// complete.
	Code PaymentResultCode
	// Object is the document returned: "order", or "subscription" after a
	// subscription's payment method change.
	Object string
	// OrderID, OrderKey, OrderNumber and OrderStatus describe the order, when
	// the document is an order. OrderKey is needed to view or pay for it as a
	// guest; OrderNumber is for display and can differ from the ID;
	// OrderStatus is the status code, e.g. "processing", "on-hold" or "pending".
	OrderID     int
	OrderKey    string
	OrderNumber string
	OrderStatus string
	// SubscriptionID is the subscription's ID, when the document is a subscription.
	SubscriptionID int
	// RedirectURL is where to send the customer: the off-site step when
	// NeedsRedirect, otherwise the page to show once the payment is done (the
	// order-received page, or the gateway's own). Only ever an https or http
	// address.
	RedirectURL string
	// ClientSecret and IntentType ("payment_intent" or "setup_intent") are for
	// an in-page step (3D Secure), or a payment the customer makes in the page:
	// give them to the gateway's SDK.
	ClientSecret string
	IntentType   string
	// PaymentMethodType is the Stripe payment method to pay with when
	// IsAwaitingPayment: cashapp, wechat_pay, boleto, oxxo or multibanco.
	PaymentMethodType string
	// Message is for the customer, for every code: why a payment failed, or
	// that the order was received. Plain text.
	Message string
	// ErrorCode is CoCart's code for why the payment failed: card_declined,
	// insufficient_funds… or payment_failed.
	ErrorCode string
	// GatewayErrorCode and GatewayMessage are the gateway's own, for logging.
	GatewayErrorCode string
	GatewayMessage   string
	// ActionType is what the customer must do, as the gateway integration
	// names it, e.g. stripe_confirm_payment or stripe_voucher_payment, and
	// ActionData what the action needs, e.g. client_secret or redirect.
	ActionType string
	ActionData map[string]any
	// Response is the raw response: the order, or the subscription.
	Response *Response
}

// IsComplete reports whether the order is placed (paid, nothing to pay, or
// held for review or offline payment), or the subscription's payment method
// changed, with nothing left for the customer to do.
func (r *CheckoutResult) IsComplete() bool {
	return (r.Code == PaymentOrderPlaced || r.Code == PaymentMethodChanged) && r.ActionType == ""
}

// IsAwaitingPayment reports whether the order is placed but not paid yet: the
// customer pays outside the checkout. With ClientSecret, pay in the page with
// the gateway's SDK for PaymentMethodType (Cash App Pay, WeChat Pay, a Boleto,
// OXXO or Multibanco voucher); otherwise ActionType says what to show
// (stripe_blik_confirmation_pending: approve in the banking app; bank transfer
// instructions…). The gateway's webhook marks the order paid, so don't place
// the order again.
func (r *CheckoutResult) IsAwaitingPayment() bool {
	return r.Code == PaymentOrderPlaced && r.ActionType != ""
}

// NeedsConfirmation reports whether the card needs an in-page step (3D
// Secure). Confirm ClientSecret with the gateway's SDK, then make the same
// call again.
func (r *CheckoutResult) NeedsConfirmation() bool {
	return r.Code == PaymentActionRequired && r.ClientSecret != ""
}

// NeedsRedirect reports whether the payment needs an off-site step. Send the
// customer to RedirectURL, then call ConfirmReturn when they come back.
func (r *CheckoutResult) NeedsRedirect() bool {
	return r.Code == PaymentActionRequired && r.ClientSecret == "" && webURLOf(r.ActionData["redirect"]) != ""
}

// IsPending reports whether the gateway hasn't confirmed the payment yet.
// Confirm again shortly, or check the order.
func (r *CheckoutResult) IsPending() bool {
	return r.Code == PaymentPending
}

// IsFailed reports whether the payment was declined or couldn't be taken.
// Show Message; the order can be paid again.
func (r *CheckoutResult) IsFailed() bool {
	return r.Code == PaymentFailed
}

// String describes the result for %v and %+v. It leaves out ClientSecret,
// OrderKey, ActionData and the response, so logging a result doesn't leak them.
func (r CheckoutResult) String() string {
	return fmt.Sprintf("CheckoutResult{Code:%s Object:%s OrderID:%d OrderNumber:%s OrderStatus:%s SubscriptionID:%d ActionType:%s Message:%q ErrorCode:%s}",
		r.Code, r.Object, r.OrderID, r.OrderNumber, r.OrderStatus, r.SubscriptionID, r.ActionType, r.Message, r.ErrorCode)
}

// GoString describes the result for %#v. Like String, it leaves out
// ClientSecret, OrderKey, ActionData and the response.
func (r CheckoutResult) GoString() string {
	return fmt.Sprintf("cocart.CheckoutResult{Code:%q, Object:%q, OrderID:%d, OrderNumber:%q, OrderStatus:%q, SubscriptionID:%d, RedirectURL:%q, IntentType:%q, PaymentMethodType:%q, Message:%q, ErrorCode:%q, GatewayErrorCode:%q, GatewayMessage:%q, ActionType:%q}",
		string(r.Code), r.Object, r.OrderID, r.OrderNumber, r.OrderStatus, r.SubscriptionID, r.RedirectURL, r.IntentType, r.PaymentMethodType, r.Message, r.ErrorCode, r.GatewayErrorCode, r.GatewayMessage, r.ActionType)
}

// SavedPaymentMethod is a logged-in customer's saved payment method.
type SavedPaymentMethod struct {
	// ID goes in payment_data as wc-{GatewayID}-payment-token, or as token_id
	// to change a subscription's card.
	ID int
	// GatewayID is the payment method's gateway, e.g. "stripe".
	GatewayID string
	// Label is the method as the customer sees it, e.g. "Visa ending in 4242".
	Label string
	Type  string
	// Expires is MM/YYYY, or "" when it has no expiry.
	Expires   string
	IsDefault bool
	// SupportsSubscriptions reports whether the gateway can charge renewals with it.
	SupportsSubscriptions bool
}

// PaymentDataPairs turns a payment_data map into the {key, value} pairs the
// API takes, sorted by key. Any other value passes through as it is.
func PaymentDataPairs(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, map[string]any{"key": k, "value": m[k]})
	}
	return pairs
}

func withPaymentDataPairs(input map[string]any) map[string]any {
	body := make(map[string]any, len(input))
	for k, v := range input {
		body[k] = v
	}
	if data, ok := body["payment_data"]; ok {
		body["payment_data"] = PaymentDataPairs(data)
	}
	return body
}

// firstParams returns the first of a method's optional parameter maps, or nil.
func firstParams(params []map[string]string) map[string]string {
	if len(params) > 0 {
		return params[0]
	}
	return nil
}

// copyQuery returns a copy of query parameters, so adding to it never
// changes the caller's map.
func copyQuery(params map[string]string) map[string]string {
	query := make(map[string]string, len(params)+1)
	for k, v := range params {
		query[k] = v
	}
	return query
}

func textOf(v any) string {
	s, _ := v.(string)
	return s
}

var webURLPattern = regexp.MustCompile(`(?i)^https?://[^\s/?#]`)

// webURLOf returns v when it is a web address the customer can be sent to:
// https or http only. Anything else (javascript:, data:, another app's link,
// a relative path) is left out, so a response can't make a page run script
// or open another app.
func webURLOf(v any) string {
	if s := textOf(v); webURLPattern.MatchString(s) {
		return s
	}
	return ""
}

func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// numberText returns a string as it is, or a whole number as a string.
func numberText(v any) string {
	if n, ok := v.(float64); ok && n == float64(int(n)) {
		return strconv.Itoa(int(n))
	}
	return textOf(v)
}

// checkoutResultFromResponse reads the document a call that takes payment
// returns, with what the payment did in result.
func checkoutResultFromResponse(resp *Response) *CheckoutResult {
	data := resp.ToObject()
	result := mapOf(data["result"])
	action := mapOf(result["action"])
	actionData := mapOf(action["data"])
	failure := mapOf(result["error"])
	object := textOf(data["object"])

	// A missing code fails safe: never "complete".
	code := PaymentResultCode(textOf(result["code"]))
	if code == "" {
		code = PaymentFailed
	}

	// With action_required, the off-site step comes first; otherwise the page
	// to show once the payment is done.
	offSite := webURLOf(actionData["redirect"])
	redirect := webURLOf(result["redirect_url"])
	if (code == PaymentActionRequired && offSite != "") || redirect == "" {
		redirect = offSite
	}

	r := &CheckoutResult{
		Code:              code,
		Object:            object,
		RedirectURL:       redirect,
		ClientSecret:      textOf(actionData["client_secret"]),
		IntentType:        textOf(actionData["intent_type"]),
		PaymentMethodType: textOf(actionData["payment_method_type"]),
		Message:           textOf(result["message"]),
		ErrorCode:         textOf(failure["code"]),
		GatewayErrorCode:  textOf(failure["gateway_code"]),
		GatewayMessage:    textOf(failure["gateway_message"]),
		ActionType:        textOf(action["type"]),
		ActionData:        actionData,
		Response:          resp,
	}
	if object == "subscription" {
		r.SubscriptionID = intOf(data["id"])
	} else {
		r.OrderID = intOf(data["id"])
		r.OrderKey = textOf(data["key"])
		r.OrderNumber = numberText(data["number"])
		r.OrderStatus = textOf(mapOf(data["status"])["code"])
	}
	return r
}

// CheckoutEndpoint places orders and takes payment through CoCart Plus 2.0's
// checkout routes. Card details never reach this code: the browser turns them
// into a payment method ID with the gateway's own JavaScript, and the order
// is placed with that ID.
type CheckoutEndpoint struct {
	client *Client
}

// Checkout returns the checkout endpoint (CoCart Plus 2.0).
func (c *Client) Checkout() *CheckoutEndpoint {
	return &CheckoutEndpoint{client: c}
}

// Get returns the checkout as it stands, like a receipt: items, coupons,
// fees, totals and total_lines, the payment method, the addresses, and each
// shipping package with its rates. params optionally takes currency,
// formatted and thumb.
//
// Call it for the page the customer reviews before paying: CoCart Plus
// remembers the total it showed, and refuses the order if the total has
// changed by then (409 cocart_checkout_total_mismatch). Don't call it just
// before placing the order.
func (e *CheckoutEndpoint) Get(ctx context.Context, params ...map[string]string) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodGet, checkoutRoute, copyQuery(firstParams(params)), nil)
}

// Update saves addresses, the shipping method, the payment method, a coupon
// (coupon_code with coupon_action "apply" or "remove") or the currency,
// without placing the order. It returns the checkout with what it did in
// result: code "checkout_updated", or "checkout_partly_updated" with what
// wasn't applied, and why, in result.not_applied ([{field, code, message}]).
func (e *CheckoutEndpoint) Update(ctx context.Context, data map[string]any) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodPut, checkoutRoute, nil, data)
}

// Process creates the order and attempts payment, sending the input as
// given: the low-level call behind PlaceOrder.
func (e *CheckoutEndpoint) Process(ctx context.Context, data map[string]any) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodPost, checkoutRoute, nil, data)
}

// checkoutIDPattern is the checkout_id CoCart Plus accepts: 16 to 64
// letters, digits, "-" or "_".
var checkoutIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)

// PlaceOrder creates the order and takes payment.
//
// payment_data is a map of the keys the WooCommerce gateway reads, e.g.
// {"wc-stripe-payment-method": "pm_…"}: gateway tokens only, never card
// details. A free order needs neither it nor payment_method. Calling again
// for the same cart lands on the same order. A declined card is a failed
// result; errors before payment are returned.
//
// The order is sent with a random checkout_id, or the one in input, which
// must be 16 to 64 letters, digits, "-" or "_" (else a
// checkout_invalid_checkout_id error, before anything is sent). When no
// answer comes back, or another request is still taking payment for the cart
// (409 cocart_payment_in_progress), the order may have been placed: the error
// is a *CheckoutOutcomeUnknownError, and GetStatus with its CheckoutID tells
// how it ended.
//
// A total that changed since Get or Update is 409
// cocart_checkout_total_mismatch: get the checkout again and show the new
// total.
func (e *CheckoutEndpoint) PlaceOrder(ctx context.Context, input map[string]any) (*CheckoutResult, error) {
	// Left out (or ""), a random one is sent; anything else must be valid.
	checkoutID, isText := input["checkout_id"].(string)
	if input["checkout_id"] == nil || (isText && checkoutID == "") {
		checkoutID = newCheckoutID()
	}
	if !checkoutIDPattern.MatchString(checkoutID) {
		return nil, NewCoCartError(`checkout_id must be 16 to 64 letters, digits, "-" or "_".`, 0, "checkout_invalid_checkout_id")
	}
	body := withPaymentDataPairs(input)
	body["checkout_id"] = checkoutID
	if keys, ok := body["meta_keys"].([]string); ok {
		body["meta_keys"] = strings.Join(keys, ",")
	}

	resp, err := e.Process(ctx, body)
	if err != nil {
		var cocartErr *CoCartError
		if errors.As(err, &cocartErr) && (isLostResponse(cocartErr) || isPaymentInProgress(cocartErr)) {
			return nil, &CheckoutOutcomeUnknownError{CoCartError: cocartErr, CheckoutID: checkoutID}
		}
		return nil, err
	}
	return checkoutResultFromResponse(resp), nil
}

// CheckoutOutcomeUnknownError is returned by PlaceOrder when the order was
// sent but how it ended isn't known: no answer came back (the connection
// dropped, the request timed out, or a proxy gave up waiting), or another
// request is still taking payment for the cart (409
// cocart_payment_in_progress). The order may or may not have been placed.
// Don't place it again blindly: GetStatus(ctx, err.CheckoutID) tells how it
// ended. It wraps the original *CoCartError.
type CheckoutOutcomeUnknownError struct {
	*CoCartError
	// CheckoutID is the checkout_id the order was sent with.
	CheckoutID string
}

// Unwrap returns the original error.
func (e *CheckoutOutcomeUnknownError) Unwrap() error {
	return e.CoCartError
}

// CheckoutStatus is how a checkout ended, from GetStatus.
type CheckoutStatus struct {
	CheckoutID string
	// Code is "order_placed" (show the order), "payment_required" (placed but
	// not paid: pay it with PayForOrder), "in_progress" (ask again in a few
	// seconds) or "not_placed" (place the order again).
	Code string
	// Message describes the outcome, for the customer.
	Message string
	// OrderID, OrderKey, OrderNumber and OrderStatus (the order's status
	// code, e.g. "processing") are zero unless the order was placed or needs
	// payment.
	OrderID     int
	OrderKey    string
	OrderNumber string
	OrderStatus string
}

// GetStatus tells how a checkout ended, when PlaceOrder returned a
// *CheckoutOutcomeUnknownError. Ask from the same cart, or as the same
// customer.
func (e *CheckoutEndpoint) GetStatus(ctx context.Context, checkoutID string) (*CheckoutStatus, error) {
	resp, err := e.client.RequestRaw(ctx, http.MethodGet, checkoutRoute+"/status", map[string]string{"checkout_id": checkoutID}, nil)
	if err != nil {
		return nil, err
	}
	data := resp.ToObject()
	result := mapOf(data["result"])
	order := mapOf(data["order"])
	status := &CheckoutStatus{
		CheckoutID:  textOf(data["checkout_id"]),
		Code:        textOf(result["code"]),
		Message:     textOf(result["message"]),
		OrderID:     intOf(order["id"]),
		OrderKey:    textOf(order["key"]),
		OrderNumber: numberText(order["number"]),
		OrderStatus: textOf(mapOf(order["status"])["code"]),
	}
	if status.CheckoutID == "" {
		status.CheckoutID = checkoutID
	}
	return status, nil
}

// newCheckoutID returns a random ID for one checkout: "chk_" and 32 hex digits.
func newCheckoutID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("chk_%032x", time.Now().UnixNano())
	}
	return "chk_" + hex.EncodeToString(b)
}

// isLostResponse reports whether no answer came back: the connection
// dropped, it timed out, or a proxy gave up.
func isLostResponse(err *CoCartError) bool {
	if err.HTTPCode == 0 {
		return err.ErrorCode == "network_error" || err.ErrorCode == "request_timeout"
	}
	return err.HTTPCode == http.StatusBadGateway || err.HTTPCode == http.StatusGatewayTimeout
}

// isPaymentInProgress reports whether another request is still taking
// payment for the same cart, so how this one ends isn't known yet.
func isPaymentInProgress(err *CoCartError) bool {
	return err.HTTPCode == http.StatusConflict && err.ErrorCode == "cocart_payment_in_progress"
}

// GetConfig returns the store's form settings: fields, countries, shipping,
// account and store settings, order_received_url for ConfirmReturn, whether
// address search is available (address_autocomplete.enabled), and the terms
// text with its links. No cart needed.
func (e *CheckoutEndpoint) GetConfig(ctx context.Context) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodGet, checkoutRoute+"/config", nil, nil)
}

// GetPaymentMethods returns the payment methods for this cart: a list, each
// with its saved_tokens. Read it with Unmarshal[[]map[string]any].
func (e *CheckoutEndpoint) GetPaymentMethods(ctx context.Context) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodGet, checkoutRoute+"/payment-methods", nil, nil)
}

// GetSavedPaymentMethods returns a logged-in customer's saved payment
// methods, for every gateway ("") or one, in the gateways' and methods'
// order. Pay with one as wc-{GatewayID}-payment-token in payment_data, or
// change a subscription's card with its ID as token_id. Empty for a guest.
func (e *CheckoutEndpoint) GetSavedPaymentMethods(ctx context.Context, gatewayID string) ([]SavedPaymentMethod, error) {
	resp, err := e.GetPaymentMethods(ctx)
	if err != nil {
		return nil, err
	}
	methods, _ := Unmarshal[[]any](resp)

	cards := []SavedPaymentMethod{}
	for _, m := range methods {
		method := mapOf(m)
		id := textOf(method["id"])
		if id == "" || (gatewayID != "" && id != gatewayID) {
			continue
		}
		supportsSubscriptions, _ := mapOf(method["supports"])["subscriptions"].(bool)
		tokens, _ := method["saved_tokens"].([]any)
		for _, t := range tokens {
			token := mapOf(t)
			if token == nil {
				continue
			}
			isDefault, _ := token["is_default"].(bool)
			cards = append(cards, SavedPaymentMethod{
				ID:                    intOf(token["id"]),
				GatewayID:             id,
				Label:                 textOf(token["label"]),
				Type:                  textOf(token["type"]),
				Expires:               textOf(token["expires"]),
				IsDefault:             isDefault,
				SupportsSubscriptions: supportsSubscriptions,
			})
		}
	}
	return cards, nil
}

// SearchAddresses returns address suggestions. Params: query (at least 3
// characters), and optionally country, type and provider. No cart needed.
// Without a server-side address provider it's 404
// cocart_address_search_no_provider: check GetConfig's
// address_autocomplete.enabled first.
func (e *CheckoutEndpoint) SearchAddresses(ctx context.Context, params map[string]string) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodGet, checkoutAPI+"/address/search", params, nil)
}

// GetAddressDetails returns a suggestion's full address. No cart needed.
func (e *CheckoutEndpoint) GetAddressDetails(ctx context.Context, addressID, provider string) (*Response, error) {
	return e.client.RequestRaw(ctx, http.MethodGet, checkoutAPI+"/address/details",
		map[string]string{"address_id": addressID, "provider": provider}, nil)
}

// GetOrderReceived returns the order, for the thank-you page. orderKey must
// match the order's key; a wrong key, or someone else's order, is 404
// cocart_order_not_found. params optionally takes formatted ("true") and
// meta_keys (comma-joined).
func (e *CheckoutEndpoint) GetOrderReceived(ctx context.Context, orderID int, orderKey string, params ...map[string]string) (*Response, error) {
	query := copyQuery(firstParams(params))
	query["order_key"] = orderKey
	return e.client.RequestRaw(ctx, http.MethodGet, fmt.Sprintf("%s/order-received/%d", checkoutAPI, orderID), query, nil)
}

// PayForOrder takes payment for an unpaid (pending or failed) order, e.g. to
// retry a failed payment. Read the result as for PlaceOrder: a card that
// needs 3D Secure gives NeedsConfirmation, then call again with the same
// input. A declined card is a failed result, not an error. The order key is
// sent in the body, not the URL. An order that needs no payment or can't be
// paid is a 409, and so is another payment still running for it
// (cocart_payment_in_progress).
func (e *CheckoutEndpoint) PayForOrder(ctx context.Context, orderID int, orderKey string, input map[string]any) (*CheckoutResult, error) {
	body := withPaymentDataPairs(input)
	body["order_key"] = orderKey
	resp, err := e.client.RequestRaw(ctx, http.MethodPost, fmt.Sprintf("%s/order-received/%d/pay", checkoutAPI, orderID), nil, body)
	if err != nil {
		return nil, err
	}
	return checkoutResultFromResponse(resp), nil
}

// ConfirmOrderPayment finishes a payment after an off-site step, from the
// parameters the customer came back with: pass every one unchanged. The order
// key is sent in the body. Safe to call more than once. ConfirmReturn does
// this straight from the return URL, and reads the result.
func (e *CheckoutEndpoint) ConfirmOrderPayment(ctx context.Context, orderID int, orderKey string, returnParams map[string]string) (*Response, error) {
	if returnParams == nil {
		returnParams = map[string]string{}
	}
	return e.client.RequestRaw(ctx, http.MethodPost,
		fmt.Sprintf("%s/order-received/%d/confirm", checkoutAPI, orderID), nil,
		map[string]any{"order_key": orderKey, "return_params": returnParams})
}

// ConfirmReturn finishes a payment from the URL the customer returned to
// after an off-site step, e.g. r.URL.String(). It reads the order ID
// (order_id) and key (key) from the URL and forwards every query parameter.
// IsPending means the gateway hasn't confirmed yet: call again shortly. Safe
// to call more than once.
func (e *CheckoutEndpoint) ConfirmReturn(ctx context.Context, returnURL string) (*CheckoutResult, error) {
	u, err := url.Parse(returnURL)
	if err != nil {
		return nil, fmt.Errorf("cocart: invalid return URL: %w", err)
	}
	return e.ConfirmReturnFor(ctx, 0, "", u.Query())
}

// ConfirmReturnFor is ConfirmReturn for a route that carries the order
// elsewhere: orderID and orderKey are used when set, and read from params
// (the return URL's query) when not.
func (e *CheckoutEndpoint) ConfirmReturnFor(ctx context.Context, orderID int, orderKey string, params url.Values) (*CheckoutResult, error) {
	returnParams := make(map[string]string, len(params))
	for k := range params {
		returnParams[k] = params.Get(k)
	}
	if orderID == 0 {
		orderID, _ = strconv.Atoi(returnParams["order_id"])
	}
	if orderKey == "" {
		orderKey = returnParams["key"]
	}
	if orderID <= 0 || orderKey == "" {
		return nil, NewCoCartError("The return URL has no order ID or order key. Use ConfirmReturnFor if your route carries them elsewhere.", 0, "checkout_return_missing_order")
	}
	resp, err := e.ConfirmOrderPayment(ctx, orderID, orderKey, returnParams)
	if err != nil {
		return nil, err
	}
	return checkoutResultFromResponse(resp), nil
}
