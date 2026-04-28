package cocart

import (
	"context"
	"errors"
)

// SessionManager helps manage cart sessions, especially useful for
// tracking guest customer carts and handling the transition to authenticated users.
type SessionManager struct {
	client     *Client
	storage    Storage
	storageKey string
	jwtManager *JWTManager
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(client *Client, storage Storage) *SessionManager {
	return &SessionManager{
		client:     client,
		storage:    storage,
		storageKey: "cocart_cart_key",
	}
}

// SetStorageKey sets a custom storage key name.
func (s *SessionManager) SetStorageKey(key string) *SessionManager {
	s.storageKey = key
	return s
}

// GetCartKey returns the current cart key.
func (s *SessionManager) GetCartKey() string {
	return s.client.GetCartKey()
}

// SetCartKey sets the cart key and persists it to storage.
func (s *SessionManager) SetCartKey(cartKey string) error {
	s.client.SetCartKey(cartKey)
	if s.storage != nil {
		return s.storage.Set(s.storageKey, cartKey)
	}
	return nil
}

// InitializeCart creates a new cart session and returns the cart key.
func (s *SessionManager) InitializeCart(ctx context.Context) (string, error) {
	resp, err := s.client.Cart().Get(ctx)
	if err != nil {
		return "", err
	}

	cartKey := resp.GetCartKey()
	if cartKey == "" {
		cartKey = s.client.GetCartKey()
	}

	if cartKey != "" && s.storage != nil {
		_ = s.storage.Set(s.storageKey, cartKey)
	}

	return cartKey, nil
}

// Login authenticates with Basic Auth and optionally transfers the guest cart.
func (s *SessionManager) Login(ctx context.Context, username, password string, mergeCart bool) (*Response, error) {
	guestCartKey := s.client.GetCartKey()

	s.client.SetAuth(username, password)
	s.clearStoredCartKey()

	if mergeCart && guestCartKey != "" {
		return s.client.Cart().Get(ctx, &CartGetParams{CartKey: guestCartKey})
	}
	return s.client.Cart().Get(ctx)
}

// LoginWithToken authenticates with an existing JWT token.
func (s *SessionManager) LoginWithToken(ctx context.Context, token string) (*Response, error) {
	guestCartKey := s.client.GetCartKey()

	s.client.SetJWTToken(token)
	s.clearStoredCartKey()

	if guestCartKey != "" {
		return s.client.Cart().Get(ctx, &CartGetParams{CartKey: guestCartKey})
	}
	return s.client.Cart().Get(ctx)
}

// JWT returns the JWT manager instance.
func (s *SessionManager) JWT(opts ...JWTOption) *JWTManager {
	if s.jwtManager == nil {
		s.jwtManager = NewJWTManager(s.client, s.storage, opts...)
	}
	return s.jwtManager
}

// LoginWithJWT authenticates via JWT and optionally merges the guest cart.
func (s *SessionManager) LoginWithJWT(ctx context.Context, username, password string, mergeCart bool) (*Response, error) {
	guestCartKey := s.client.GetCartKey()

	loginResp, err := s.JWT().Login(ctx, username, password)
	if err != nil {
		return loginResp, err
	}

	s.clearStoredCartKey()

	if mergeCart && guestCartKey != "" {
		_, _ = s.client.Cart().Get(ctx, &CartGetParams{CartKey: guestCartKey})
	}

	return loginResp, nil
}

// LoginWithJWT2FA completes a JWT login after a [*TwoFactorRequiredError].
//
// Call this after catching [*TwoFactorRequiredError] from [SessionManager.LoginWithJWT].
// Pass an empty string for provider to use the server's default.
func (s *SessionManager) LoginWithJWT2FA(ctx context.Context, username, password, code, provider string, mergeCart bool) (*Response, error) {
	guestCartKey := s.client.GetCartKey()

	loginResp, err := s.JWT().LoginWith2FA(ctx, username, password, code, provider)
	if err != nil {
		return loginResp, err
	}

	s.clearStoredCartKey()

	if mergeCart && guestCartKey != "" {
		_, _ = s.client.Cart().Get(ctx, &CartGetParams{CartKey: guestCartKey})
	}

	return loginResp, nil
}

// Logout clears all auth state and starts a new guest session.
func (s *SessionManager) Logout(ctx context.Context) error {
	if s.jwtManager != nil {
		_ = s.jwtManager.ClearTokens()
	}
	if err := s.client.ClearSession(); err != nil && !errors.Is(err, ErrKeyNotFound) {
		return err
	}
	s.clearStoredCartKey()
	return nil
}

// IsAuthenticated returns true if the client has credentials.
func (s *SessionManager) IsAuthenticated() bool {
	return s.client.IsAuthenticated()
}

// IsGuest returns true if the client has no credentials.
func (s *SessionManager) IsGuest() bool {
	return s.client.IsGuest()
}

// GetClient returns the associated client.
func (s *SessionManager) GetClient() *Client {
	return s.client
}

func (s *SessionManager) clearStoredCartKey() {
	if s.storage != nil {
		_ = s.storage.Delete(s.storageKey)
	}
}
