package templates

import "testing"

func TestTeamContactValue(t *testing.T) {
	if actual := TeamContactValue("self", "Signer"); actual != "" {
		t.Fatalf("self-mode contact should not prefill team fields, got %q", actual)
	}
	if actual := TeamContactValue("team", "Operator"); actual != "Operator" {
		t.Fatalf("team contact should be retained, got %q", actual)
	}
}
