package cocart

import (
	"errors"
	"fmt"
)

// ErrKeyNotFound is returned by Storage when a key does not exist.
var ErrKeyNotFound = errors.New("cocart: key not found in storage")

// CoCartError is the base error type for all SDK errors.
type CoCartError struct {
	Message      string
	HTTPCode     int
	ErrorCode    string
	ResponseData map[string]any
}

func (e *CoCartError) Error() string {
	return e.Message
}

// NewCoCartError creates a new CoCartError.
func NewCoCartError(message string, httpCode int, errorCode string, responseData ...map[string]any) *CoCartError {
	var data map[string]any
	if len(responseData) > 0 {
		data = responseData[0]
	}
	return &CoCartError{
		Message:      message,
		HTTPCode:     httpCode,
		ErrorCode:    errorCode,
		ResponseData: data,
	}
}

// AuthenticationError represents a 401/403 authentication failure.
type AuthenticationError struct {
	CoCartError
}

// Unwrap returns the underlying CoCartError for errors.As compatibility.
func (e *AuthenticationError) Unwrap() error {
	return &e.CoCartError
}

// NewAuthenticationError creates a new AuthenticationError.
func NewAuthenticationError(message string, httpCode int, errorCode string, responseData ...map[string]any) *AuthenticationError {
	var data map[string]any
	if len(responseData) > 0 {
		data = responseData[0]
	}
	return &AuthenticationError{
		CoCartError: CoCartError{
			Message:      message,
			HTTPCode:     httpCode,
			ErrorCode:    errorCode,
			ResponseData: data,
		},
	}
}

// ValidationError represents a 400 validation failure.
type ValidationError struct {
	CoCartError
}

// Unwrap returns the underlying CoCartError for errors.As compatibility.
func (e *ValidationError) Unwrap() error {
	return &e.CoCartError
}

// NewValidationError creates a new ValidationError.
func NewValidationError(message string, httpCode int, errorCode string, responseData ...map[string]any) *ValidationError {
	var data map[string]any
	if len(responseData) > 0 {
		data = responseData[0]
	}
	return &ValidationError{
		CoCartError: CoCartError{
			Message:      message,
			HTTPCode:     httpCode,
			ErrorCode:    errorCode,
			ResponseData: data,
		},
	}
}

// TwoFactorRequiredError is returned when the server requires a 2FA code to complete login.
//
// Returned by [JWTManager.Login] when the CoCart 2FA plugin is installed and the user
// has 2FA enabled. Catch this error, inspect AvailableProviders and DefaultProvider,
// prompt the user for a code, then call [JWTManager.LoginWith2FA] to complete login.
type TwoFactorRequiredError struct {
	AuthenticationError
	// AvailableProviders lists the providers the user can use (e.g. "totp", "email", "backup").
	AvailableProviders []string
	// DefaultProvider is the provider the server will use if none is specified.
	DefaultProvider string
	// EmailSent indicates whether the server has already sent a code via email.
	EmailSent bool
}

// Unwrap returns the embedded AuthenticationError for errors.As compatibility.
func (e *TwoFactorRequiredError) Unwrap() error { return &e.AuthenticationError }

// NewTwoFactorRequiredError creates a TwoFactorRequiredError from the API response data.
func NewTwoFactorRequiredError(message string, data map[string]any) *TwoFactorRequiredError {
	inner := data
	if d, ok := data["data"].(map[string]any); ok {
		inner = d
	}

	providers, _ := inner["available_providers"].([]any)
	strProviders := make([]string, 0, len(providers))
	for _, p := range providers {
		if s, ok := p.(string); ok {
			strProviders = append(strProviders, s)
		}
	}

	defaultProvider, _ := inner["default_provider"].(string)
	emailSent, _ := inner["email_sent"].(bool)

	return &TwoFactorRequiredError{
		AuthenticationError: AuthenticationError{
			CoCartError: CoCartError{
				Message:      message,
				HTTPCode:     401,
				ErrorCode:    "cocart_2fa_required",
				ResponseData: data,
			},
		},
		AvailableProviders: strProviders,
		DefaultProvider:    defaultProvider,
		EmailSent:          emailSent,
	}
}

// VersionError represents a feature requiring CoCart Basic.
type VersionError struct {
	CoCartError
	Method string
}

// Unwrap returns the underlying CoCartError for errors.As compatibility.
func (e *VersionError) Unwrap() error {
	return &e.CoCartError
}

// NewVersionError creates a new VersionError for a method that requires CoCart Basic.
func NewVersionError(method string) *VersionError {
	return &VersionError{
		CoCartError: CoCartError{
			Message:   fmt.Sprintf("%s() requires CoCart Basic. Please upgrade from the legacy CoCart plugin.", method),
			HTTPCode:  0,
			ErrorCode: "cocart_version_required",
		},
		Method: method,
	}
}
