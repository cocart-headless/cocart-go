package cocart

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// CartEndpoint handles all cart-related API operations.
type CartEndpoint struct {
	endpoint
}

// Get returns the cart contents.
func (e *CartEndpoint) Get(ctx context.Context, params ...*CartGetParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.client.Get(ctx, e.basePath, p)
}

// GetFiltered returns specific fields from the cart.
func (e *CartEndpoint) GetFiltered(ctx context.Context, fields ...string) (*Response, error) {
	p := map[string]string{"_fields": joinFields(fields)}
	return e.client.Get(ctx, e.basePath, p)
}

// AddItem adds an item to the cart. productID is a product ID, variation ID
// or SKU.
//
// options holds the request's other fields: "variation" (a variable
// product's attributes), "item_data" (custom data), "email", "price" and
// "return_item", and the choices for CoCart Plus's supported plugins:
// "convert_to_sub" (a subscription plan's value, "0" for one-time),
// "recipient_email" (a gift's recipient) and "item_data": {"warranty": key}
// (a warranty option). Other plugins' fields pass through as given.
func (e *CartEndpoint) AddItem(ctx context.Context, productID any, quantity int, options ...map[string]any) (*Response, error) {
	if err := ValidateProductID(productID); err != nil {
		return nil, err
	}
	if err := ValidateQuantity(quantity); err != nil {
		return nil, err
	}
	data := map[string]any{
		"id":       fmt.Sprintf("%v", productID),
		"quantity": fmt.Sprintf("%d", quantity),
	}
	if len(options) > 0 {
		for k, v := range options[0] {
			data[k] = v
		}
	}
	return e.doPost(ctx, "add-item", data)
}

// AddItems adds multiple children of a WooCommerce Grouped Product to the
// cart in a single request, via the dedicated `cart/add-items` endpoint.
//
// This is NOT a generic "add several unrelated products" call — the server
// requires a single grouped product ID plus a map of that group's child
// product IDs to quantities. For adding unrelated products in one request,
// use [Client.Batch] instead.
//
// groupedProductID is the parent grouped product's ID (int or string).
// items maps child product ID => quantity.
func (e *CartEndpoint) AddItems(ctx context.Context, groupedProductID any, items map[string]int) (*Response, error) {
	if len(items) == 0 {
		return nil, NewValidationError("AddItems() requires at least one item.", 0, "cocart_missing_items")
	}

	quantity := make(map[string]string, len(items))
	for childID, qty := range items {
		quantity[childID] = fmt.Sprintf("%d", qty)
	}

	return e.doPost(ctx, "add-items", map[string]any{
		"id":       fmt.Sprintf("%v", groupedProductID),
		"quantity": quantity,
	})
}

// UpdateItem updates an item's quantity in the cart.
func (e *CartEndpoint) UpdateItem(ctx context.Context, itemKey string, quantity int, options ...map[string]any) (*Response, error) {
	if err := ValidateQuantity(quantity); err != nil {
		return nil, err
	}
	data := map[string]any{
		"quantity": fmt.Sprintf("%d", quantity),
	}
	if len(options) > 0 {
		for k, v := range options[0] {
			data[k] = v
		}
	}
	return e.doPost(ctx, "item/"+itemKey, data)
}

// UpdateItems updates multiple items' quantities, one request per item,
// sequentially, and returns the response from the last update (there is no
// real bulk-update endpoint server-side). For a true single round trip, use
// [CartEndpoint.BatchUpdateItems] (requires CoCart Plus).
func (e *CartEndpoint) UpdateItems(ctx context.Context, items map[string]int) (*Response, error) {
	if len(items) == 0 {
		return nil, NewValidationError("UpdateItems() requires at least one item.", 0, "cocart_missing_items")
	}

	var resp *Response
	var err error
	for itemKey, qty := range items {
		resp, err = e.UpdateItem(ctx, itemKey, qty)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// BatchUpdateItems updates multiple items' quantities in a single request via
// the `cocart/batch` endpoint (requires CoCart Plus). Unlike
// [CartEndpoint.UpdateItems], this is a true single round trip instead of one
// sequential request per item.
func (e *CartEndpoint) BatchUpdateItems(ctx context.Context, items map[string]int) (*Response, error) {
	if len(items) == 0 {
		return nil, NewValidationError("BatchUpdateItems() requires at least one item.", 0, "cocart_missing_items")
	}

	requests := make([]BatchRequestItem, 0, len(items))
	for itemKey, qty := range items {
		requests = append(requests, BatchRequestItem{
			Method: http.MethodPost,
			Path:   fmt.Sprintf("/%s/%s/cart/item/%s", e.client.GetNamespace(), APIVersion, itemKey),
			Body:   map[string]any{"quantity": fmt.Sprintf("%d", qty)},
		})
	}

	return e.client.Batch(ctx, requests)
}

// RemoveItem removes an item from the cart.
func (e *CartEndpoint) RemoveItem(ctx context.Context, itemKey string) (*Response, error) {
	return e.doDelete(ctx, "item/"+itemKey)
}

// RemoveItems removes multiple items from the cart, one request per item,
// sequentially, and returns the response from the last removal (there is no
// real bulk-remove endpoint server-side). For a true single round trip, use
// [CartEndpoint.BatchRemoveItems] (requires CoCart Plus).
func (e *CartEndpoint) RemoveItems(ctx context.Context, itemKeys []string) (*Response, error) {
	if len(itemKeys) == 0 {
		return nil, NewValidationError("RemoveItems() requires at least one item key.", 0, "cocart_missing_items")
	}

	var resp *Response
	var err error
	for _, itemKey := range itemKeys {
		resp, err = e.RemoveItem(ctx, itemKey)
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}

// BatchRemoveItems removes multiple items in a single request via the
// `cocart/batch` endpoint (requires CoCart Plus). Unlike
// [CartEndpoint.RemoveItems], this is a true single round trip instead of one
// sequential request per item.
func (e *CartEndpoint) BatchRemoveItems(ctx context.Context, itemKeys []string) (*Response, error) {
	if len(itemKeys) == 0 {
		return nil, NewValidationError("BatchRemoveItems() requires at least one item key.", 0, "cocart_missing_items")
	}

	requests := make([]BatchRequestItem, 0, len(itemKeys))
	for _, itemKey := range itemKeys {
		requests = append(requests, BatchRequestItem{
			Method: http.MethodDelete,
			Path:   fmt.Sprintf("/%s/%s/cart/item/%s", e.client.GetNamespace(), APIVersion, itemKey),
		})
	}

	return e.client.Batch(ctx, requests)
}

// RestoreItem restores a removed item to the cart.
func (e *CartEndpoint) RestoreItem(ctx context.Context, itemKey string) (*Response, error) {
	return e.doPut(ctx, "item/"+itemKey, nil)
}

// GetRemovedItems returns items that have been removed and can be restored.
func (e *CartEndpoint) GetRemovedItems(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "removed_items"})
}

// Clear removes all items from the cart.
func (e *CartEndpoint) Clear(ctx context.Context) (*Response, error) {
	return e.doPost(ctx, "clear", nil)
}

// Empty is an alias for Clear.
func (e *CartEndpoint) Empty(ctx context.Context) (*Response, error) {
	return e.Clear(ctx)
}

// Calculate recalculates cart totals.
func (e *CartEndpoint) Calculate(ctx context.Context, params ...map[string]any) (*Response, error) {
	var data any
	if len(params) > 0 {
		data = params[0]
	}
	return e.doPost(ctx, "calculate", data)
}

// GetTotals returns cart totals.
func (e *CartEndpoint) GetTotals(ctx context.Context, html ...bool) (*Response, error) {
	var p map[string]string
	if len(html) > 0 && html[0] {
		p = map[string]string{"html": "true"}
	}
	return e.client.Get(ctx, "cart/totals", p)
}

// GetItemCount returns the count of items in the cart.
func (e *CartEndpoint) GetItemCount(ctx context.Context) (*Response, error) {
	return e.client.Get(ctx, "cart/items/count")
}

// Create creates a new guest cart session without adding items.
// Requires CoCart Starter.
func (e *CartEndpoint) Create(ctx context.Context) (*Response, error) {
	if err := e.client.RequiresStarter("cart()->create"); err != nil {
		return nil, err
	}
	return e.doPost(ctx, "", nil)
}

// GetItems returns all items in the cart.
func (e *CartEndpoint) GetItems(ctx context.Context, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, "items", params...)
}

// GetItem returns a specific item from the cart by item key.
func (e *CartEndpoint) GetItem(ctx context.Context, itemKey string, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, "item/"+itemKey, params...)
}

// Update updates the entire cart.
func (e *CartEndpoint) Update(ctx context.Context, data map[string]any) (*Response, error) {
	return e.doPost(ctx, "update", data)
}

// ChangePlan switches a cart item to another subscription plan, or to
// one-time with "0" (CoCart Plus).
//
// The options are in the item's extensions["subscription_schemes"]; pass
// one's value. It posts the update-item-schema callback on POST /cart/update.
func (e *CartEndpoint) ChangePlan(ctx context.Context, itemKey, value string) (*Response, error) {
	if itemKey == "" {
		return nil, NewValidationError("ChangePlan() requires an item key.", 0, "cocart_missing_item_key")
	}
	return e.Update(ctx, map[string]any{
		"namespace":      "update-item-schema",
		"item_key":       itemKey,
		"convert_to_sub": value,
	})
}

// GetMini returns a lighter cart response for a mini cart (CoCart Plus).
func (e *CartEndpoint) GetMini(ctx context.Context, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, "mini", params...)
}

// RemoveAllItems removes every item but keeps the cart's coupons, fees and
// customer details (CoCart Plus, DELETE /cart/items). Clear empties the whole
// cart.
func (e *CartEndpoint) RemoveAllItems(ctx context.Context) (*Response, error) {
	return e.doDelete(ctx, "items")
}

// ApplyCoupon applies a coupon to the cart (CoCart Plus). Pass the
// customer's email for coupons restricted to certain emails when the cart
// doesn't know it yet.
func (e *CartEndpoint) ApplyCoupon(ctx context.Context, couponCode string, email ...string) (*Response, error) {
	// CoCart Plus's apply-coupon route takes `code` (`coupon` is rejected as a missing parameter).
	data := map[string]any{"code": couponCode}
	if len(email) > 0 && email[0] != "" {
		data["email"] = email[0]
	}
	return e.doPost(ctx, "apply-coupon", data)
}

// RemoveCoupon removes a coupon from the cart.
//
// The code goes in the URL path, so it's encoded: codes can contain spaces,
// %, #, ? and other characters. A code containing / can't be removed by
// code (CoCart Plus matches any path segment without /); RemoveCoupons
// removes all of them instead.
func (e *CartEndpoint) RemoveCoupon(ctx context.Context, couponCode string) (*Response, error) {
	return e.doDelete(ctx, "coupons/"+url.PathEscape(couponCode))
}

// RemoveCoupons removes every coupon from the cart (CoCart Plus).
func (e *CartEndpoint) RemoveCoupons(ctx context.Context) (*Response, error) {
	return e.doDelete(ctx, "coupons")
}

// GetCoupons returns applied coupons.
func (e *CartEndpoint) GetCoupons(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "coupons"})
}

// CheckCoupons validates applied coupons (CoCart Plus). Pass the customer's
// email to check coupons restricted to certain emails.
func (e *CartEndpoint) CheckCoupons(ctx context.Context, email ...string) (*Response, error) {
	if len(email) > 0 && email[0] != "" {
		return e.doGet(ctx, "coupons/validate", map[string]string{"email": email[0]})
	}
	return e.doGet(ctx, "coupons/validate")
}

// UpdateCustomer updates the customer's billing address on the cart, and
// mirrors it into the shipping address unless a distinct shipping address is
// provided.
//
// Billing fields are sent unprefixed (first_name, address_1, ...) and
// shipping fields are sent s_-prefixed (s_first_name, s_address_1, ...),
// matching the CoCart plugin's actual update-customer callback, which always
// validates any address field the destination country marks required -
// independent of whether ship_to_different_address is set. If shipping is
// omitted or empty, billing is mirrored into the s_ fields so that check
// passes, same as leaving "ship to a different address" unchecked at a
// normal WooCommerce checkout.
func (e *CartEndpoint) UpdateCustomer(ctx context.Context, billing, shipping map[string]string) (*Response, error) {
	data := map[string]any{"namespace": "update-customer"}

	for k, v := range billing {
		data[k] = v
	}

	shipTo := shipping
	if len(shipTo) == 0 {
		shipTo = billing
	}
	for k, v := range shipTo {
		data["s_"+k] = v
	}

	if len(shipping) > 0 {
		data["ship_to_different_address"] = true
	}

	return e.doPost(ctx, "update", data)
}

// GetCustomer returns customer details from the cart.
func (e *CartEndpoint) GetCustomer(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "customer"})
}

// GetShippingMethods returns the cart's shipping packages and their rates.
// Read them with [Response.GetShippingMethods] (a list of packages) or
// [Response.GetShipping] (the whole field).
func (e *CartEndpoint) GetShippingMethods(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "shipping"})
}

// SetShippingMethod selects a shipping rate for a package (CoCart Plus).
//
// rateID is the chosen rate's key, e.g. "flat_rate:2" (see a shipping
// package's Rates map). packageID restricts the selection to one package;
// omit it (or pass "") to apply the rate to every package.
func (e *CartEndpoint) SetShippingMethod(ctx context.Context, rateID string, packageID ...string) (*Response, error) {
	data := map[string]any{"rate_id": rateID}
	if len(packageID) > 0 && packageID[0] != "" {
		data["package_id"] = packageID[0]
	}
	return e.doPost(ctx, "set-shipping-method", data)
}

// CalculateShipping calculates shipping for the cart.
//
// Deprecated: there is no address-taking shipping calculation endpoint in
// the CoCart REST API - POST /cart/calculate/shipping (what this method used
// to call) does not exist. To calculate shipping, call UpdateCustomer with
// the destination address first (the server recalculates totals as part of
// that request); this method now just delegates to Calculate, ignoring
// address. Prefer calling Calculate directly.
func (e *CartEndpoint) CalculateShipping(ctx context.Context, address map[string]string) (*Response, error) {
	_ = address
	return e.Calculate(ctx)
}

// GetFees returns cart fees.
func (e *CartEndpoint) GetFees(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "fees"})
}

// AddFee adds a fee to the cart (CoCart Plus). Pass a tax class to use when
// the fee is taxable.
func (e *CartEndpoint) AddFee(ctx context.Context, name string, amount float64, taxable bool, taxClass ...string) (*Response, error) {
	// CoCart Plus takes the amount as a string (a number is rejected as rest_invalid_param).
	data := map[string]any{
		"name":    name,
		"amount":  strconv.FormatFloat(amount, 'f', -1, 64),
		"taxable": taxable,
	}
	if len(taxClass) > 0 && taxClass[0] != "" {
		data["tax_class"] = taxClass[0]
	}
	return e.doPost(ctx, "add-fee", data)
}

// RemoveFees removes all fees from the cart.
func (e *CartEndpoint) RemoveFees(ctx context.Context) (*Response, error) {
	return e.doPost(ctx, "remove-fees", nil)
}

// GetCrossSells returns cross-sell product recommendations.
func (e *CartEndpoint) GetCrossSells(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "cross_sells"})
}

// Add is a shorthand for adding a simple product to the cart.
func (e *CartEndpoint) Add(ctx context.Context, productID any, quantity int) (*Response, error) {
	return e.AddItem(ctx, productID, quantity)
}

// AddVariation adds a variable product to the cart. variationID accepts a
// numeric variation ID or a SKU — see AddItem for why.
func (e *CartEndpoint) AddVariation(ctx context.Context, variationID any, quantity int, attributes map[string]string) (*Response, error) {
	opts := map[string]any{"variation": attributes}
	return e.AddItem(ctx, variationID, quantity, opts)
}

// joinFields joins field names with commas.
func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ","
		}
		result += f
	}
	return result
}
