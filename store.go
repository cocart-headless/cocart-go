package cocart

import "context"

// StoreEndpoint handles store information API operations.
type StoreEndpoint struct {
	endpoint
}

// Info returns store information.
func (e *StoreEndpoint) Info(ctx context.Context, params ...map[string]string) (*Response, error) {
	return e.doGet(ctx, "", params...)
}
