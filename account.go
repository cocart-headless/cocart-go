package cocart

import (
	"context"
	"fmt"
	"net/http"
)

const accountBase = "cocart/v2/my-account"

// AccountEndpoint handles all my-account API operations.
type AccountEndpoint struct {
	endpoint
}

func (e *AccountEndpoint) rawPath(sub string) string {
	if sub == "" {
		return accountBase
	}
	return accountBase + "/" + sub
}

func (e *AccountEndpoint) doRawGet(ctx context.Context, sub string, params map[string]string) (*Response, error) {
	resp, err := e.client.RequestRaw(ctx, http.MethodGet, e.rawPath(sub), params, nil)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

func (e *AccountEndpoint) doRawPost(ctx context.Context, sub string, data any) (*Response, error) {
	resp, err := e.client.RequestRaw(ctx, http.MethodPost, e.rawPath(sub), nil, data)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

// GetProfile returns the authenticated user's account profile.
func (e *AccountEndpoint) GetProfile(ctx context.Context) (*Response, error) {
	return e.doRawGet(ctx, "", nil)
}

// UpdateProfile updates the authenticated user's profile fields.
func (e *AccountEndpoint) UpdateProfile(ctx context.Context, data map[string]any) (*Response, error) {
	return e.doRawPost(ctx, "", data)
}

// ChangePassword changes the authenticated user's password.
// The three fields are remapped to the wire names expected by the API.
func (e *AccountEndpoint) ChangePassword(ctx context.Context, current, password, confirm string) (*Response, error) {
	return e.doRawPost(ctx, "change-password", map[string]any{
		"password_current": current,
		"password_1":       password,
		"password_2":       confirm,
	})
}

// GetOrders returns the user's order history. Pass nil for no filters.
func (e *AccountEndpoint) GetOrders(ctx context.Context, params map[string]string) (*Response, error) {
	return e.doRawGet(ctx, "orders", params)
}

// GetOrder returns a single order by ID.
func (e *AccountEndpoint) GetOrder(ctx context.Context, id int) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d", id), nil)
}

// GetGuestOrder returns a single order by ID for a guest, identified by billing email.
func (e *AccountEndpoint) GetGuestOrder(ctx context.Context, id int, email string) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d", id), map[string]string{"email": email})
}

// GetOrderDownloads returns downloadable files for a specific order.
func (e *AccountEndpoint) GetOrderDownloads(ctx context.Context, id int) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d/downloads", id), nil)
}

// GetGuestOrderDownloads returns downloadable files for a specific guest order.
func (e *AccountEndpoint) GetGuestOrderDownloads(ctx context.Context, id int, email string) (*Response, error) {
	return e.doRawGet(ctx, fmt.Sprintf("orders/%d/downloads", id), map[string]string{"email": email})
}

// GetDownloads returns all downloadable files available to the authenticated user.
func (e *AccountEndpoint) GetDownloads(ctx context.Context) (*Response, error) {
	return e.doRawGet(ctx, "downloads", nil)
}

// GetReviews returns the authenticated user's product reviews.
func (e *AccountEndpoint) GetReviews(ctx context.Context) (*Response, error) {
	return e.doRawGet(ctx, "reviews", nil)
}
