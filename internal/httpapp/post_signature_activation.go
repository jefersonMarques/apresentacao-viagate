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

func (a *App) issueActivationOwnerPath(ctx context.Context, access contracts.SignerAccess) (string, error) {
	profile, err := a.activationStore.EnsureForSignedContract(ctx, access.Contract.ID)
	if err != nil {
		return "", err
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.activationStore.CreateAccessToken(ctx, profile.ID, "owner", "all", access.Signer.Name, access.Signer.Email, access.Signer.ID, hash, expiresAt); err != nil {
		return "", err
	}
	return "/activation/" + plain, nil
}

func (a *App) queuePostSignatureActivation(ctx context.Context, access contracts.SignerAccess) error {
	if _, err := a.activationStore.EnsureForSignedContract(ctx, access.Contract.ID); err != nil {
		return err
	}
	proposalPath, err := a.proposalPublicPathByContract(ctx, access.Contract.ID)
	if err != nil {
		return err
	}

	signerToken, err := a.contractStore.SignerPublicToken(ctx, access.Signer.ID)
	if err != nil {
		return err
	}

	baseURL := strings.TrimRight(a.cfg.BaseURL, "/")
	activationLink := baseURL + proposalPath
	contractLink := baseURL + "/sign/" + signerToken + "/contract"

	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Seu contrato ViaGate foi assinado com sucesso.</p><p><a href=\"%s\">Baixar contrato assinado</a></p><p>Para prepararmos sua operação, faltam apenas três informações:</p><ul><li><strong>Financeiro</strong> — responsável por faturamento</li><li><strong>Operação</strong> — principais mercadorias transportadas</li><li><strong>Acessos</strong> — usuários iniciais do sistema</li></ul><p><a href=\"%s\">Preencher dados para ativação</a></p><p>Se preferir continuar depois, use o mesmo link da proposta. Ele sempre abrirá a etapa atual da contratação.</p>",
		html.EscapeString(access.Signer.Name),
		html.EscapeString(contractLink),
		html.EscapeString(activationLink),
	)

	textBody := fmt.Sprintf(
		"Seu contrato ViaGate foi assinado com sucesso.\n\nBaixar contrato assinado: %s\n\nPreencher dados para ativação: %s\n\nO mesmo link da proposta pode ser usado para retomar o processo depois.",
		contractLink,
		activationLink,
	)

	return notifications.EnqueueWithOptions(ctx, a.pool, notifications.MessageOptions{
		DedupeKey: "activation-access:" + access.Contract.ID,
		Kind:      "activation_access",
		ToName:    access.Signer.Name,
		ToEmail:   access.Signer.Email,
		Subject:   "Contrato assinado — próximos passos para ativar a ViaGate",
		HTMLBody:  htmlBody,
		TextBody:  textBody,
		Sensitive: true,
	})
}
