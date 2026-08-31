package contracts

import (
	"strings"
	"testing"
)

func TestVerificationTokenIsStableForRenderedHTML(t *testing.T) {
	html := "<h1>Contrato</h1>" + contractVerificationMarker
	first := VerificationToken(html)
	second := VerificationToken(html)

	if first != second {
		t.Fatal("verification token must be stable for the same rendered HTML")
	}
	if len(first) != 64 {
		t.Fatalf("expected SHA-256 token with 64 hex characters, got %d", len(first))
	}
	if VerificationToken(html+"alterado") == first {
		t.Fatal("verification token must change when rendered HTML changes")
	}
}

func TestInjectVerificationBlockReplacesMarkerWithQRCode(t *testing.T) {
	t.Setenv("APP_BASE_URL", "https://viagate.com.br")
	html := "<h1>Contrato</h1>" + contractVerificationMarker
	token := VerificationToken(html)

	result, err := InjectVerificationBlock(html)
	if err != nil {
		t.Fatalf("InjectVerificationBlock returned error: %v", err)
	}
	if strings.Contains(result, contractVerificationMarker) {
		t.Fatal("verification marker must be replaced")
	}
	if !strings.Contains(result, "data:image/png;base64,") {
		t.Fatal("expected embedded QR code image")
	}
	if !strings.Contains(result, "https://viagate.com.br/verify/"+token) {
		t.Fatal("expected public verification URL")
	}
	if !strings.Contains(result, VerificationCode(token)) {
		t.Fatal("expected human-readable verification code")
	}
}
