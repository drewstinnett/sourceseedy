package update

import "testing"

func TestNewer(t *testing.T) {
	for _, tt := range []struct {
		current, latest string
		want            bool
	}{
		{"v0.2.6", "v0.3.0", true},
		{"v0.2.6", "v0.2.7", true},
		{"v0.2.6", "v1.0.0", true},
		{"v0.2.9", "v0.2.10", true}, // numbers, not text
		{"v0.3.0", "v0.3.0", false},
		{"v0.3.0", "v0.2.6", false},
		{"v1.0.0", "v0.9.9", false},
		{"0.2.6", "v0.3.0", true}, // v is optional
		{"dev", "v0.3.0", false},
		{"", "v0.3.0", false},
		{"v0.2.6-dirty", "v0.3.0", false},
		{"v0.2.6-3-gabcdef", "v0.3.0", false},
		{"v0.2.6-snapshot", "v0.3.0", false},
		{"v0.2.6", "latest", false},
		{"v0.2.6", "", false},
		{"v0.2", "v0.3.0", false},
		{"v0.2.6.1", "v0.3.0", false},
		{"v0.-2.6", "v0.3.0", false},
		{"v0.+2.6", "v0.3.0", false},
	} {
		if got := Newer(tt.current, tt.latest); got != tt.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestIsRelease(t *testing.T) {
	for v, want := range map[string]bool{
		"v0.3.0":           true,
		"v10.20.30":        true,
		"dev":              false,
		"":                 false,
		"v0.3.0-dirty":     false,
		"v0.3.0-1-gabcdef": false,
	} {
		if got := IsRelease(v); got != want {
			t.Errorf("IsRelease(%q) = %v, want %v", v, got, want)
		}
	}
}
