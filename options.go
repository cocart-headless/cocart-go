package cocart

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithCartKey sets an existing guest cart key.
func WithCartKey(key string) Option {
	return func(c *Client) {
		c.cartKey = key
	}
}

// WithBasicAuth sets username and password for Basic authentication.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		c.auth = &authCredentials{username: username, password: password}
	}
}

// WithJWTToken sets a JWT access token.
func WithJWTToken(token string) Option {
	return func(c *Client) {
		c.jwtToken = token
	}
}

// WithJWTRefreshToken sets a JWT refresh token.
func WithJWTRefreshToken(token string) Option {
	return func(c *Client) {
		c.refreshToken = token
	}
}

// WithWooCommerceKeys sets WooCommerce consumer key and secret for admin operations.
func WithWooCommerceKeys(key, secret string) Option {
	return func(c *Client) {
		c.consumerKey = key
		c.consumerSecret = secret
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithRESTPrefix sets the WordPress REST API prefix (default: "wp-json").
func WithRESTPrefix(prefix string) Option {
	return func(c *Client) {
		c.restPrefix = trimSlashes(prefix)
	}
}

// WithNamespace sets the API namespace (default: "cocart"). Supports white-labelling.
func WithNamespace(ns string) Option {
	return func(c *Client) {
		c.namespace = trimSlashes(ns)
	}
}

// WithHeaders sets custom headers to send with every request.
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		for k, v := range headers {
			c.customHeaders[k] = v
		}
	}
}

// WithStorage sets the storage adapter for persisting cart keys and tokens.
func WithStorage(s Storage) Option {
	return func(c *Client) {
		c.storage = s
	}
}

// WithStorageKey sets the storage key name for the cart key (default: "cocart_cart_key").
func WithStorageKey(key string) Option {
	return func(c *Client) {
		c.storageKey = key
	}
}

// WithMaxRetries sets the maximum number of retries for transient failures.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

// WithDebug enables or disables debug logging.
func WithDebug(enabled bool) Option {
	return func(c *Client) {
		c.debug = enabled
	}
}

// WithAuthHeaderName sets a custom authorization header name (default: "Authorization").
func WithAuthHeaderName(name string) Option {
	return func(c *Client) {
		c.authHeaderName = name
	}
}

// WithResponseTransformer sets a function to transform every API response before returning.
func WithResponseTransformer(fn func(*Response) *Response) Option {
	return func(c *Client) {
		c.responseTransformer = fn
	}
}

// WithETag enables or disables ETag conditional requests (default: true).
func WithETag(enabled bool) Option {
	return func(c *Client) {
		c.etagEnabled = enabled
	}
}

// WithMainPlugin sets the CoCart main plugin variant ("basic" or "legacy").
func WithMainPlugin(plugin MainPlugin) Option {
	return func(c *Client) {
		c.mainPlugin = plugin
	}
}

// WithHTTPClient sets a custom http.Client for making requests.
// Useful for testing or custom transport configuration.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}
