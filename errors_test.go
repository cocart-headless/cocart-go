package cocart

import (
	"errors"
	"testing"
)

func TestCoCartError(t *testing.T) {
	err := NewCoCartError("test error", 500, "test_code")
	if err.Error() != "test error" {
		t.Errorf("Error() = %s", err.Error())
	}
	if err.HTTPCode != 500 {
		t.Errorf("HTTPCode = %d", err.HTTPCode)
	}
	if err.ErrorCode != "test_code" {
		t.Errorf("ErrorCode = %s", err.ErrorCode)
	}
}

func TestAuthenticationError(t *testing.T) {
	err := NewAuthenticationError("auth failed", 401, "rest_forbidden")
	if err.Error() != "auth failed" {
		t.Errorf("Error() = %s", err.Error())
	}

	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Error("should be AuthenticationError")
	}

	var cocartErr *CoCartError
	if !errors.As(err, &cocartErr) {
		t.Error("should also be CoCartError")
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("invalid input", 400, "cocart_invalid")

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Error("should be ValidationError")
	}
}

func TestVersionError(t *testing.T) {
	err := NewVersionError("cart()->create")
	if err.Method != "cart()->create" {
		t.Errorf("Method = %s", err.Method)
	}
	if err.ErrorCode != "cocart_version_required" {
		t.Errorf("ErrorCode = %s", err.ErrorCode)
	}
	expected := "cart()->create() requires CoCart Starter. Please upgrade from the CoCart Community plugin."
	if err.Error() != expected {
		t.Errorf("Error() = %s", err.Error())
	}
}

func TestErrKeyNotFound(t *testing.T) {
	if !errors.Is(ErrKeyNotFound, ErrKeyNotFound) {
		t.Error("ErrKeyNotFound should be itself")
	}
}
