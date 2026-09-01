package httpapp

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
)

func (a *App) queueCustomerJourneyEmail(ctx context.Context, acceptanceID, recipientName, recipientEmail string) error {
	var publicToken string
	if err := a.pool.QueryRow(ctx, `
		select pv.public_token::text
		from proposal_acceptances pa
		join proposal_versions pv on pv.id=pa.proposal_version_id
		where pa.id=$1
	`, acceptanceID).Scan(&publicToken); err != nil {
		return err
	}

	link := strings.TrimRight(a.cfg.BaseURL, "/") + "/p/" + publicToken
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Sua proposta ViaGate foi aceita com sucesso.</p><p>Agora precisamos confirmar os dados da empresa e do seguro para preparar o contrato.</p><ul><li>Empresa e endereço</li><li>Seguradora e vigência</li><li>Apólice em PDF, JPG ou PNG</li></ul><p><a href=\"%s\">Continuar contratação</a></p><p>Guarde este e-mail: este é o mesmo link da proposta e ele sempre abrirá exatamente na etapa em que a contratação estiver.</p>",
		html.EscapeString(recipientName),
		html.EscapeString(link),
	)
	textBody := "Sua proposta ViaGate foi aceita. Continue a contratação em " + link + ". Este mesmo link sempre abrirá a etapa atual da contratação."
	return notifications.EnqueueWithOptions(ctx, a.pool, notifications.MessageOptions{
		DedupeKey: "customer-journey:" + acceptanceID,
		Kind:      "customer_journey",
		ToName:    recipientName,
		ToEmail:   recipientEmail,
		Subject:   "Proposta aceita — continue sua contratação ViaGate",
		HTMLBody:  htmlBody,
		TextBody:  textBody,
		Sensitive: true,
	})
}
