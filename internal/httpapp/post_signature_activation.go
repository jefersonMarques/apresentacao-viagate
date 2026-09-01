package httpapp

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
)

func (a *App) queuePostSignatureActivation(ctx context.Context, access contracts.SignerAccess) error {
	profile, err := a.activationStore.EnsureForSignedContract(ctx, access.Contract.ID)
	if err != nil {
		return err
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.activationStore.CreateAccessToken(ctx, profile.ID, "owner", "all", access.Signer.Name, access.Signer.Email, access.Signer.ID, hash, expiresAt); err != nil {
		return err
	}

	link := strings.TrimRight(a.cfg.BaseURL, "/") + "/activation/" + plain
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Seu contrato ViaGate foi assinado com sucesso.</p><p>Para prepararmos sua operação, faltam apenas três informações:</p><ul><li><strong>Financeiro</strong> — responsável por faturamento</li><li><strong>Operação</strong> — principais mercadorias transportadas</li><li><strong>Acessos</strong> — usuários iniciais do sistema</li></ul><p><a href=\"%s\">Preencher dados para ativação</a></p><p>Você pode começar agora, continuar depois e também encaminhar partes do preenchimento para alguém da sua equipe.</p>",
		html.EscapeString(access.Signer.Name),
		html.EscapeString(link),
	)
	return notifications.EnqueueWithOptions(ctx, a.pool, notifications.MessageOptions{
		DedupeKey: "activation-access:" + access.Contract.ID,
		Kind:      "activation_access",
		ToName:    access.Signer.Name,
		ToEmail:   access.Signer.Email,
		Subject:   "Contrato assinado — próximos passos para ativar a ViaGate",
		HTMLBody:  htmlBody,
		TextBody:  "Seu contrato foi assinado. Preencha os dados para ativação em " + link,
		ExpiresAt: &expiresAt,
		Sensitive: true,
	})
}
