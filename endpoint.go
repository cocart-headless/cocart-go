package cocart

import (
	"context"
	"errors"
	"strings"
)

// endpoint is the base helper for all endpoint implementations.
type endpoint struct {
	client   *Client
	basePath string
}

// buildPath constructs the full endpoint path.
func (e *endpoint) buildPath(path string) string {
	if path == "" {
		return e.basePath
	}
	return strings.TrimRight(e.basePath, "/") + "/" + strings.TrimLeft(path, "/")
}

// doGet makes a GET request with the endpoint prefix.
func (e *endpoint) doGet(ctx context.Context, path string, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	resp, err := e.client.Get(ctx, e.buildPath(path), p)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

// doPost makes a POST request with the endpoint prefix.
func (e *endpoint) doPost(ctx context.Context, path string, data any, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	resp, err := e.client.Post(ctx, e.buildPath(path), data, p)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

// doPut makes a PUT request with the endpoint prefix.
func (e *endpoint) doPut(ctx context.Context, path string, data any, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	resp, err := e.client.Put(ctx, e.buildPath(path), data, p)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

// doDelete makes a DELETE request with the endpoint prefix.
func (e *endpoint) doDelete(ctx context.Context, path string, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	resp, err := e.client.Delete(ctx, e.buildPath(path), p)
	if err != nil {
		return resp, e.handleNoRoute(err)
	}
	return resp, nil
}

// handleNoRoute wraps rest_no_route errors with a friendly message.
func (e *endpoint) handleNoRoute(err error) error {
	var cocartErr *CoCartError
	if errors.As(err, &cocartErr) && cocartErr.ErrorCode == "rest_no_route" {
		return NewCoCartError(
			"This method is only available with another CoCart plugin. Please ask support for assistance!",
			404,
			"cocart_plugin_required",
		)
	}
	return err
}
