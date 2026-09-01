package httpapp

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
)

const customerJourneyTTL = 30 * 24 * time.Hour

func (a *App) queueCustomerJourneyEmail(ctx context.Context, acceptanceID, recipientName, recipientEmail string) error {
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(customerJourneyTTL)
	if err := a.proposalStore.CreateCustomerJourneyToken(ctx, acceptanceID, hash, expiresAt); err != nil {
		return err
	}

	link := strings.TrimRight(a.cfg.BaseURL, "/") + "/onboarding/resume/" + plain
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Sua proposta ViaGate foi aceita com sucesso.</p><p>Para prepararmos o contrato, confirme os dados da empresa e do seguro e envie a apólice.</p><ul><li>Empresa e endereço</li><li>Seguradora e vigência</li><li>Apólice em PDF, JPG ou PNG</li></ul><p><a href=\"%s\">Continuar contratação</a></p><p>Você pode começar agora e retomar depois por este mesmo link.</p>",
		html.EscapeString(recipientName),
		html.EscapeString(link),
	)
	textBody := "Sua proposta ViaGate foi aceita. Continue a contratação em " + link + ". Você pode retomar depois por este mesmo link."
	return notifications.EnqueueWithOptions(ctx, a.pool, notifications.MessageOptions{
		DedupeKey: "customer-journey:" + acceptanceID + ":" + time.Now().UTC().Format("20060102"),
		Kind:      "customer_journey",
		ToName:    recipientName,
		ToEmail:   recipientEmail,
		Subject:   "Proposta aceita — continue sua contratação ViaGate",
		HTMLBody:  htmlBody,
		TextBody:  textBody,
		ExpiresAt: &expiresAt,
		Sensitive: true,
	})
}
