package httpapp

import (
	"strings"
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


func TestProposalEmailProductVariablesRenderPublishedSnapshotSafely(t *testing.T) {
	variables := proposalEmailProductVariables([]proposalEmailProduct{
		{
			Label:       "Cargo <Score>",
			Description: "Análise & risco",
			GroupName:   "Plataforma Cargo",
		},
		{
			Label:       "Cargo Logística",
			Description: "Acompanhamento por aplicativo",
			GroupName:   "Plataforma Cargo",
			IsOptional:  true,
		},
	}, true)

	htmlBody := variables["proposal.products_html"]
	if strings.Contains(htmlBody, "Cargo <Score>") {
		t.Fatal("product label must be escaped in trusted HTML block")
	}
	if !strings.Contains(htmlBody, "Cargo &lt;Score&gt;") || !strings.Contains(htmlBody, "Análise &amp; risco") {
		t.Fatal("escaped product snapshot values missing from HTML block")
	}
	if !strings.Contains(htmlBody, "Opcional") {
		t.Fatal("optional proposal item should be identified in HTML block")
	}
	if variables["proposal.products_count"] != "2" {
		t.Fatalf("unexpected product count: %q", variables["proposal.products_count"])
	}
	if variables["proposal.requires_policy"] != "Sim" {
		t.Fatalf("unexpected policy requirement: %q", variables["proposal.requires_policy"])
	}
	if !strings.Contains(variables["proposal.products_text"], "Cargo Logística (Opcional)") {
		t.Fatal("plain text product list should identify optional proposal items")
	}
}

func TestProposalEmailProductVariablesWithoutPolicy(t *testing.T) {
	variables := proposalEmailProductVariables([]proposalEmailProduct{
		{Label: "Cadastro de motorista"},
	}, false)

	if variables["proposal.requires_policy"] != "Não" {
		t.Fatalf("unexpected policy requirement: %q", variables["proposal.requires_policy"])
	}
	if variables["proposal.products_count"] != "1" {
		t.Fatalf("unexpected product count: %q", variables["proposal.products_count"])
	}
}


func TestProposalEmailInputUsesPublishedSnapshot(t *testing.T) {
	input := proposals.EditorInput{
		Title:           "Rascunho novo",
		ClientLegalName: "Cliente alterado",
		ClientTradeName: "Cliente alterado",
		ContactName:     "Contato alterado",
		ContactEmail:    "novo@example.com",
	}
	published := map[string]any{
		"proposal": map[string]any{
			"title":       "Proposta publicada",
			"valid_until": "2026-12-31",
		},
		"client": map[string]any{
			"legal_name": "Cliente Publicado Ltda.",
			"trade_name": "Cliente Publicado",
			"email":      "cliente@example.com",
		},
		"contact": map[string]any{
			"name":  "Mariana",
			"role":  "Gerente",
			"email": "mariana@example.com",
			"phone": "41999990000",
		},
	}

	result := proposalEmailInputFromPublishedContent(input, published)

	if result.Title != "Proposta publicada" {
		t.Fatalf("email should use published title, got %q", result.Title)
	}
	if result.ClientTradeName != "Cliente Publicado" {
		t.Fatalf("email should use published client snapshot, got %q", result.ClientTradeName)
	}
	if result.ContactEmail != "mariana@example.com" {
		t.Fatalf("email should use published contact snapshot, got %q", result.ContactEmail)
	}
	if result.ValidUntil == nil || result.ValidUntil.Format("2006-01-02") != "2026-12-31" {
		t.Fatalf("email should use published validity, got %v", result.ValidUntil)
	}
}
