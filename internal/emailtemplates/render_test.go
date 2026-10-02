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
