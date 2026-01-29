package config

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in       string
		wantM    int
		wantN    int
		wantP    int
		wantErr  bool
	}{
		{"1", 1, 0, 0, false},
		{"1.2", 1, 2, 0, false},
		{"1.2.3", 1, 2, 3, false},
		{"0.0.0", 0, 0, 0, false},
		{" 1 . 2 . 3 ", 1, 2, 3, false},
		{"", 0, 0, 0, true},
		{"a", 0, 0, 0, true},
		{"1.a", 0, 0, 0, true},
		{"1.2.3.4", 0, 0, 0, true},
	}
	for _, tt := range tests {
		m, n, p, err := ParseVersion(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVersion(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (m != tt.wantM || n != tt.wantN || p != tt.wantP) {
			t.Errorf("ParseVersion(%q) = %d,%d,%d want %d,%d,%d", tt.in, m, n, p, tt.wantM, tt.wantN, tt.wantP)
		}
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"1", "1.0.0", false},
		{"1.2", "1.2.0", false},
		{"1.2.3", "1.2.3", false},
		{"0.5.0", "0.5.0", false},
		{"", "", true},
		{"x", "", true},
	}
	for _, tt := range tests {
		got, err := Normalize(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("Normalize(%q) err=%v wantErr=%v", tt.in, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("Normalize(%q) = %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "2.0.0", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.0.0", false},
		{"2.0.0", "1.0.0", false},
		{"1", "2", true},
		{"1.2", "1.3", true},
		{"0.5.0", "1.0.0", true},
		{"invalid", "1.0.0", false},
		{"1.0.0", "invalid", false},
	}
	for _, tt := range tests {
		got := Less(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("Less(%q, %q) = %v want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.0", true},
		{"1", "1.0.0", true},
		{"1.2", "1.2.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.0.0", "1.0.1", false},
		{"invalid", "1.0.0", false},
		{"1.0.0", "invalid", false},
	}
	for _, tt := range tests {
		got := Equal(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("Equal(%q, %q) = %v want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
