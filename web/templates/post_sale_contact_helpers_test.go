package templates

import (
    "bytes"
    "context"
    "strings"
    "testing"

    "github.com/jefersonMarques/apresentacao-viagate/internal/postsale"
)

func TestTeamContactValue(t *testing.T) {
	if actual := TeamContactValue("self", "Signer"); actual != "" {
		t.Fatalf("self-mode contact should not prefill team fields, got %q", actual)
	}
	if actual := TeamContactValue("team", "Operator"); actual != "Operator" {
		t.Fatalf("team contact should be retained, got %q", actual)
	}
}

func TestPostSaleContactPageReusesPublicComponents(t *testing.T) {
    tests := []struct {
        name string
        contact postsale.Contact
    }{
        {name: "signer", contact: postsale.Contact{Mode: "self"}},
        {name: "team", contact: postsale.Contact{
            Mode: "team", Name: "Operations Contact",
            Phone: "41999999999", Email: "operations@example.com",
        }},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            var output bytes.Buffer
            err := PostSaleContactPage("test-token", test.contact, "", "").Render(
                context.Background(), &output,
            )
            if err != nil {
                t.Fatalf("render post-signature contact: %v", err)
            }

            rendered := output.String()
            required := []string{
                "public-flow-header",
                "badge success",
                "signature-complete-box",
                "implementation-contact-choices",
                "data-implementation-team-fields",
                "name=\"contact_mode\"",
                "Baixar contrato assinado",
                "Confirmar contato",
            }
            for _, value := range required {
                if !strings.Contains(rendered, value) {
                    t.Errorf("missing expected component or action %q", value)
                }
            }
            if test.contact.Mode == "self" &&
                strings.Contains(rendered, "operations@example.com") {
                t.Fatal("team details must not be prefilled in self mode")
            }
        })
    }
}
