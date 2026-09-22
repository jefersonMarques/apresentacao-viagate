package email

import (
	"strings"
	"testing"
)

func TestRenderViaGateHTMLWrapsFragmentInBrandLayout(t *testing.T) {
	result := RenderViaGateHTML(
		"Convite ViaGate",
		"<p>Olá.</p><p><a href=\"https://viagate.com.br\">Continuar</a></p>",
		"",
	)

	for _, expected := range []string{
		"<!doctype html>",
		"#071827",
		"#ff6b18",
		"VIAGATE",
		"Convite ViaGate",
		"<p>Olá.</p>",
	} {
		if !strings.Contains(result, expected) {
			t.Fatalf("rendered email should contain %q", expected)
		}
	}
}

func TestRenderViaGateHTMLUsesEscapedTextWhenHTMLIsEmpty(t *testing.T) {
	result := RenderViaGateHTML(
		"Aviso",
		"",
		"Olá <cliente>\n\nLinha 2",
	)

	if strings.Contains(result, "<cliente>") {
		t.Fatal("plain text fallback must be escaped")
	}
	if !strings.Contains(result, "Olá &lt;cliente&gt;") {
		t.Fatal("escaped plain text should be present in HTML")
	}
	if !strings.Contains(result, "Linha 2") {
		t.Fatal("second paragraph should be present")
	}
}

func TestRenderViaGateHTMLPreservesCompleteHTMLDocument(t *testing.T) {
	document := "<!doctype html><html><body><strong>Custom</strong></body></html>"
	if got := RenderViaGateHTML("Assunto", document, "fallback"); got != document {
		t.Fatal("complete HTML document must not be wrapped twice")
	}
}
