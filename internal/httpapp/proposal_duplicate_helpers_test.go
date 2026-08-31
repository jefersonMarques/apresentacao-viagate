package httpapp

import "testing"

func TestScrubDuplicatedProposalTextRemovesKnownClientData(t *testing.T) {
	value := "Solução exclusiva para ACME Transportes — contato Maria Silva"
	cleaned := scrubDuplicatedProposalText(value, []string{"ACME Transportes", "Maria Silva"})
	if cleaned != "Solução exclusiva para — contato" {
		t.Fatalf("unexpected scrubbed text: %q", cleaned)
	}
}

func TestScrubDuplicatedProposalTextIsCaseInsensitive(t *testing.T) {
	value := "Condição negociada com Cliente Exemplo"
	cleaned := scrubDuplicatedProposalText(value, []string{"cliente exemplo"})
	if cleaned != "Condição negociada com" {
		t.Fatalf("unexpected scrubbed text: %q", cleaned)
	}
}
