package emailtemplates

import (
	"strings"
	"testing"
)

func TestRenderEscapesHTMLVariables(t *testing.T) {
	template := Template{
		SubjectTemplate: "Proposta — {client.display_name}",
		HTMLTemplate:    "<p>Olá {contact.name}</p><a href=\"{proposal.url}\">Abrir</a>",
		TextTemplate:    "Olá {contact.name}: {proposal.url}",
	}
	draft, err := Render(template, map[string]string{
		"client.display_name": "Cliente XPTO",
		"contact.name":        "<b>Mariana</b>",
		"proposal.url":        "https://viagate.com.br/p/token?a=1&b=2",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(draft.HTMLBody, "<b>Mariana</b>") {
		t.Fatal("HTML variable was not escaped")
	}
	if !strings.Contains(draft.HTMLBody, "&lt;b&gt;Mariana&lt;/b&gt;") {
		t.Fatal("escaped contact name missing")
	}
	if !strings.Contains(draft.HTMLBody, "a=1&amp;b=2") {
		t.Fatal("URL should be escaped for HTML context")
	}
	if !strings.Contains(draft.TextBody, "<b>Mariana</b>") {
		t.Fatal("text body should preserve plain text variable value")
	}
}

func TestValidateRejectsUnsafeHTMLAndUnknownVariables(t *testing.T) {
	for _, htmlTemplate := range []string{
		"<script>alert(1)</script>",
		"<p onclick=\"alert(1)\">x</p>",
		"<a href=\"javascript:alert(1)\">x</a>",
		"<p>{unknown.value}</p>",
	} {
		if err := Validate("Assunto", htmlTemplate, "Texto"); err == nil {
			t.Fatalf("expected validation error for %q", htmlTemplate)
		}
	}
}


func TestRenderAllowsOnlyTrustedProposalProductHTML(t *testing.T) {
	template := Template{
		SubjectTemplate: "Proposta — {client.display_name}",
		HTMLTemplate:    "<div>{proposal.products_html}</div><p>{contact.name}</p>",
		TextTemplate:    "{proposal.products_text}",
	}
	draft, err := Render(template, map[string]string{
		"client.display_name":    "Cliente",
		"contact.name":           "<b>Mariana</b>",
		"proposal.products_html": "<table><tr><td><strong>Cargo Score</strong></td></tr></table>",
		"proposal.products_text": "- Cargo Score",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft.HTMLBody, "<table><tr><td><strong>Cargo Score</strong></td></tr></table>") {
		t.Fatal("trusted proposal product HTML should be inserted as system-generated markup")
	}
	if strings.Contains(draft.HTMLBody, "<b>Mariana</b>") {
		t.Fatal("ordinary variables must remain HTML escaped")
	}
}

func TestValidateRejectsProductHTMLOutsideHTMLBody(t *testing.T) {
	if err := Validate("Produtos {proposal.products_html}", "<p>OK</p>", "Texto"); err == nil {
		t.Fatal("trusted HTML variable must not be allowed in subject")
	}
	if err := Validate("Assunto", "<p>OK</p>", "{proposal.products_html}"); err == nil {
		t.Fatal("trusted HTML variable must not be allowed in text body")
	}
}
