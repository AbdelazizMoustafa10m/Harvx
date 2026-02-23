package security

import (
	"strings"
	"testing"
)

// BenchmarkCalculate measures the entropy calculation performance for strings
// of varying lengths. The target is under 1 microsecond for all sizes up to
// 256 characters (the MaxLength clamp).
//
// The sub-benchmark form (BenchmarkCalculate/short_16 etc.) matches the naming
// convention in the task spec and allows selective execution with -bench flags.

func BenchmarkCalculate_Short16(b *testing.B) {
	a := NewEntropyAnalyzer()
	s := "abcdefghijklmnop" // exactly 16 chars, diverse
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Calculate(s)
	}
}

func BenchmarkCalculate_Medium64(b *testing.B) {
	a := NewEntropyAnalyzer()
	// 64-char high-diversity alphanumeric string
	s := strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGH0123456789", 2)[:64]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Calculate(s)
	}
}

func BenchmarkCalculate_Long256(b *testing.B) {
	a := NewEntropyAnalyzer()
	// 256-char string (hits MaxLength cap exactly)
	s := strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/", 4)[:256]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Calculate(s)
	}
}

func BenchmarkCalculate_VeryLong10K(b *testing.B) {
	a := NewEntropyAnalyzer()
	// 10 000-char string -- Calculate should clamp to MaxLength (256) internally
	s := strings.Repeat("abcdefghijklmnopqrstuvwxyz0123456789", 300)[:10000]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Calculate(s)
	}
}

// BenchmarkCalculate exercises the sub-benchmark naming pattern required by
// the task spec: BenchmarkCalculate/short_16, /medium_64, /long_256.
func BenchmarkCalculate(b *testing.B) {
	a := NewEntropyAnalyzer()

	cases := []struct {
		name string
		s    string
	}{
		{
			name: "short_16",
			s:    "abcdefghijklmnop",
		},
		{
			name: "medium_64",
			s:    strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGH0123456789", 2)[:64],
		},
		{
			name: "long_256",
			s:    strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/", 4)[:256],
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = a.Calculate(tc.s)
			}
		})
	}
}

func BenchmarkDetectCharset(b *testing.B) {
	s := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectCharset(s)
	}
}

func BenchmarkAnalyzeToken(b *testing.B) {
	a := NewEntropyAnalyzer()
	token := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	ctx := TokenContext{
		VariableName: "AWS_SECRET_ACCESS_KEY",
		LineContent:  `AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.AnalyzeToken(token, ctx)
	}
}
