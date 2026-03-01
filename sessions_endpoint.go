package cocart

import (
	"context"
	"fmt"
)

// SessionsEndpoint handles cart session management for administrators.
// Requires WooCommerce REST API credentials (consumer_key/consumer_secret).
type SessionsEndpoint struct {
	endpoint
}

// All returns all cart sessions.
func (e *SessionsEndpoint) All(ctx context.Context, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.doGet(ctx, "", p)
}

// Find returns a specific cart session.
func (e *SessionsEndpoint) Find(ctx context.Context, sessionKey string, params ...*PaginationParams) (*Response, error) {
	var p map[string]string
	if len(params) > 0 && params[0] != nil {
		p = structToParams(params[0])
	}
	return e.client.Get(ctx, "session/"+sessionKey, p)
}

// Destroy deletes a cart session.
func (e *SessionsEndpoint) Destroy(ctx context.Context, sessionKey string) (*Response, error) {
	return e.client.Delete(ctx, "session/"+sessionKey)
}

// GetItems returns items for a specific session.
func (e *SessionsEndpoint) GetItems(ctx context.Context, sessionKey string) (*Response, error) {
	return e.client.Get(ctx, "session/"+sessionKey+"/items")
}

// BySession returns a session by customer ID.
func (e *SessionsEndpoint) BySession(ctx context.Context, customerID int) (*Response, error) {
	return e.client.Get(ctx, fmt.Sprintf("session/%d", customerID))
}

// DestroySession deletes a session by customer ID.
func (e *SessionsEndpoint) DestroySession(ctx context.Context, customerID int) (*Response, error) {
	return e.client.Delete(ctx, fmt.Sprintf("session/%d", customerID))
}
