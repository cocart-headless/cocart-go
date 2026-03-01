package cocart

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

// JWTOption configures a JWTManager.
type JWTOption func(*JWTManager)

// WithAutoRefresh enables or disables automatic token refresh on auth errors.
func WithAutoRefresh(enabled bool) JWTOption {
	return func(j *JWTManager) {
		j.autoRefresh = enabled
	}
}

// WithTokenStorageKey sets the storage key for the JWT access token.
func WithTokenStorageKey(key string) JWTOption {
	return func(j *JWTManager) {
		j.tokenStorageKey = key
	}
}

// WithRefreshTokenStorageKey sets the storage key for the JWT refresh token.
func WithRefreshTokenStorageKey(key string) JWTOption {
	return func(j *JWTManager) {
		j.refreshTokenStorageKey = key
	}
}

// JWTManager handles JWT token lifecycle: acquisition, refresh, validation,
// and optional persistence using a storage adapter.
type JWTManager struct {
	client              *Client
	storage             Storage
	tokenStorageKey     string
	refreshTokenStorageKey string
	autoRefresh         bool
	isRefreshing        bool
	mu                  sync.Mutex
}

// NewJWTManager creates a new JWTManager.
func NewJWTManager(client *Client, storage Storage, opts ...JWTOption) *JWTManager {
	j := &JWTManager{
		client:              client,
		storage:             storage,
		tokenStorageKey:     "cocart_jwt_token",
		refreshTokenStorageKey: "cocart_jwt_refresh_token",
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

// RestoreTokensFromStorage restores JWT tokens from storage into the client.
func (j *JWTManager) RestoreTokensFromStorage() error {
	if j.storage == nil {
		return nil
	}

	storedToken, err := j.storage.Get(j.tokenStorageKey)
	if err != nil && !errors.Is(err, ErrKeyNotFound) {
		return err
	}
	storedRefresh, err := j.storage.Get(j.refreshTokenStorageKey)
	if err != nil && !errors.Is(err, ErrKeyNotFound) {
		return err
	}

	if storedToken != "" {
		j.client.SetJWTToken(storedToken)
	}
	if storedRefresh != "" {
		j.client.SetRefreshToken(storedRefresh)
	}
	return nil
}

// Login authenticates with username and password to acquire JWT tokens.
func (j *JWTManager) Login(ctx context.Context, username, password string) (*Response, error) {
	resp, err := j.client.Post(ctx, "login", map[string]any{
		"username": username,
		"password": password,
	})
	if err != nil {
		return resp, err
	}

	data := resp.ToObject()
	extras, _ := data["extras"].(map[string]any)
	jwtToken, _ := extras["jwt_token"].(string)
	refreshToken, _ := extras["jwt_refresh"].(string)

	if jwtToken != "" {
		j.client.SetJWTToken(jwtToken)
		if refreshToken != "" {
			j.client.SetRefreshToken(refreshToken)
		}
		if err := j.persistTokens(); err != nil {
			return resp, err
		}
	} else {
		return resp, NewAuthenticationError(
			"JWT token not found in login response. Is the CoCart JWT Authentication plugin installed?",
			0,
			"cocart_jwt_missing",
		)
	}

	return resp, nil
}

// Refresh refreshes the JWT access token using the refresh token.
func (j *JWTManager) Refresh(ctx context.Context, refreshToken ...string) (*Response, error) {
	return j.doRefreshWithContext(ctx, refreshToken...)
}

// doRefresh performs a token refresh (called internally by the client).
func (j *JWTManager) doRefresh(ctx context.Context) error {
	_, err := j.doRefreshWithContext(ctx)
	return err
}

func (j *JWTManager) doRefreshWithContext(ctx context.Context, refreshToken ...string) (*Response, error) {
	token := ""
	if len(refreshToken) > 0 {
		token = refreshToken[0]
	}
	if token == "" {
		token = j.client.GetRefreshToken()
	}

	if token == "" {
		return nil, NewAuthenticationError(
			"No refresh token available. Please login first.",
			0,
			"cocart_jwt_no_refresh_token",
		)
	}

	route := j.client.GetNamespace() + "/jwt/refresh-token"
	resp, err := j.client.RequestRaw(ctx, "POST", route, nil, map[string]any{
		"refresh_token": token,
	})
	if err != nil {
		return resp, err
	}

	data := resp.ToObject()
	if newToken, ok := data["token"].(string); ok && newToken != "" {
		j.client.SetJWTToken(newToken)
	}
	if newRefresh, ok := data["refresh_token"].(string); ok && newRefresh != "" {
		j.client.SetRefreshToken(newRefresh)
	}

	if err := j.persistTokens(); err != nil {
		return resp, err
	}

	return resp, nil
}

// Validate validates the current JWT token with the server.
func (j *JWTManager) Validate(ctx context.Context) (bool, error) {
	if !j.client.HasJWTToken() {
		return false, nil
	}

	route := j.client.GetNamespace() + "/jwt/validate-token"
	resp, err := j.client.RequestRaw(ctx, "POST", route, nil, nil)
	if err != nil {
		var authErr *AuthenticationError
		if errors.As(err, &authErr) {
			return false, nil
		}
		return false, err
	}
	return resp.IsSuccessful(), nil
}

// WithAutoRefreshCallback executes a callback with automatic token refresh on auth error.
func (j *JWTManager) WithAutoRefreshCallback(ctx context.Context, fn func(ctx context.Context, c *Client) error) error {
	err := fn(ctx, j.client)
	if err != nil {
		var authErr *AuthenticationError
		if errors.As(err, &authErr) && !j.isRefreshing && j.client.GetRefreshToken() != "" {
			j.mu.Lock()
			j.isRefreshing = true
			j.mu.Unlock()

			defer func() {
				j.mu.Lock()
				j.isRefreshing = false
				j.mu.Unlock()
			}()

			if refreshErr := j.doRefresh(ctx); refreshErr != nil {
				return err
			}
			return fn(ctx, j.client)
		}
		return err
	}
	return nil
}

// ClearTokens removes all JWT tokens from the client and storage.
func (j *JWTManager) ClearTokens() error {
	j.client.ClearJWTToken()
	if j.storage != nil {
		_ = j.storage.Delete(j.tokenStorageKey)
		_ = j.storage.Delete(j.refreshTokenStorageKey)
	}
	return nil
}

// HasTokens returns true if the client has a JWT token.
func (j *JWTManager) HasTokens() bool {
	return j.client.HasJWTToken()
}

// IsTokenExpired checks if the current JWT token is expired.
// An optional leeway duration can be specified (default: 30 seconds).
func (j *JWTManager) IsTokenExpired(leeway ...time.Duration) bool {
	token := j.client.GetJWTToken()
	if token == "" {
		return true
	}

	payload := decodeTokenPayload(token)
	if payload == nil {
		return true
	}

	exp, ok := payload["exp"].(float64)
	if !ok {
		return false
	}

	l := 30 * time.Second
	if len(leeway) > 0 {
		l = leeway[0]
	}

	return float64(time.Now().Unix()) >= exp-l.Seconds()
}

// GetTokenExpiry returns the expiry time of the current JWT token.
func (j *JWTManager) GetTokenExpiry() *time.Time {
	token := j.client.GetJWTToken()
	if token == "" {
		return nil
	}

	payload := decodeTokenPayload(token)
	if payload == nil {
		return nil
	}

	exp, ok := payload["exp"].(float64)
	if !ok {
		return nil
	}

	t := time.Unix(int64(exp), 0)
	return &t
}

// SetAutoRefreshEnabled enables or disables automatic token refresh.
func (j *JWTManager) SetAutoRefreshEnabled(enabled bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.autoRefresh = enabled
}

// IsAutoRefreshEnabled returns whether auto refresh is enabled.
func (j *JWTManager) IsAutoRefreshEnabled() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.autoRefresh
}

// GetClient returns the associated client.
func (j *JWTManager) GetClient() *Client {
	return j.client
}

// --- Internal ---

func (j *JWTManager) persistTokens() error {
	if j.storage == nil {
		return nil
	}

	token := j.client.GetJWTToken()
	refreshToken := j.client.GetRefreshToken()

	if token != "" {
		if err := j.storage.Set(j.tokenStorageKey, token); err != nil {
			return err
		}
	}
	if refreshToken != "" {
		if err := j.storage.Set(j.refreshTokenStorageKey, refreshToken); err != nil {
			return err
		}
	}
	return nil
}

// decodeTokenPayload decodes the payload section of a JWT token without verification.
func decodeTokenPayload(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}

	return result
}
