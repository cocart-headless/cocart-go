package cocart

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Version is the SDK version.
	Version = "1.0.0"
	// APIVersion is the CoCart API version.
	APIVersion = "v2"
)

// Client is the main CoCart API client.
type Client struct {
	storeURL            string
	restPrefix          string
	namespace           string
	storageKey          string
	cartKey             string
	auth                *authCredentials
	jwtToken            string
	refreshToken        string
	consumerKey         string
	consumerSecret      string
	maxRetries          int
	timeout             time.Duration
	customHeaders       map[string]string
	storage             Storage
	debug               bool
	authHeaderName      string
	responseTransformer func(*Response) *Response
	etagEnabled         bool
	etagCache           map[string]etagCacheEntry
	mainPlugin          MainPlugin
	httpClient          *http.Client
	lastResponse        *Response
	emitter             *eventEmitter

	// inFlightGets de-duplicates concurrent identical GET requests (same
	// built URL) so simultaneous callers share one network round trip.
	inFlightGets map[string]*inFlightGet
	inFlightMu   sync.Mutex

	// Lazy-loaded endpoint instances
	jwtManager       *JWTManager
	sessionManager   *SessionManager
	accountEndpoint  *AccountEndpoint
	cartEndpoint     *CartEndpoint
	productsEndpoint *ProductsEndpoint
	storeEndpoint    *StoreEndpoint
	sessionsEndpoint *SessionsEndpoint

	mu sync.RWMutex
}

// etagCacheEntry caches the ETag together with the body/headers of the fresh
// (non-304) GET response that produced it, so a later 304 can be resolved
// back to the actual cached data instead of an empty body.
type etagCacheEntry struct {
	etag    string
	body    []byte
	headers http.Header
}

// inFlightGet tracks a GET request in progress so concurrent identical GETs
// can share its result instead of firing one request each.
type inFlightGet struct {
	done chan struct{}
	resp *Response
	err  error
}

// NewClient creates a new CoCart client.
//
// Unless [WithCartKey] gives one, the client reads the cart key its storage
// adapter kept from an earlier run (see [WithStorage]), so the same cart
// continues on its own. A storage holds one cart key: don't share one between
// customers.
func NewClient(storeURL string, opts ...Option) *Client {
	c := newClient(storeURL, opts...)
	if c.cartKey == "" {
		// Storage unavailable: start without a cart key.
		_ = c.RestoreSession()
	}
	return c
}

func newClient(storeURL string, opts ...Option) *Client {
	c := &Client{
		storeURL:       strings.TrimRight(storeURL, "/"),
		restPrefix:     "wp-json",
		namespace:      "cocart",
		storageKey:     "cocart_cart_key",
		timeout:        30 * time.Second,
		customHeaders:  make(map[string]string),
		storage:        NewMemoryStorage(),
		authHeaderName: "Authorization",
		etagEnabled:    true,
		etagCache:      make(map[string]etagCacheEntry),
		mainPlugin:     MainPluginStarter,
		emitter:        newEventEmitter(),
		inFlightGets:   make(map[string]*inFlightGet),
	}

	c.emitter.logger = c.logEvent

	for _, opt := range opts {
		opt(c)
	}

	// Keep a client passed with WithHTTPClient.
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.timeout,
		}
	}

	return c
}

// FromRequest creates a new Client from an incoming HTTP request,
// extracting the cart key from the X-Cart-Key header if present.
// This is useful for building Go middleware that proxies CoCart requests.
//
// The cart key comes from the request only: unlike [NewClient], it doesn't
// read the storage adapter, which a server may share between customers.
func FromRequest(storeURL string, r *http.Request, opts ...Option) *Client {
	c := newClient(storeURL, opts...)
	if cartKey := r.Header.Get("X-Cart-Key"); cartKey != "" {
		c.SetCartKey(cartKey)
	}
	return c
}

// --- Cart key ---

// SetCartKey sets the cart key.
func (c *Client) SetCartKey(key string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cartKey = key
	return c
}

// GetCartKey returns the current cart key.
func (c *Client) GetCartKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cartKey
}

// --- Authentication ---

// SetAuth sets username and password for Basic authentication.
func (c *Client) SetAuth(username, password string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.auth = &authCredentials{username: username, password: password}
	c.jwtToken = ""
	return c
}

// SetJWTToken sets a JWT access token and clears Basic auth.
func (c *Client) SetJWTToken(token string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jwtToken = token
	c.auth = nil
	return c
}

// GetJWTToken returns the current JWT token.
func (c *Client) GetJWTToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtToken
}

// SetRefreshToken sets a JWT refresh token.
func (c *Client) SetRefreshToken(token string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshToken = token
	return c
}

// GetRefreshToken returns the current JWT refresh token.
func (c *Client) GetRefreshToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshToken
}

// HasJWTToken returns true if a JWT token is set.
func (c *Client) HasJWTToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jwtToken != ""
}

// ClearJWTToken clears the JWT and refresh tokens.
func (c *Client) ClearJWTToken() *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jwtToken = ""
	c.refreshToken = ""
	return c
}

// SetWooCommerceCredentials sets WooCommerce consumer key and secret.
func (c *Client) SetWooCommerceCredentials(key, secret string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consumerKey = key
	c.consumerSecret = secret
	return c
}

// --- Configuration ---

// SetTimeout sets the HTTP request timeout.
func (c *Client) SetTimeout(d time.Duration) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = d
	c.httpClient.Timeout = d
	return c
}

// SetMaxRetries sets the maximum retry count for transient failures: a 429
// for any request, a 503 or timeout for a GET only.
func (c *Client) SetMaxRetries(n int) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n < 0 {
		n = 0
	}
	c.maxRetries = n
	return c
}

// SetRESTPrefix sets the WordPress REST API prefix.
func (c *Client) SetRESTPrefix(prefix string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.restPrefix = trimSlashes(prefix)
	return c
}

// GetRESTPrefix returns the current REST prefix.
func (c *Client) GetRESTPrefix() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.restPrefix
}

// SetNamespace sets the API namespace.
func (c *Client) SetNamespace(ns string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.namespace = trimSlashes(ns)
	return c
}

// GetNamespace returns the current namespace.
func (c *Client) GetNamespace() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.namespace
}

// AddHeader adds a custom header to all requests.
func (c *Client) AddHeader(name, value string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.customHeaders[name] = value
	return c
}

// SetStorage sets the storage adapter.
func (c *Client) SetStorage(s Storage) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storage = s
	return c
}

// GetStorage returns the current storage adapter.
func (c *Client) GetStorage() Storage {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.storage
}

// SetDebug enables or disables debug logging.
func (c *Client) SetDebug(enabled bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.debug = enabled
	return c
}

// SetAuthHeaderName sets a custom authorization header name.
func (c *Client) SetAuthHeaderName(name string) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authHeaderName = name
	return c
}

// SetETag enables or disables ETag conditional requests.
func (c *Client) SetETag(enabled bool) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.etagEnabled = enabled
	return c
}

// ClearETagCache clears the ETag cache.
func (c *Client) ClearETagCache() *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.etagCache = make(map[string]etagCacheEntry)
	return c
}

// GetMainPlugin returns the configured main plugin variant.
func (c *Client) GetMainPlugin() MainPlugin {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mainPlugin
}

// SetMainPlugin sets the CoCart main plugin variant.
func (c *Client) SetMainPlugin(plugin MainPlugin) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mainPlugin = plugin
	return c
}

// SetResponseTransformer sets or clears the response transformer.
func (c *Client) SetResponseTransformer(fn func(*Response) *Response) *Client {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseTransformer = fn
	return c
}

// GetStoreURL returns the store URL.
func (c *Client) GetStoreURL() string {
	return c.storeURL
}

// GetLastResponse returns the most recent API response.
func (c *Client) GetLastResponse() *Response {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastResponse
}

// RequiresStarter throws a VersionError if the SDK is configured for the legacy plugin.
func (c *Client) RequiresStarter(method string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.mainPlugin == MainPluginLegacy {
		return NewVersionError(method)
	}
	return nil
}

// --- Auth convenience ---

// JWT returns the JWT manager instance.
func (c *Client) JWT() *JWTManager {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.jwtManager == nil {
		c.jwtManager = NewJWTManager(c, c.storage, WithAutoRefresh(true))
	}
	return c.jwtManager
}

// Session returns the SessionManager instance (lazily created).
// It uses the client's configured storage adapter.
func (c *Client) Session() *SessionManager {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessionManager == nil {
		c.sessionManager = NewSessionManager(c, c.storage)
	}
	return c.sessionManager
}

// Login authenticates with username and password via JWT.
func (c *Client) Login(ctx context.Context, username, password string) (*Response, error) {
	return c.JWT().Login(ctx, username, password)
}

// Logout clears the JWT session.
func (c *Client) Logout(ctx context.Context) error {
	_, _ = c.Post(ctx, "logout", nil)
	return c.JWT().ClearTokens()
}

// IsAuthenticated returns true if the client has credentials set.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.auth != nil || c.jwtToken != ""
}

// IsGuest returns true if the client has no credentials.
func (c *Client) IsGuest() bool {
	return !c.IsAuthenticated()
}

// ClearSession clears all authentication and cart state.
func (c *Client) ClearSession() error {
	c.mu.Lock()
	c.auth = nil
	c.jwtToken = ""
	c.refreshToken = ""
	c.mu.Unlock()
	return c.forgetCartKey()
}

// forgetCartKey clears the cart key, in storage too.
func (c *Client) forgetCartKey() error {
	c.mu.Lock()
	c.cartKey = ""
	storage := c.storage
	storageKey := c.storageKey
	c.mu.Unlock()
	return storage.Delete(storageKey)
}

// RestoreSession restores the cart key from storage. [NewClient] already
// does this, so you only need it after swapping the storage with
// [Client.SetStorage].
func (c *Client) RestoreSession() error {
	c.mu.RLock()
	storage := c.storage
	storageKey := c.storageKey
	c.mu.RUnlock()

	stored, err := storage.Get(storageKey)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return nil
		}
		return err
	}
	c.mu.Lock()
	if c.cartKey == "" {
		c.cartKey = stored
	}
	c.mu.Unlock()
	return nil
}

// TransferCartToCustomer transfers a guest cart to an authenticated user.
func (c *Client) TransferCartToCustomer(ctx context.Context, username, password string) (*Response, error) {
	guestCartKey := c.GetCartKey()
	c.SetAuth(username, password)

	if guestCartKey != "" {
		return c.Cart().Get(ctx, &CartGetParams{CartKey: guestCartKey})
	}
	return c.Cart().Get(ctx)
}

// Batch dispatches multiple sub-requests in a single call via `{namespace}/batch`
// (requires CoCart Plus). Returns one merged, up-to-date cart response with
// per-operation notices, instead of one response per request. See
// [CartEndpoint.BatchUpdateItems] and [CartEndpoint.BatchRemoveItems] for
// typed convenience wrappers over this.
func (c *Client) Batch(ctx context.Context, requests []BatchRequestItem) (*Response, error) {
	if len(requests) == 0 {
		return nil, NewValidationError("Batch() requires at least one request.", 0, "cocart_batch_empty")
	}

	c.mu.RLock()
	namespace := c.namespace
	cartKey := c.cartKey
	authenticated := c.auth != nil || c.jwtToken != ""
	c.mu.RUnlock()

	var params map[string]string
	if cartKey != "" && !authenticated {
		params = map[string]string{"cart_key": cartKey}
	}

	resp, err := c.RequestRaw(ctx, http.MethodPost, namespace+"/batch", params, map[string]any{"requests": requests})
	if err != nil {
		var cocartErr *CoCartError
		if errors.As(err, &cocartErr) && cocartErr.ErrorCode == "rest_no_route" {
			return resp, NewCoCartError(
				"This method is only available with another CoCart plugin. Please ask support for assistance!",
				404,
				"cocart_plugin_required",
			)
		}
		return resp, err
	}
	return resp, nil
}

// --- Endpoints (lazy-loaded) ---

// Cart returns the cart endpoint.
func (c *Client) Cart() *CartEndpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cartEndpoint == nil {
		c.cartEndpoint = &CartEndpoint{endpoint: endpoint{client: c, basePath: "cart"}}
	}
	return c.cartEndpoint
}

// Products returns the products endpoint.
func (c *Client) Products() *ProductsEndpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.productsEndpoint == nil {
		c.productsEndpoint = &ProductsEndpoint{endpoint: endpoint{client: c, basePath: "products"}}
	}
	return c.productsEndpoint
}

// Store returns the store endpoint.
func (c *Client) Store() *StoreEndpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.storeEndpoint == nil {
		c.storeEndpoint = &StoreEndpoint{endpoint: endpoint{client: c, basePath: "store"}}
	}
	return c.storeEndpoint
}

// Sessions returns the sessions endpoint (admin).
func (c *Client) Sessions() *SessionsEndpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessionsEndpoint == nil {
		c.sessionsEndpoint = &SessionsEndpoint{endpoint: endpoint{client: c, basePath: "sessions"}}
	}
	return c.sessionsEndpoint
}

// Account returns the account endpoint for customer profile, orders, downloads, and reviews.
//
// Note: the CoCart Account API is not yet available in a released version of
// the CoCart plugin — this endpoint is implemented and ready for when it
// ships.
func (c *Client) Account() *AccountEndpoint {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.accountEndpoint == nil {
		c.accountEndpoint = &AccountEndpoint{endpoint: endpoint{client: c}}
	}
	return c.accountEndpoint
}

// --- HTTP methods ---

// Get makes an HTTP GET request to the API.
func (c *Client) Get(ctx context.Context, endpoint string, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	return c.request(ctx, http.MethodGet, endpoint, p, nil)
}

// Post makes an HTTP POST request to the API.
func (c *Client) Post(ctx context.Context, endpoint string, data any, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	return c.request(ctx, http.MethodPost, endpoint, p, data)
}

// Put makes an HTTP PUT request to the API.
func (c *Client) Put(ctx context.Context, endpoint string, data any, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	return c.request(ctx, http.MethodPut, endpoint, p, data)
}

// Delete makes an HTTP DELETE request to the API.
func (c *Client) Delete(ctx context.Context, endpoint string, params ...map[string]string) (*Response, error) {
	var p map[string]string
	if len(params) > 0 {
		p = params[0]
	}
	return c.request(ctx, http.MethodDelete, endpoint, p, nil)
}

// RequestRaw makes an HTTP request using a full REST route (no namespace/version prefix).
func (c *Client) RequestRaw(ctx context.Context, method, route string, params map[string]string, data any) (*Response, error) {
	return c.requestRaw(ctx, method, route, params, data, false)
}

// requestRaw is RequestRaw, optionally without the cart key for routes that
// refuse one (products/{id}/shipping-rates).
//
// It emits the same events as the other requests, so debug output and
// listeners see the CoCart Plus routes too. It's never sent again: a raw
// request can take payment.
func (c *Client) requestRaw(ctx context.Context, method, route string, params map[string]string, data any, omitCartKey bool) (*Response, error) {
	c.mu.RLock()
	rawURL := fmt.Sprintf("%s/%s/%s", c.storeURL, c.restPrefix, strings.TrimLeft(route, "/"))
	c.mu.RUnlock()

	if len(params) > 0 {
		rawURL += "?" + encodeParams(params)
	}

	headers := c.buildHeaders(!omitCartKey)
	var body []byte
	if data != nil {
		var err error
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("cocart: failed to marshal request body: %w", err)
		}
	}

	c.emitter.emitRequest(RequestEvent{Method: method, URL: rawURL, Headers: headers, Body: string(body)})
	start := time.Now()

	resp, err := c.doHTTP(ctx, method, rawURL, headers, body)
	if err != nil {
		c.emitter.emitError(ErrorEvent{Method: method, URL: rawURL, Err: err})
		return nil, err
	}

	c.mu.Lock()
	c.lastResponse = resp
	c.mu.Unlock()

	c.extractCartKeyFromHeaders(resp)

	c.emitter.emitResponse(ResponseEvent{Method: method, URL: rawURL, Status: resp.StatusCode, Duration: time.Since(start), Response: resp})

	if resp.StatusCode >= 400 {
		apiErr := c.handleErrorResponse(resp, method, rawURL)
		c.emitter.emitError(ErrorEvent{Method: method, URL: rawURL, Err: apiErr})
		return resp, apiErr
	}

	return c.applyTransformer(resp), nil
}

// request is the main request method.
//
// If a guest's cart no longer exists (CoCart Starter 5 refuses the cart key
// with cocart_invalid_cart, for example after the cart expired), the stored
// cart key is forgotten and the request retried once as a new guest.
func (c *Client) request(ctx context.Context, method, endpoint string, params map[string]string, data any) (*Response, error) {
	resp, err := c.requestWithAuthRefresh(ctx, method, endpoint, params, data)
	if err == nil {
		return resp, nil
	}

	// The stored guest cart no longer exists (e.g. it expired). CoCart refuses
	// the request before running it, so forget the key and retry once as a
	// new guest.
	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) || cocartErr.ErrorCode != "cocart_invalid_cart" || c.GetCartKey() == "" || c.IsAuthenticated() {
		return resp, err
	}
	_ = c.forgetCartKey()
	return c.requestWithAuthRefresh(ctx, method, endpoint, params, data)
}

// requestWithAuthRefresh makes the request; if a JWTManager with auto-refresh
// is attached and the request fails with an authentication error, the token
// is refreshed and the request retried once.
func (c *Client) requestWithAuthRefresh(ctx context.Context, method, endpoint string, params map[string]string, data any) (*Response, error) {
	resp, err := c.executeRequest(ctx, method, endpoint, params, data)
	if err != nil {
		var authErr *AuthenticationError
		if errors.As(err, &authErr) {
			c.mu.RLock()
			hasRefresh := c.refreshToken != ""
			jwtMgr := c.jwtManager
			c.mu.RUnlock()

			if jwtMgr != nil && jwtMgr.IsAutoRefreshEnabled() && hasRefresh {
				refreshErr := jwtMgr.doRefresh(ctx)
				if refreshErr == nil {
					c.emitter.emitAuthRefresh(AuthRefreshEvent{Success: true})
					return c.executeRequest(ctx, method, endpoint, params, data)
				}
				c.emitter.emitAuthRefresh(AuthRefreshEvent{Success: false})
			}
		}
		return resp, err
	}
	return resp, nil
}

// executeRequest performs the actual HTTP request with retry logic,
// de-duplicating concurrent identical GET requests.
func (c *Client) executeRequest(ctx context.Context, method, endpoint string, params map[string]string, data any) (*Response, error) {
	reqURL := c.buildURL(endpoint, params)

	if method != http.MethodGet {
		return c.performRequestWithRetry(ctx, method, reqURL, data)
	}

	// De-duplicate identical concurrent GETs (e.g. two goroutines requesting
	// the same resource at once) so they share one network round trip
	// instead of firing one each.
	c.inFlightMu.Lock()
	if call, ok := c.inFlightGets[reqURL]; ok {
		c.inFlightMu.Unlock()
		<-call.done
		return call.resp, call.err
	}
	call := &inFlightGet{done: make(chan struct{})}
	c.inFlightGets[reqURL] = call
	c.inFlightMu.Unlock()

	resp, err := c.performRequestWithRetry(ctx, method, reqURL, data)

	call.resp, call.err = resp, err
	close(call.done)

	c.inFlightMu.Lock()
	delete(c.inFlightGets, reqURL)
	c.inFlightMu.Unlock()

	return resp, err
}

// performRequestWithRetry performs the actual HTTP request with retry logic.
func (c *Client) performRequestWithRetry(ctx context.Context, method, reqURL string, data any) (*Response, error) {
	headers := c.buildHeaders(true)

	var body []byte
	if data != nil {
		var err error
		body, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("cocart: failed to marshal request body: %w", err)
		}
	}

	// ETag: add If-None-Match for GET requests
	c.mu.RLock()
	if method == http.MethodGet && c.etagEnabled {
		if cached, ok := c.etagCache[reqURL]; ok {
			headers["If-None-Match"] = cached.etag
		}
	}
	maxRetries := c.maxRetries
	c.mu.RUnlock()

	c.emitter.emitRequest(RequestEvent{
		Method:  method,
		URL:     reqURL,
		Headers: headers,
		Body:    string(body),
	})

	start := time.Now()
	attempt := 0

	for {
		resp, err := c.doHTTP(ctx, method, reqURL, headers, body)
		if err != nil {
			// Only a GET is sent again: a request that changes something may
			// have run before the connection dropped, and running it twice
			// could, for example, charge a card twice.
			if attempt < maxRetries && method == http.MethodGet && isTransientError(err) {
				attempt++
				delay := c.getRetryDelay(attempt, nil)
				c.emitter.emitRetry(RetryEvent{
					Method:     method,
					URL:        reqURL,
					Attempt:    attempt,
					MaxRetries: maxRetries,
					Delay:      delay,
					Reason:     "transient_error",
				})
				if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
					return nil, sleepErr
				}
				continue
			}
			var cocartErr *CoCartError
			if !errors.As(err, &cocartErr) {
				err = NewCoCartError(err.Error(), 0, "network_error")
			}
			c.emitter.emitError(ErrorEvent{Method: method, URL: reqURL, Err: err})
			return nil, err
		}

		duration := time.Since(start)

		// A 304 has no body — substitute the body/headers cached alongside
		// the ETag that produced the match, so callers still get the actual
		// data instead of an empty response. Falls back to the live (empty)
		// response if we have no cache entry for this URL.
		if method == http.MethodGet && c.etagEnabled && resp.StatusCode == 304 {
			c.mu.RLock()
			cached, ok := c.etagCache[reqURL]
			c.mu.RUnlock()
			if ok {
				resp.Body = cached.body
				resp.Headers = cached.headers
			}
		}

		c.mu.Lock()
		c.lastResponse = resp
		c.mu.Unlock()

		c.extractCartKeyFromHeaders(resp)

		// Cache the ETag + body from a fresh (non-304) GET response, unless
		// the store says it mustn't be kept (checkout, orders and the account
		// are "no-store, private").
		if method == http.MethodGet && resp.StatusCode != 304 && mayStore(resp.GetHeader("Cache-Control")) {
			c.mu.Lock()
			if c.etagEnabled {
				if etag := resp.GetETag(); etag != "" {
					c.etagCache[reqURL] = etagCacheEntry{etag: etag, body: resp.Body, headers: resp.Headers}
				}
			}
			c.mu.Unlock()
		}

		// Retry on transient HTTP status codes (429, and 503 for a GET)
		if attempt < maxRetries && isRetryableStatus(method, resp.StatusCode) {
			attempt++
			delay := c.getRetryDelay(attempt, resp)
			c.emitter.emitRetry(RetryEvent{
				Method:     method,
				URL:        reqURL,
				Attempt:    attempt,
				MaxRetries: maxRetries,
				Delay:      delay,
				Reason:     fmt.Sprintf("http_%d", resp.StatusCode),
			})
			if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if resp.StatusCode >= 400 {
			c.emitter.emitResponse(ResponseEvent{
				Method:   method,
				URL:      reqURL,
				Status:   resp.StatusCode,
				Duration: duration,
				Response: resp,
			})
			return resp, c.handleErrorResponse(resp, method, reqURL)
		}

		c.emitter.emitResponse(ResponseEvent{
			Method:   method,
			URL:      reqURL,
			Status:   resp.StatusCode,
			Duration: duration,
			Response: resp,
		})

		return c.applyTransformer(resp), nil
	}
}

// doHTTP performs a single HTTP request.
func (c *Client) doHTTP(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) (*Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		// Not wrapped: the error names the full URL, query string (cart key) included.
		return nil, NewCoCartError("cocart: failed to create request: "+safeErrorMessage(err.Error()), 0, "network_error")
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.mu.RLock()
	client := c.httpClient
	c.mu.RUnlock()

	httpResp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, NewCoCartError(
				fmt.Sprintf("Request timed out or cancelled: %v", ctx.Err()),
				0,
				"request_timeout",
			)
		}
		// Not wrapped: net/http's *url.Error names the full URL, query string
		// (cart key) included.
		return nil, NewCoCartError(safeErrorMessage(err.Error()), 0, "network_error")
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("cocart: failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    httpResp.Header,
		Body:       respBody,
	}, nil
}

// buildURL constructs the full API URL.
func (c *Client) buildURL(endpoint string, params map[string]string) string {
	c.mu.RLock()
	resolvedParams := make(map[string]string)
	for k, v := range params {
		resolvedParams[k] = v
	}

	// Add cart_key if set and not authenticated
	if c.cartKey != "" && c.auth == nil && c.jwtToken == "" {
		resolvedParams["cart_key"] = c.cartKey
	}

	// Normalize field filtering parameter based on main plugin
	if c.mainPlugin == MainPluginLegacy {
		if v, ok := resolvedParams["_fields"]; ok {
			if _, has := resolvedParams["fields"]; !has {
				resolvedParams["fields"] = v
				delete(resolvedParams, "_fields")
			}
		}
	} else {
		if v, ok := resolvedParams["fields"]; ok {
			if _, has := resolvedParams["_fields"]; !has {
				resolvedParams["_fields"] = v
				delete(resolvedParams, "fields")
			}
		}
	}

	rawURL := fmt.Sprintf("%s/%s/%s/%s/%s", c.storeURL, c.restPrefix, c.namespace, APIVersion, strings.TrimLeft(endpoint, "/"))
	c.mu.RUnlock()

	if len(resolvedParams) > 0 {
		rawURL += "?" + encodeParams(resolvedParams)
	}

	return rawURL
}

// buildHeaders constructs the request headers, with the cart key header
// unless includeCartKey is false.
func (c *Client) buildHeaders(includeCartKey bool) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	headers := map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
		"User-Agent":   "CoCart-Go-SDK/" + Version,
	}

	// Authentication
	if c.jwtToken != "" {
		headers[c.authHeaderName] = "Bearer " + c.jwtToken
	} else if c.auth != nil {
		encoded := base64.StdEncoding.EncodeToString([]byte(c.auth.username + ":" + c.auth.password))
		headers[c.authHeaderName] = "Basic " + encoded
	} else if c.consumerKey != "" && c.consumerSecret != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(c.consumerKey + ":" + c.consumerSecret))
		headers[c.authHeaderName] = "Basic " + encoded
	}

	// Cart key header: send only the header name the configured plugin
	// variant actually reads.
	if includeCartKey && c.cartKey != "" && c.auth == nil && c.jwtToken == "" {
		cartKeyHeader := "Cart-Key"
		if c.mainPlugin == MainPluginLegacy {
			cartKeyHeader = "CoCart-API-Cart-Key"
		}
		headers[cartKeyHeader] = c.cartKey
	}

	// Custom headers (override defaults)
	for k, v := range c.customHeaders {
		headers[k] = v
	}

	return headers
}

// extractCartKeyFromHeaders extracts and stores the Cart-Key from response headers.
func (c *Client) extractCartKeyFromHeaders(resp *Response) {
	cartKey := resp.GetHeader("Cart-Key")
	if cartKey == "" {
		cartKey = resp.GetHeader("CoCart-API-Cart-Key") // Fallback for older plugin versions
	}
	if cartKey != "" {
		c.mu.Lock()
		c.cartKey = cartKey
		storage := c.storage
		storageKey := c.storageKey
		c.mu.Unlock()
		_ = storage.Set(storageKey, cartKey)
	}
}

// handleErrorResponse classifies and returns an appropriate error.
func (c *Client) handleErrorResponse(resp *Response, method, reqURL string) error {
	data := resp.ToObject()
	code, _ := data["code"].(string)
	if code == "" {
		code = "unknown_error"
	}
	apiMessage, _ := data["message"].(string)
	if apiMessage == "" {
		apiMessage = "An unknown error occurred"
	}

	// "POST /cocart/v2/cart/add-item: Product not found [cocart_product_not_found]".
	// Route only: the full URL's query string can carry the cart key and
	// consumer key/secret, which must never end up in a message an app logs
	// or displays.
	context := ""
	if method != "" && reqURL != "" {
		context = fmt.Sprintf("%s %s: ", method, routePath(reqURL))
	}
	codeLabel := ""
	if code != "unknown_error" {
		codeLabel = fmt.Sprintf(" [%s]", code)
	}
	message := fmt.Sprintf("%s%s%s", context, apiMessage, codeLabel)

	httpCode := resp.StatusCode

	// 2FA challenge (checked before generic 401 handling)
	if code == "cocart_2fa_required" {
		return NewTwoFactorRequiredError(message, data)
	}

	// Authentication errors
	if httpCode == 401 || httpCode == 403 || strings.Contains(code, "authenticat") {
		return NewAuthenticationError(message, httpCode, code, data)
	}

	// A refused state (409), a rate limit (429) or the server's fault (5xx)
	// is never a validation error, whatever its code says (e.g.
	// cocart_order_invalid_status is a 409).
	if httpCode == 409 || httpCode == 429 || httpCode >= 500 {
		return NewCoCartError(message, httpCode, code, data)
	}

	// Validation errors
	if httpCode == 400 || strings.Contains(code, "invalid") || strings.Contains(code, "missing") {
		return NewValidationError(message, httpCode, code, data)
	}

	return NewCoCartError(message, httpCode, code, data)
}

// applyTransformer applies the response transformer if set.
func (c *Client) applyTransformer(resp *Response) *Response {
	c.mu.RLock()
	transformer := c.responseTransformer
	c.mu.RUnlock()
	if transformer != nil {
		return transformer(resp)
	}
	return resp
}

// getRetryDelay calculates the retry delay.
func (c *Client) getRetryDelay(attempt int, resp *Response) time.Duration {
	if resp != nil {
		// CoCart Plus sends RateLimit-Retry-After; a proxy may send Retry-After.
		retryAfter := resp.GetHeader("RateLimit-Retry-After")
		if retryAfter == "" {
			retryAfter = resp.GetHeader("Retry-After")
		}
		if retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				if seconds > 60 {
					seconds = 60
				}
				return time.Duration(seconds) * time.Second
			}
		}
	}
	// Exponential backoff: 1s, 2s, 4s, ... max 30s, with ±20% jitter so many
	// clients retrying at once don't re-collide on the same schedule.
	base := math.Min(math.Pow(2, float64(attempt-1)), 30)
	jitter := 0.8 + rand.Float64()*0.4
	return time.Duration(base * jitter * float64(time.Second))
}

// logEvent writes an event to the debug log (log.Printf) when debug
// logging is on. URLs are written with every query value replaced by "***":
// the query can hold the cart key, an order key, an email address or a
// verification key, which mustn't end up in logs.
func (c *Client) logEvent(event any) {
	c.mu.RLock()
	debug := c.debug
	c.mu.RUnlock()
	if !debug {
		return
	}
	switch e := event.(type) {
	case RequestEvent:
		log.Printf("[CoCart] %s %s", e.Method, redactQuery(e.URL))
	case ResponseEvent:
		log.Printf("[CoCart] %s %s -> %d (%dms)", e.Method, redactQuery(e.URL), e.Status, e.Duration.Milliseconds())
	case ErrorEvent:
		log.Printf("[CoCart] %s %s failed: %v", e.Method, redactQuery(e.URL), e.Err)
	case RetryEvent:
		log.Printf("[CoCart] retry %d/%d in %v (%s)", e.Attempt, e.MaxRetries, e.Delay, e.Reason)
	case AuthRefreshEvent:
		if e.Success {
			log.Printf("[CoCart] JWT token refreshed")
		} else {
			log.Printf("[CoCart] JWT token refresh failed")
		}
	}
}

// redactQuery returns a URL for debug output, with every query value
// replaced by "***".
func redactQuery(rawURL string) string {
	i := strings.Index(rawURL, "?")
	if i < 0 {
		return rawURL
	}
	pairs := strings.Split(rawURL[i+1:], "&")
	for j, pair := range pairs {
		if eq := strings.Index(pair, "="); eq >= 0 {
			pairs[j] = pair[:eq] + "=***"
		}
	}
	return rawURL[:i] + "?" + strings.Join(pairs, "&")
}

// mayStore reports whether a response may be kept in the ETag cache: not
// when its Cache-Control says no-store or private.
func mayStore(cacheControl string) bool {
	return !noStorePattern.MatchString(cacheControl)
}

var noStorePattern = regexp.MustCompile(`(?i)(^|[\s,])(no-store|private)([\s,;=]|$)`)

// --- Internal helpers ---

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "timed out") ||
		strings.Contains(msg, "connection")
}

// isRetryableStatus reports whether a response can be retried. A 429 is
// refused before the request runs, so any method can be sent again. A 503 can
// come from a proxy after the store has run the request, so only a GET is.
func isRetryableStatus(method string, status int) bool {
	return status == 429 || (status == 503 && method == http.MethodGet)
}

func encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

// routePath returns the REST route of a request URL for error messages: the
// path from /wp-json on, with no host and no query string (which can hold the
// cart key or API credentials).
func routePath(rawURL string) string {
	path := rawURL
	if u, err := url.Parse(rawURL); err == nil {
		path = u.EscapedPath()
	} else if i := strings.IndexAny(rawURL, "?#"); i >= 0 {
		path = rawURL[:i]
	}
	if i := strings.Index(path, "/wp-json/"); i >= 0 {
		return path[i+len("/wp-json"):]
	}
	return path
}

// urlQuery matches a URL or path followed by its query string or fragment.
var urlQuery = regexp.MustCompile(`(?i)((?:https?://|/)[^\s?#"'<>()]*)[?#][^\s"'<>()]*`)

// safeErrorMessage removes the query string from any URL or path in a
// transport error message: net/http puts the request URL in its errors
// (Get "https://store.com/wp-json/cocart/v2/cart?cart_key=…": …), and its
// query string can carry the cart key or API credentials.
func safeErrorMessage(message string) string {
	return urlQuery.ReplaceAllString(message, "$1")
}

func trimSlashes(s string) string {
	return strings.Trim(s, "/")
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
