package httpapp

import "testing"

func TestProposalDeleteConfirmed(t *testing.T) {
	cases := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "exact", value: "Quero excluir", valid: true},
		{name: "outer whitespace", value: "  Quero excluir  ", valid: true},
		{name: "wrong case", value: "quero excluir", valid: false},
		{name: "extra text", value: "Quero excluir agora", valid: false},
		{name: "empty", value: "", valid: false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := proposalDeleteConfirmed(test.value); got != test.valid {
				t.Fatalf("proposalDeleteConfirmed(%q) = %v, expected %v", test.value, got, test.valid)
			}
		})
	}
}
