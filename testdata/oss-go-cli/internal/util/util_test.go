package util

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr bool
	}{
		{name: "bytes", input: "100B", want: 100},
		{name: "kilobytes", input: "10KB", want: 10240},
		{name: "megabytes", input: "50MB", want: 52428800},
		{name: "gigabytes", input: "1GB", want: 1073741824},
		{name: "lowercase", input: "10mb", want: 10485760},
		{name: "with spaces", input: " 5MB ", want: 5242880},
		{name: "empty", input: "", wantErr: true},
		{name: "no unit", input: "100", wantErr: true},
		{name: "invalid number", input: "abcMB", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "bytes", bytes: 500, want: "500B"},
		{name: "kilobytes", bytes: 2048, want: "2.0KB"},
		{name: "megabytes", bytes: 5242880, want: "5.0MB"},
		{name: "gigabytes", bytes: 1073741824, want: "1.0GB"},
		{name: "zero", bytes: 0, want: "0B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  int
	}{
		{name: "no duplicates", input: []string{"a", "b", "c"}, want: 3},
		{name: "with duplicates", input: []string{"a", "b", "a", "c", "b"}, want: 3},
		{name: "all same", input: []string{"x", "x", "x"}, want: 1},
		{name: "empty", input: []string{}, want: 0},
		{name: "nil", input: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Unique(tt.input)
			if len(got) != tt.want {
				t.Errorf("Unique(%v) has %d elements, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}