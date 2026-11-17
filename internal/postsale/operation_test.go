package postsale

import "testing"

func TestOperationComplete(t *testing.T) {
	operation := Operation{
		Type: "normal",
		Insurer: "Example Insurer",
		StartDate: "2026-11-01",
		EndDate: "2027-11-01",
	}
	if !OperationComplete(operation) {
		t.Fatal("operation should be complete")
	}
	operation.Insurer = ""
	if OperationComplete(operation) {
		t.Fatal("operation must require an insurer")
	}
}

func TestValidateOperation(t *testing.T) {
	cases := []struct {
		name string
		operation Operation
		valid bool
	}{
		{"empty draft",Operation{},true},
		{"normal",Operation{Type:"normal",StartDate:"2026-11-01",EndDate:"2027-11-01"},true},
		{"invalid type",Operation{Type:"invalid"},false},
		{"invalid date",Operation{StartDate:"2026-13-01"},false},
		{"reversed period",Operation{StartDate:"2027-11-01",EndDate:"2026-11-01"},false},
	}
	for _,test := range cases {
		t.Run(test.name,func(t *testing.T) {
			valid := ValidateOperation(test.operation)==nil
			if valid!=test.valid {
				t.Errorf("valid=%v, want %v",valid,test.valid)
			}
		})
	}
}
