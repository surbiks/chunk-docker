package config

import "testing"

func TestParseByteSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int64
	}{
		{input: "20MB", want: 20_000_000},
		{input: "32MiB", want: 32 * 1024 * 1024},
		{input: "50 MB", want: 50_000_000},
		{input: "1GiB", want: 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, err := ParseByteSize(tt.input)
			if err != nil {
				t.Fatalf("ParseByteSize(%q) returned error: %v", tt.input, err)
			}
			if got.Int64() != tt.want {
				t.Fatalf("ParseByteSize(%q) = %d, want %d", tt.input, got.Int64(), tt.want)
			}
		})
	}
}

func TestParseByteSizeRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	invalid := []string{"", "abc", "0", "-1MB", "5XB"}
	for _, input := range invalid {
		if _, err := ParseByteSize(input); err == nil {
			t.Fatalf("expected ParseByteSize(%q) to fail", input)
		}
	}
}
