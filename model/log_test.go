package model

import "testing"

func TestNormalizeClientIPCountryCode(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "upper alpha2", raw: "US", want: "US"},
		{name: "lower alpha2", raw: "cn", want: "CN"},
		{name: "trim spaces", raw: " jp ", want: "JP"},
		{name: "empty", raw: "", want: ""},
		{name: "too long", raw: "USA", want: ""},
		{name: "contains digit", raw: "C1", want: ""},
		{name: "contains symbol", raw: "U-", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeClientIPCountryCode(tt.raw); got != tt.want {
				t.Fatalf("normalizeClientIPCountryCode(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
