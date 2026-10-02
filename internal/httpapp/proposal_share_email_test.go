package httpapp

import (
	"testing"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func TestProposalEmailVariablesUseCanonicalBaseURL(t *testing.T) {
	validUntil := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	app := &App{cfg: config.Config{BaseURL: "https://viagate.com.br/"}}
	input := proposals.EditorInput{
		ClientLegalName:  "Cliente XPTO Ltda.",
		ClientTradeName:  "Cliente XPTO",
		ContactName:      "Mariana",
		ContactEmail:     "mariana@example.com",
		Title:            "Proposta Comercial",
		ValidUntil:       &validUntil,
	}
	seller := domain.User{
		Name:     "Jeferson Marques",
		Email:    "jeferson@example.com",
		Phone:    "(41) 99999-0000",
		JobTitle: "Executivo Comercial",
		PhotoURL: "/media/photo-id",
	}

	variables := app.proposalEmailVariables(input, seller, "public-token")

	if variables["proposal.url"] != "https://viagate.com.br/p/public-token" {
		t.Fatalf("unexpected proposal URL: %q", variables["proposal.url"])
	}
	if variables["salesperson.photo_url"] != "https://viagate.com.br/media/photo-id" {
		t.Fatalf("unexpected seller photo URL: %q", variables["salesperson.photo_url"])
	}
	if variables["contact.greeting"] != "Olá, Mariana." {
		t.Fatalf("unexpected greeting: %q", variables["contact.greeting"])
	}
	if variables["proposal.valid_until"] != "31/12/2026" {
		t.Fatalf("unexpected validity: %q", variables["proposal.valid_until"])
	}
}

func TestProposalEmailVariablesFallbackGreeting(t *testing.T) {
	app := &App{cfg: config.Config{BaseURL: "https://viagate.com.br"}}
	variables := app.proposalEmailVariables(
		proposals.EditorInput{ClientLegalName: "Cliente"},
		domain.User{},
		"token",
	)
	if variables["contact.greeting"] != "Olá." {
		t.Fatalf("unexpected greeting: %q", variables["contact.greeting"])
	}
}
