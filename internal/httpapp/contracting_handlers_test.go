package httpapp

import "testing"

func TestNormalizeOperationType(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantValid bool
	}{
		{name: "normal", input: "normal", want: "normal", wantValid: true},
		{name: "normal with spaces and case", input: " Normal ", want: "normal", wantValid: true},
		{name: "avulsa", input: "avulsa", want: "avulsa", wantValid: true},
		{name: "avulsa with spaces and case", input: " AVULSA ", want: "avulsa", wantValid: true},
		{name: "empty", input: "", want: "", wantValid: false},
		{name: "unknown", input: "expressa", want: "expressa", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := normalizeOperationType(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeOperationType(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if valid != tt.wantValid {
				t.Fatalf("normalizeOperationType(%q) valid = %v, want %v", tt.input, valid, tt.wantValid)
			}
		})
	}
}
