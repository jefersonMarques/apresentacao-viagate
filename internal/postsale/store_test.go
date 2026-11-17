package postsale

import "testing"

func TestValidateContact(t *testing.T) {
	tests := []struct {
		name string
		contact Contact
		valid bool
	}{
		{"self", Contact{Mode: "self", Name: "Signer", Phone: "41999999999"}, true},
		{"team without email", Contact{Mode: "team", Name: "Operator", Phone: "41999999999"}, true},
		{"missing mode", Contact{Name: "Operator", Phone: "41999999999"}, false},
		{"missing name", Contact{Mode: "team", Phone: "41999999999"}, false},
		{"missing phone", Contact{Mode: "team", Name: "Operator"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateContact(test.contact)
			if (err == nil) != test.valid {
				t.Errorf("ValidateContact error = %v, expected valid=%v", err, test.valid)
			}
		})
	}
}
