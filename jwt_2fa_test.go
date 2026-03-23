package cocart

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var twoFAResponse = map[string]any{
	"code":    "cocart_2fa_required",
	"message": "Two factor authentication is required.",
	"data": map[string]any{
		"status":              401,
		"2fa_required":        true,
		"available_providers": []any{"totp", "email"},
		"default_provider":    "totp",
		"email_sent":          false,
	},
}

var twoFAEmailResponse = map[string]any{
	"code":    "cocart_2fa_required",
	"message": "Two factor authentication is required.",
	"data": map[string]any{
		"status":              401,
		"2fa_required":        true,
		"available_providers": []any{"email"},
		"default_provider":    "email",
		"email_sent":          true,
	},
}

var loginSuccessResponse = map[string]any{
	"extras": map[string]any{
		"jwt_token":   "access123",
		"jwt_refresh": "refresh456",
	},
}

// ---------------------------------------------------------------------------
// TwoFactorRequiredError — type and fields
// ---------------------------------------------------------------------------

func TestNewTwoFactorRequiredError(t *testing.T) {
	err := NewTwoFactorRequiredError("2FA required", twoFAResponse)

	if err.ErrorCode != "cocart_2fa_required" {
		t.Errorf("ErrorCode = %s", err.ErrorCode)
	}
	if err.HTTPCode != 401 {
		t.Errorf("HTTPCode = %d", err.HTTPCode)
	}
	if err.DefaultProvider != "totp" {
		t.Errorf("DefaultProvider = %s", err.DefaultProvider)
	}
	if err.EmailSent != false {
		t.Error("EmailSent should be false")
	}
	if len(err.AvailableProviders) != 2 ||
		err.AvailableProviders[0] != "totp" ||
		err.AvailableProviders[1] != "email" {
		t.Errorf("AvailableProviders = %v", err.AvailableProviders)
	}
}

func TestTwoFactorRequiredErrorEmailSent(t *testing.T) {
	err := NewTwoFactorRequiredError("2FA required", twoFAEmailResponse)
	if !err.EmailSent {
		t.Error("EmailSent should be true")
	}
	if err.DefaultProvider != "email" {
		t.Errorf("DefaultProvider = %s", err.DefaultProvider)
	}
}

func TestTwoFactorRequiredError_IsAuthenticationError(t *testing.T) {
	err := NewTwoFactorRequiredError("2FA required", twoFAResponse)

	var twoFactorErr *TwoFactorRequiredError
	if !errors.As(err, &twoFactorErr) {
		t.Error("should be *TwoFactorRequiredError")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Error("should also satisfy *AuthenticationError via errors.As")
	}

	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) {
		t.Error("should also satisfy *CoCartError via errors.As")
	}
}

// ---------------------------------------------------------------------------
// JWTManager.Login() — 2FA detection
// ---------------------------------------------------------------------------

func TestJWTManager_Login_Raises2FAError(t *testing.T) {
	body, _ := json.Marshal(twoFAResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)

	_, err := jwtMgr.Login(context.Background(), "user", "pass")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var twoFactorErr *TwoFactorRequiredError
	if !errors.As(err, &twoFactorErr) {
		t.Fatalf("expected *TwoFactorRequiredError, got %T: %v", err, err)
	}
	if twoFactorErr.DefaultProvider != "totp" {
		t.Errorf("DefaultProvider = %s", twoFactorErr.DefaultProvider)
	}
	if len(twoFactorErr.AvailableProviders) != 2 {
		t.Errorf("AvailableProviders = %v", twoFactorErr.AvailableProviders)
	}
}

func TestJWTManager_Login_EmailSentFlag(t *testing.T) {
	body, _ := json.Marshal(twoFAEmailResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)

	_, err := jwtMgr.Login(context.Background(), "user", "pass")

	var twoFactorErr *TwoFactorRequiredError
	if !errors.As(err, &twoFactorErr) {
		t.Fatalf("expected *TwoFactorRequiredError, got %T", err)
	}
	if !twoFactorErr.EmailSent {
		t.Error("EmailSent should be true")
	}
}

func TestJWTManager_Login_Regular401_IsAuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"code":"cocart_authentication_error","message":"Invalid credentials."}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)

	_, err := jwtMgr.Login(context.Background(), "user", "wrong")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var twoFactorErr *TwoFactorRequiredError
	if errors.As(err, &twoFactorErr) {
		t.Error("should NOT be *TwoFactorRequiredError for a regular 401")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// JWTManager.LoginWith2FA()
// ---------------------------------------------------------------------------

func TestJWTManager_LoginWith2FA_SendsCorrectPayloadWithoutProvider(t *testing.T) {
	var received map[string]any
	body, _ := json.Marshal(loginSuccessResponse)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)
	jwtMgr.LoginWith2FA(context.Background(), "user", "pass", "123456")

	if received["username"] != "user" {
		t.Errorf("username = %v", received["username"])
	}
	if received["password"] != "pass" {
		t.Errorf("password = %v", received["password"])
	}
	if received["2fa_code"] != "123456" {
		t.Errorf("2fa_code = %v", received["2fa_code"])
	}
	if _, ok := received["2fa_provider"]; ok {
		t.Error("2fa_provider should not be sent when not specified")
	}
}

func TestJWTManager_LoginWith2FA_SendsCorrectPayloadWithProvider(t *testing.T) {
	var received map[string]any
	body, _ := json.Marshal(loginSuccessResponse)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)
	jwtMgr.LoginWith2FA(context.Background(), "user", "pass", "123456", "email")

	if received["2fa_provider"] != "email" {
		t.Errorf("2fa_provider = %v", received["2fa_provider"])
	}
}

func TestJWTManager_LoginWith2FA_ExtractsTokens(t *testing.T) {
	body, _ := json.Marshal(loginSuccessResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)
	_, err := jwtMgr.LoginWith2FA(context.Background(), "user", "pass", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.GetJWTToken() != "access123" {
		t.Errorf("jwt token = %s", c.GetJWTToken())
	}
	if c.GetRefreshToken() != "refresh456" {
		t.Errorf("refresh token = %s", c.GetRefreshToken())
	}
}

func TestJWTManager_LoginWith2FA_ErrorOnMissingJWT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"extras":{}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	jwtMgr := NewJWTManager(c, nil)
	_, err := jwtMgr.LoginWith2FA(context.Background(), "user", "pass", "123456")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Errorf("expected *AuthenticationError, got %T", err)
	}
	if authErr.ErrorCode != "cocart_jwt_missing" {
		t.Errorf("ErrorCode = %s", authErr.ErrorCode)
	}
}

func TestJWTManager_LoginWith2FA_PersistsTokensToStorage(t *testing.T) {
	body, _ := json.Marshal(loginSuccessResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	storage := NewMemoryStorage()
	jwtMgr := NewJWTManager(c, storage)
	jwtMgr.LoginWith2FA(context.Background(), "user", "pass", "123456")

	tok, _ := storage.Get("cocart_jwt_token")
	ref, _ := storage.Get("cocart_jwt_refresh_token")
	if tok != "access123" {
		t.Errorf("stored token = %s", tok)
	}
	if ref != "refresh456" {
		t.Errorf("stored refresh = %s", ref)
	}
}

// ---------------------------------------------------------------------------
// WithAutoRefreshCallback — must NOT retry on 2FA challenge
// ---------------------------------------------------------------------------

func TestJWTManager_WithAutoRefreshCallback_DoesNotRetryOn2FA(t *testing.T) {
	body, _ := json.Marshal(twoFAResponse)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write(body)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	c.SetRefreshToken("some_refresh_token")
	jwtMgr := NewJWTManager(c, nil)

	callCount := 0
	err := jwtMgr.WithAutoRefreshCallback(context.Background(), func(ctx context.Context, cl *Client) error {
		callCount++
		_, err := cl.Post(ctx, "login", map[string]any{"username": "user", "password": "pass"})
		return err
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var twoFactorErr *TwoFactorRequiredError
	if !errors.As(err, &twoFactorErr) {
		t.Errorf("expected *TwoFactorRequiredError, got %T", err)
	}

	// Must only be called once — no retry after a 2FA challenge
	if callCount != 1 {
		t.Errorf("callback called %d times, expected 1", callCount)
	}
}
