package cocart

import (
	"context"
	"fmt"
	"net/http"
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

// AddItem adds an item to the cart.
func (e *CartEndpoint) AddItem(ctx context.Context, productID int, quantity int, options ...map[string]any) (*Response, error) {
	if err := ValidateProductID(productID); err != nil {
		return nil, err
	}
	if err := ValidateQuantity(quantity); err != nil {
		return nil, err
	}
	data := map[string]any{
		"id":       fmt.Sprintf("%d", productID),
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
	if err := e.client.RequiresBasic("cart()->create"); err != nil {
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

// ApplyCoupon applies a coupon to the cart.
func (e *CartEndpoint) ApplyCoupon(ctx context.Context, couponCode string) (*Response, error) {
	return e.doPost(ctx, "apply-coupon", map[string]any{"coupon": couponCode})
}

// RemoveCoupon removes a coupon from the cart.
func (e *CartEndpoint) RemoveCoupon(ctx context.Context, couponCode string) (*Response, error) {
	return e.doDelete(ctx, "coupons/"+couponCode)
}

// GetCoupons returns applied coupons.
func (e *CartEndpoint) GetCoupons(ctx context.Context) (*Response, error) {
	return e.doGet(ctx, "", map[string]string{"_fields": "coupons"})
}

// CheckCoupons validates applied coupons.
func (e *CartEndpoint) CheckCoupons(ctx context.Context) (*Response, error) {
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

// GetShippingMethods returns available shipping methods.
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

// AddFee adds a fee to the cart.
func (e *CartEndpoint) AddFee(ctx context.Context, name string, amount float64, taxable bool) (*Response, error) {
	return e.doPost(ctx, "add-fee", map[string]any{
		"name":    name,
		"amount":  amount,
		"taxable": taxable,
	})
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
func (e *CartEndpoint) Add(ctx context.Context, productID int, quantity int) (*Response, error) {
	return e.AddItem(ctx, productID, quantity)
}

// AddVariation adds a variable product to the cart.
func (e *CartEndpoint) AddVariation(ctx context.Context, variationID int, quantity int, attributes map[string]string) (*Response, error) {
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
