package cocart

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
var numericStringRegex = regexp.MustCompile(`^\s*-?\d+(\.\d+)?\s*$`)

// invalidProductIDError is returned whenever a product ID is certain to be
// invalid (empty, or numeric but not a positive integer).
func invalidProductIDError() error {
	return NewValidationError(
		"Product ID must be a positive integer",
		0,
		"cocart_invalid_product_id",
	)
}

// ValidateProductID validates a product ID, mirroring the server's own resolution rules.
//
// A numeric value (int, or a string containing only a number) must be a
// positive integer. A non-numeric string is treated as a potential SKU and
// passed through untouched — the server resolves a non-numeric ID before falling back to a 404.
// This SDK can't verify a SKU exists without a network request, so it only rejects input
// that's certain to be invalid — empty, or numeric but not a positive integer — client-side.
func ValidateProductID(id any) error {
	switch v := id.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return invalidProductIDError()
		}
		if !numericStringRegex.MatchString(v) {
			return nil // Non-numeric string — treat as a SKU; the server resolves it.
		}
		f, err := strconv.ParseFloat(trimmed, 64)
		if err != nil || f < 1 || f != math.Trunc(f) {
			return invalidProductIDError()
		}
		return nil
	case int:
		if v < 1 {
			return invalidProductIDError()
		}
		return nil
	case int64:
		if v < 1 {
			return invalidProductIDError()
		}
		return nil
	default:
		return invalidProductIDError()
	}
}

// ValidateQuantity validates that a quantity is a positive number.
func ValidateQuantity(quantity int) error {
	if quantity < 1 {
		return NewValidationError(
			"Quantity must be a positive number",
			0,
			"cocart_invalid_quantity",
		)
	}
	return nil
}

// ValidateEmail validates that an email address has a valid basic format.
func ValidateEmail(email string) error {
	if email == "" || !emailRegex.MatchString(email) {
		return NewValidationError(
			"A valid email address is required",
			0,
			"cocart_invalid_email",
		)
	}
	return nil
}
