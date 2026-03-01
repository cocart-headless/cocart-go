package cocart

import (
	"strings"
	"time"
)

// TimezoneHelper provides timezone conversion utilities for WooCommerce date strings.
type TimezoneHelper struct{}

// NewTimezoneHelper creates a new TimezoneHelper.
func NewTimezoneHelper() *TimezoneHelper {
	return &TimezoneHelper{}
}

// DetectTimezone returns the system's IANA timezone string.
func (t *TimezoneHelper) DetectTimezone() string {
	zone, _ := time.Now().Zone()
	loc := time.Now().Location()
	if loc != nil && loc.String() != "Local" {
		return loc.String()
	}
	return zone
}

// Convert converts an ISO date string from one timezone to another.
func (t *TimezoneHelper) Convert(dateString, fromTZ, toTZ string) (string, error) {
	fromLoc, err := time.LoadLocation(fromTZ)
	if err != nil {
		return "", err
	}
	toLoc, err := time.LoadLocation(toTZ)
	if err != nil {
		return "", err
	}

	parsed, err := parseDate(dateString, fromLoc)
	if err != nil {
		return "", err
	}

	converted := parsed.In(toLoc)
	return converted.Format("2006-01-02T15:04:05"), nil
}

// ToLocal converts a date string to the local system timezone.
func (t *TimezoneHelper) ToLocal(dateString string, storeTZ ...string) (string, error) {
	fromTZ := "UTC"
	if len(storeTZ) > 0 && storeTZ[0] != "" {
		fromTZ = storeTZ[0]
	}
	return t.Convert(dateString, fromTZ, t.DetectTimezone())
}

// parseDate parses a date string in various ISO 8601 formats.
func parseDate(dateString string, loc *time.Location) (time.Time, error) {
	// Try formats from most specific to least
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	// If the string has a timezone indicator, parse directly
	if strings.Contains(dateString, "Z") || strings.Contains(dateString, "+") {
		for _, format := range formats {
			if t, err := time.Parse(format, dateString); err == nil {
				return t, nil
			}
		}
	}

	// Otherwise, parse in the specified location
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, dateString, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &CoCartError{
		Message:   "Failed to parse date: " + dateString,
		ErrorCode: "cocart_invalid_date",
	}
}
