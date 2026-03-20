package service

import "testing"

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "/"},
		{"/", "/"},
		{"photos", "/photos"},
		{"/photos/", "/photos"},
		{"/photos//cats", "/photos/cats"},
		{"//photos/../docs", "/docs"},
		{"/photos/./cats", "/photos/cats"},
	}

	for _, tt := range tests {
		got := NormalizePath(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"/photos", false},
		{"/photos/cats/a.jpg", false},
		{"/", false},
		{"", true},          // empty
		{"photos", true},    // no leading /
		{"/photos/..", true}, // contains ..
		{"/pho\\tos", true}, // backslash
	}

	for _, tt := range tests {
		err := ValidatePath(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}
