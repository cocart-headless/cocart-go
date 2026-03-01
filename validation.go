package cocart

import "regexp"

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// ValidateProductID validates that a product ID is a positive integer.
func ValidateProductID(id int) error {
	if id < 1 {
		return NewValidationError(
			"Product ID must be a positive integer",
			0,
			"cocart_invalid_product_id",
		)
	}
	return nil
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
