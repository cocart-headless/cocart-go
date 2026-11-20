package cocart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// The shared checkout fixtures (testdata/checkout, copied from cocart-js's
// packages/checkout/tests/fixtures): the same request/response data every
// CoCart SDK replays, so all of them are tested the same way.

type fixtureRequest struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Query  map[string]string `json:"query"`
	Body   any               `json:"body"`
}

type checkoutFixture struct {
	Call struct {
		Target string `json:"target"`
		Method string `json:"method"`
		Args   []any  `json:"args"`
	} `json:"call"`
	Exchanges []struct {
		Request  fixtureRequest `json:"request"`
		Response struct {
			Status int `json:"status"`
			Body   any `json:"body"`
		} `json:"response"`
	} `json:"exchanges"`
	Expect map[string]json.RawMessage `json:"expect"`
}

func TestCheckoutFixtures(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "checkout", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 53 {
		t.Fatalf("found %d fixtures, want 53", len(files))
	}
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".json")
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var f checkoutFixture
			if err := json.Unmarshal(raw, &f); err != nil {
				t.Fatal(err)
			}
			runCheckoutFixture(t, f)
		})
	}
}

func runCheckoutFixture(t *testing.T, f checkoutFixture) {
	s := newRecordingServer(t)
	for _, ex := range f.Exchanges {
		body, _ := json.Marshal(ex.Response.Body)
		s.queue(ex.Response.Status, string(body))
	}

	client := NewClient(s.URL)
	outcome, callErr := callCheckoutFixture(t, client, f)

	seen := s.all()
	if len(seen) != len(f.Exchanges) {
		t.Fatalf("requests: got %d, want %d", len(seen), len(f.Exchanges))
	}
	for i, req := range seen {
		want := f.Exchanges[i].Request
		got := fixtureRequest{
			Method: req.Method,
			Path:   strings.TrimPrefix(req.Path, "/wp-json/"),
			Query:  map[string]string{},
		}
		for k := range req.Query {
			got.Query[k] = req.Query.Get(k)
		}
		if req.Body != "" {
			if err := json.Unmarshal([]byte(req.Body), &got.Body); err != nil {
				t.Fatalf("request body %q: %v", req.Body, err)
			}
		}
		if want.Query == nil {
			want.Query = map[string]string{}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("request %d:\n got  %#v\n want %#v", i, got, want)
		}
	}

	if rawErr, ok := f.Expect["error"]; ok {
		var want struct {
			Code       string  `json:"code"`
			CheckoutID *string `json:"checkoutId"`
		}
		json.Unmarshal(rawErr, &want)
		var apiErr *CoCartError
		if !errors.As(callErr, &apiErr) || apiErr.ErrorCode != want.Code {
			t.Fatalf("error: got %v, want code %s", callErr, want.Code)
		}
		// The checkout-outcome-unknown error carries the checkout_id to ask GetStatus about.
		if want.CheckoutID != nil {
			var unknown *CheckoutOutcomeUnknownError
			if !errors.As(callErr, &unknown) || unknown.CheckoutID != *want.CheckoutID {
				t.Fatalf("error: got %#v, want a *CheckoutOutcomeUnknownError with CheckoutID %s", callErr, *want.CheckoutID)
			}
		}
		return
	}
	if callErr != nil {
		t.Fatal(callErr)
	}

	if rawResult, ok := f.Expect["result"]; ok {
		result, ok := outcome.(*CheckoutResult)
		if !ok {
			t.Fatalf("outcome is %T, want *CheckoutResult", outcome)
		}
		var want map[string]any
		json.Unmarshal(rawResult, &want)
		for key, value := range want {
			got := checkoutResultField(t, result, key)
			if !reflect.DeepEqual(normalizeJSON(got), normalizeJSON(value)) {
				t.Errorf("%s: got %#v, want %#v", key, got, value)
			}
		}
	}
	if rawValue, ok := f.Expect["value"]; ok {
		if status, isStatus := outcome.(*CheckoutStatus); isStatus {
			var want map[string]any
			json.Unmarshal(rawValue, &want)
			orNil := func(v any) any {
				if v == "" || v == 0 {
					return nil
				}
				return v
			}
			got := map[string]any{
				"checkoutId": status.CheckoutID, "code": status.Code, "message": status.Message, "orderId": orNil(status.OrderID),
				"orderKey": orNil(status.OrderKey), "orderNumber": orNil(status.OrderNumber), "orderStatus": orNil(status.OrderStatus),
			}
			if !reflect.DeepEqual(normalizeJSON(got), normalizeJSON(want)) {
				t.Errorf("value:\n got  %#v\n want %#v", got, want)
			}
			return
		}
		var want []map[string]any
		json.Unmarshal(rawValue, &want)
		cards, _ := outcome.([]SavedPaymentMethod)
		var got []map[string]any
		for _, c := range cards {
			var expires any = c.Expires
			if c.Expires == "" {
				expires = nil
			}
			got = append(got, map[string]any{
				"id": c.ID, "gatewayId": c.GatewayID, "label": c.Label, "type": c.Type,
				"expires": expires, "isDefault": c.IsDefault, "supportsSubscriptions": c.SupportsSubscriptions,
			})
		}
		if !reflect.DeepEqual(normalizeJSON(got), normalizeJSON(want)) {
			t.Errorf("value:\n got  %#v\n want %#v", got, want)
		}
	}
	if rawBody, ok := f.Expect["responseBody"]; ok {
		var resp *Response
		switch o := outcome.(type) {
		case *Response:
			resp = o
		case *CheckoutResult:
			resp = o.Response
		}
		var want any
		json.Unmarshal(rawBody, &want)
		var got any
		json.Unmarshal(resp.Body, &got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("response body:\n got  %#v\n want %#v", got, want)
		}
	}
}

func callCheckoutFixture(t *testing.T, client *Client, f checkoutFixture) (any, error) {
	ctx := context.Background()
	args := f.Call.Args
	str := func(i int) string { s, _ := args[i].(string); return s }
	num := func(i int) int { n, _ := args[i].(float64); return int(n) }
	obj := func(i int) map[string]any {
		if i >= len(args) {
			return map[string]any{}
		}
		m, _ := args[i].(map[string]any)
		return m
	}

	if f.Call.Target == "account" {
		if f.Call.Method != "updateSubscriptionPaymentMethod" {
			t.Fatalf("unknown account method %s", f.Call.Method)
		}
		return client.Account().UpdateSubscriptionPaymentMethod(ctx, num(0), obj(1))
	}

	co := client.Checkout()
	switch f.Call.Method {
	case "getCheckout":
		// The object argument maps onto Get's optional parameters.
		if len(args) > 0 {
			return co.Get(ctx, queryOf(obj(0)))
		}
		return co.Get(ctx)
	case "updateCheckout":
		return co.Update(ctx, obj(0))
	case "placeOrder":
		return co.PlaceOrder(ctx, obj(0))
	case "getCheckoutConfig":
		return co.GetConfig(ctx)
	case "getCheckoutStatus":
		return co.GetStatus(ctx, str(0))
	case "getPaymentMethods":
		return co.GetPaymentMethods(ctx)
	case "getSavedPaymentMethods":
		gateway := ""
		if len(args) > 0 {
			gateway = str(0)
		}
		return co.GetSavedPaymentMethods(ctx, gateway)
	case "searchAddresses":
		params := map[string]string{}
		for k, v := range obj(0) {
			params[k], _ = v.(string)
		}
		return co.SearchAddresses(ctx, params)
	case "getAddressDetails":
		a := obj(0)
		return co.GetAddressDetails(ctx, a["address_id"].(string), a["provider"].(string))
	case "getOrderReceived":
		if len(args) > 2 {
			return co.GetOrderReceived(ctx, num(0), str(1), queryOf(obj(2)))
		}
		return co.GetOrderReceived(ctx, num(0), str(1))
	case "payForOrder":
		return co.PayForOrder(ctx, num(0), str(1), obj(2))
	case "confirmReturn":
		options := obj(1)
		if len(options) == 0 {
			return co.ConfirmReturn(ctx, str(0))
		}
		u, _ := url.Parse(str(0))
		orderID, _ := options["orderId"].(float64)
		orderKey, _ := options["orderKey"].(string)
		return co.ConfirmReturnFor(ctx, int(orderID), orderKey, u.Query())
	}
	t.Fatalf("unknown checkout method %s", f.Call.Method)
	return nil, nil
}

// checkoutResultField reads a fixture's camelCase result key from a
// CheckoutResult. null in a fixture is a field the result leaves out: "", 0
// or a nil map here.
func checkoutResultField(t *testing.T, r *CheckoutResult, key string) any {
	orNil := func(v any) any {
		switch x := v.(type) {
		case string:
			if x == "" {
				return nil
			}
		case int:
			if x == 0 {
				return nil
			}
		case map[string]any:
			if x == nil {
				return nil
			}
		}
		return v
	}
	switch key {
	case "code":
		return orNil(string(r.Code))
	case "object":
		return orNil(r.Object)
	case "isComplete":
		return r.IsComplete()
	case "isAwaitingPayment":
		return r.IsAwaitingPayment()
	case "needsConfirmation":
		return r.NeedsConfirmation()
	case "needsRedirect":
		return r.NeedsRedirect()
	case "isPending":
		return r.IsPending()
	case "isFailed":
		return r.IsFailed()
	case "orderId":
		return orNil(r.OrderID)
	case "orderKey":
		return orNil(r.OrderKey)
	case "orderNumber":
		return orNil(r.OrderNumber)
	case "orderStatus":
		return orNil(r.OrderStatus)
	case "subscriptionId":
		return orNil(r.SubscriptionID)
	case "redirectUrl":
		return orNil(r.RedirectURL)
	case "clientSecret":
		return orNil(r.ClientSecret)
	case "intentType":
		return orNil(r.IntentType)
	case "paymentMethodType":
		return orNil(r.PaymentMethodType)
	case "message":
		return orNil(r.Message)
	case "errorCode":
		return orNil(r.ErrorCode)
	case "gatewayErrorCode":
		return orNil(r.GatewayErrorCode)
	case "gatewayMessage":
		return orNil(r.GatewayMessage)
	case "actionType":
		return orNil(r.ActionType)
	case "actionData":
		return orNil(r.ActionData)
	}
	t.Fatalf("unknown result key %s", key)
	return nil
}

// normalizeJSON round-trips a value through JSON, so ints and float64s compare equal.
func normalizeJSON(v any) any {
	raw, _ := json.Marshal(v)
	var out any
	json.Unmarshal(raw, &out)
	return out
}

// queryOf turns a fixture's JSON arguments into the query strings the SDK
// takes, as a caller would write them: a list comma-joined, a boolean "true"
// or "false", nil values left out.
func queryOf(params map[string]any) map[string]string {
	query := make(map[string]string, len(params))
	for k, v := range params {
		switch value := v.(type) {
		case nil:
			continue
		case string:
			query[k] = value
		case []string:
			query[k] = strings.Join(value, ",")
		case []any:
			parts := make([]string, len(value))
			for i, part := range value {
				parts[i] = fmt.Sprint(part)
			}
			query[k] = strings.Join(parts, ",")
		case bool:
			query[k] = strconv.FormatBool(value)
		case float64:
			query[k] = strconv.FormatFloat(value, 'f', -1, 64)
		default:
			query[k] = fmt.Sprint(value)
		}
	}
	return query
}
