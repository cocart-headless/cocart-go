package cocart

import (
	"errors"
	"testing"
)

func TestValidateProductID(t *testing.T) {
	tests := []struct {
		id      int
		wantErr bool
	}{
		{1, false},
		{42, false},
		{0, true},
		{-1, true},
	}

	for _, tt := range tests {
		err := ValidateProductID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateProductID(%d) err=%v, wantErr=%v", tt.id, err, tt.wantErr)
		}
		if err != nil {
			var valErr *ValidationError
			if !errors.As(err, &valErr) {
				t.Errorf("expected ValidationError for id=%d", tt.id)
			}
		}
	}
}

func TestValidateQuantity(t *testing.T) {
	tests := []struct {
		qty     int
		wantErr bool
	}{
		{1, false},
		{100, false},
		{0, true},
		{-5, true},
	}

	for _, tt := range tests {
		err := ValidateQuantity(tt.qty)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateQuantity(%d) err=%v, wantErr=%v", tt.qty, err, tt.wantErr)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"a@b.c", false},
		{"", true},
		{"invalid", true},
		{"@example.com", true},
		{"user@", true},
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.email)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateEmail(%q) err=%v, wantErr=%v", tt.email, err, tt.wantErr)
		}
	}
}
