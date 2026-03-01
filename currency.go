package cocart

import (
	"fmt"
	"math"
	"strings"
)

// CurrencyFormatter formats prices from smallest-unit integers to human-readable strings.
type CurrencyFormatter struct{}

// NewCurrencyFormatter creates a new CurrencyFormatter.
func NewCurrencyFormatter() *CurrencyFormatter {
	return &CurrencyFormatter{}
}

// Format converts a smallest-unit integer to a formatted currency string.
//
// Example: Format(4599, currencyInfo) -> "$45.99"
func (f *CurrencyFormatter) Format(amount int, info CurrencyInfo) string {
	decimal := f.FormatDecimal(amount, info)

	// Insert thousand separators
	if info.CurrencyThousandSep != "" {
		decimal = insertThousandSep(decimal, info.CurrencyThousandSep, info.CurrencyDecimalSep)
	}

	// Replace default decimal separator
	if info.CurrencyDecimalSep != "" && info.CurrencyDecimalSep != "." {
		decimal = replaceDecimalSep(decimal, info.CurrencyDecimalSep)
	}

	return info.CurrencyPrefix + decimal + info.CurrencySuffix
}

// FormatDecimal converts a smallest-unit integer to a plain decimal string (no currency symbol).
//
// Example: FormatDecimal(4599, currencyInfo) -> "45.99"
func (f *CurrencyFormatter) FormatDecimal(amount int, info CurrencyInfo) string {
	divisor := math.Pow(10, float64(info.CurrencyMinorUnit))
	value := float64(amount) / divisor
	return fmt.Sprintf("%.*f", info.CurrencyMinorUnit, value)
}

// insertThousandSep inserts thousand separators into the integer part of a decimal string.
func insertThousandSep(decimal, thousandSep, decimalSep string) string {
	sep := "."
	if decimalSep != "" {
		sep = decimalSep
	}

	parts := strings.SplitN(decimal, ".", 2)
	intPart := parts[0]

	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	if len(intPart) <= 3 {
		result := decimal
		if negative {
			result = "-" + result
		}
		return result
	}

	var groups []string
	for len(intPart) > 3 {
		groups = append([]string{intPart[len(intPart)-3:]}, groups...)
		intPart = intPart[:len(intPart)-3]
	}
	groups = append([]string{intPart}, groups...)

	formatted := strings.Join(groups, thousandSep)
	if negative {
		formatted = "-" + formatted
	}

	if len(parts) > 1 {
		formatted += sep + parts[1]
	}

	return formatted
}

// replaceDecimalSep replaces the decimal separator in a decimal string.
func replaceDecimalSep(decimal, newSep string) string {
	return strings.Replace(decimal, ".", newSep, 1)
}
