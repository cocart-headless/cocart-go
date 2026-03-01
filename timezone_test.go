package cocart

import "testing"

func TestTimezoneHelperDetect(t *testing.T) {
	tz := NewTimezoneHelper()
	zone := tz.DetectTimezone()
	if zone == "" {
		t.Error("DetectTimezone() returned empty string")
	}
}

func TestTimezoneHelperConvert(t *testing.T) {
	tz := NewTimezoneHelper()

	// UTC to America/New_York (EST = UTC-5)
	result, err := tz.Convert("2025-01-15T15:00:00", "UTC", "America/New_York")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if result != "2025-01-15T10:00:00" {
		t.Errorf("Convert result = %s, want 2025-01-15T10:00:00", result)
	}
}

func TestTimezoneHelperConvertInvalidTimezone(t *testing.T) {
	tz := NewTimezoneHelper()
	_, err := tz.Convert("2025-01-15T10:00:00", "Invalid/Zone", "UTC")
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}

func TestTimezoneHelperConvertWithTimezoneIndicator(t *testing.T) {
	tz := NewTimezoneHelper()

	// Z suffix (UTC)
	result, err := tz.Convert("2025-01-15T15:00:00Z", "UTC", "America/New_York")
	if err != nil {
		t.Fatalf("Convert error: %v", err)
	}
	if result != "2025-01-15T10:00:00" {
		t.Errorf("Convert result = %s, want 2025-01-15T10:00:00", result)
	}
}
