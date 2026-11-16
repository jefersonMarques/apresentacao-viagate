package httpapp

import "testing"

func TestProposalSignerPrefillUsesOnlyExplicitSigner(t *testing.T) {
	content := map[string]any{
		"contact": map[string]any{
			"name": "Negotiation Contact",
			"email": "negotiation@example.com",
		},
		"signer": map[string]any{
			"name": "Legal Signer",
			"cpf": "12345678909",
			"role": "Director",
			"email": "signer@example.com",
			"phone": "41999999999",
		},
	}

	actual := proposalSignerPrefill(content)
	if actual["name"] != "Legal Signer" || actual["cpf"] != "12345678909" {
		t.Fatalf("wrong signer identification: %v", actual)
	}
	if actual["email"] != "signer@example.com" || actual["phone"] != "41999999999" {
		t.Fatalf("wrong signer contact: %v", actual)
	}
}

func TestProposalSignerPrefillDoesNotUseNegotiationContact(t *testing.T) {
	actual := proposalSignerPrefill(map[string]any{
		"contact": map[string]any{"name": "Different Contact", "email": "contact@example.com"},
	})
	for field, value := range actual {
		if value != "" {
			t.Errorf("expected empty %s for proposals without a signer, got %q", field, value)
		}
	}
}
