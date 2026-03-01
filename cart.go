package cocart

import (
	"context"
	"fmt"
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

// AddItems adds multiple items to the cart in a single request.
func (e *CartEndpoint) AddItems(ctx context.Context, items []CartItemData) (*Response, error) {
	formatted := make([]map[string]any, len(items))
	for i, item := range items {
		m := map[string]any{
			"id":       item.ID,
			"quantity": item.Quantity,
		}
		if len(item.Variation) > 0 {
			m["variation"] = item.Variation
		}
		if len(item.ItemData) > 0 {
			m["item_data"] = item.ItemData
		}
		formatted[i] = m
	}
	return e.doPost(ctx, "add-items", map[string]any{"items": formatted})
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

// UpdateItems updates multiple items in a single request.
func (e *CartEndpoint) UpdateItems(ctx context.Context, items map[string]int) (*Response, error) {
	formatted := make([]map[string]any, 0, len(items))
	for key, qty := range items {
		formatted = append(formatted, map[string]any{
			"item_key": key,
			"quantity": fmt.Sprintf("%d", qty),
		})
	}
	return e.doPost(ctx, "update", map[string]any{"items": formatted})
}

// RemoveItem removes an item from the cart.
func (e *CartEndpoint) RemoveItem(ctx context.Context, itemKey string) (*Response, error) {
	return e.doDelete(ctx, "item/"+itemKey)
}

// RemoveItems removes multiple items from the cart.
func (e *CartEndpoint) RemoveItems(ctx context.Context, itemKeys []string) (*Response, error) {
	items := make([]map[string]any, len(itemKeys))
	for i, key := range itemKeys {
		items[i] = map[string]any{
			"item_key": key,
			"quantity": "0",
		}
	}
	return e.doPost(ctx, "update", map[string]any{"items": items})
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
// Requires CoCart Basic.
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

// UpdateCustomer updates customer billing and shipping details.
func (e *CartEndpoint) UpdateCustomer(ctx context.Context, billing, shipping map[string]string) (*Response, error) {
	data := make(map[string]any)
	for k, v := range billing {
		data["billing_"+k] = v
	}
	for k, v := range shipping {
		data["shipping_"+k] = v
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

// SetShippingMethod sets the shipping method for the cart.
func (e *CartEndpoint) SetShippingMethod(ctx context.Context, methodKey string) (*Response, error) {
	return e.doPost(ctx, "set-shipping-method", map[string]any{"method_key": methodKey})
}

// CalculateShipping calculates shipping for the cart.
func (e *CartEndpoint) CalculateShipping(ctx context.Context, address map[string]string) (*Response, error) {
	data := make(map[string]any)
	for k, v := range address {
		data[k] = v
	}
	return e.doPost(ctx, "calculate/shipping", data)
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
