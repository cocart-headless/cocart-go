package cocart

import "testing"

func TestCurrencyFormatterFormat(t *testing.T) {
	f := NewCurrencyFormatter()

	tests := []struct {
		name     string
		amount   int
		info     CurrencyInfo
		expected string
	}{
		{
			"USD basic",
			4599,
			CurrencyInfo{
				CurrencyCode:        "USD",
				CurrencySymbol:      "$",
				CurrencyMinorUnit:   2,
				CurrencyDecimalSep:  ".",
				CurrencyThousandSep: ",",
				CurrencyPrefix:      "$",
				CurrencySuffix:      "",
			},
			"$45.99",
		},
		{
			"EUR with comma decimal",
			1250,
			CurrencyInfo{
				CurrencyCode:        "EUR",
				CurrencySymbol:      "€",
				CurrencyMinorUnit:   2,
				CurrencyDecimalSep:  ",",
				CurrencyThousandSep: ".",
				CurrencyPrefix:      "",
				CurrencySuffix:      " €",
			},
			"12,50 €",
		},
		{
			"large number with thousands",
			123456789,
			CurrencyInfo{
				CurrencyCode:        "USD",
				CurrencySymbol:      "$",
				CurrencyMinorUnit:   2,
				CurrencyDecimalSep:  ".",
				CurrencyThousandSep: ",",
				CurrencyPrefix:      "$",
				CurrencySuffix:      "",
			},
			"$1,234,567.89",
		},
		{
			"zero minor unit (JPY)",
			1500,
			CurrencyInfo{
				CurrencyCode:        "JPY",
				CurrencySymbol:      "¥",
				CurrencyMinorUnit:   0,
				CurrencyDecimalSep:  "",
				CurrencyThousandSep: ",",
				CurrencyPrefix:      "¥",
				CurrencySuffix:      "",
			},
			"¥1,500",
		},
		{
			"zero amount",
			0,
			CurrencyInfo{
				CurrencyCode:      "USD",
				CurrencyMinorUnit: 2,
				CurrencyPrefix:    "$",
			},
			"$0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Format(tt.amount, tt.info)
			if result != tt.expected {
				t.Errorf("Format(%d) = %q, want %q", tt.amount, result, tt.expected)
			}
		})
	}
}

func TestCurrencyFormatterFormatDecimal(t *testing.T) {
	f := NewCurrencyFormatter()
	info := CurrencyInfo{CurrencyMinorUnit: 2}

	tests := []struct {
		amount   int
		expected string
	}{
		{4599, "45.99"},
		{100, "1.00"},
		{0, "0.00"},
		{1, "0.01"},
	}

	for _, tt := range tests {
		result := f.FormatDecimal(tt.amount, info)
		if result != tt.expected {
			t.Errorf("FormatDecimal(%d) = %q, want %q", tt.amount, result, tt.expected)
		}
	}
}
