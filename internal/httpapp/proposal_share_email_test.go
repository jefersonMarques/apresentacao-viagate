package httpapp

import (
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func TestBuildProposalShareEmailIncludesProposalAndSellerSignature(t *testing.T) {
	input := proposals.EditorInput{
		ClientTradeName: "Cliente XPTO",
		ContactName:     "Mariana",
	}
	seller := domain.User{
		Name:     "Jeferson Marques",
		Email:    "jeferson@example.com",
		Phone:    "(41) 99999-0000",
		JobTitle: "Executivo Comercial",
		PhotoURL: "/media/photo-id",
	}

	subject, htmlBody, textBody := buildProposalShareEmail(
		"https://viagate.com.br/",
		input,
		seller,
		"public-token",
	)

	if subject != "Proposta ViaGate — Cliente XPTO" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	for _, expected := range []string{
		"https://viagate.com.br/p/public-token",
		"Jeferson Marques",
		"Executivo Comercial",
		"jeferson@example.com",
		"https://viagate.com.br/media/photo-id",
	} {
		if !strings.Contains(htmlBody, expected) {
			t.Fatalf("html body should contain %q", expected)
		}
	}
	if !strings.Contains(textBody, "Mariana") || !strings.Contains(textBody, "Jeferson Marques") {
		t.Fatal("text body must contain recipient greeting and seller signature")
	}
}

func TestBuildProposalShareEmailEscapesCustomerData(t *testing.T) {
	input := proposals.EditorInput{
		ClientTradeName: "<script>alert(1)</script>",
		ContactName:     "<b>Contato</b>",
	}
	_, htmlBody, _ := buildProposalShareEmail(
		"https://viagate.com.br",
		input,
		domain.User{Name: "<img src=x onerror=alert(1)>"},
		"token",
	)

	for _, unsafe := range []string{"<script>alert(1)</script>", "<b>Contato</b>", "<img src=x onerror=alert(1)>"} {
		if strings.Contains(htmlBody, unsafe) {
			t.Fatalf("html body contains unescaped value %q", unsafe)
		}
	}
}
