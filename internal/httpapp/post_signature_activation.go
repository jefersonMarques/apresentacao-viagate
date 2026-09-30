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

func (a *App) issueActivationOwnerPath(ctx context.Context, contractID, signerID, signerName, signerEmail string) (string, error) {
	profile, err := a.activationStore.EnsureForContract(ctx, contractID)
	if err != nil {
		return "", err
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.activationStore.CreateAccessToken(ctx, profile.ID, "owner", "all", signerName, signerEmail, signerID, hash, expiresAt); err != nil {
		return "", err
	}
	return "/activation/" + plain, nil
}

func (a *App) queuePostSignatureActivation(ctx context.Context, access contracts.SignerAccess) error {
	if _, err := a.activationStore.EnsureForContract(ctx, access.Contract.ID); err != nil {
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
		"<p>Olá, %s.</p><p>Seu contrato ViaGate foi assinado com sucesso.</p><p><a href=\"%s\">Baixar contrato assinado</a></p><p>Se a preparação da implantação ainda não estiver concluída, você ou alguém da sua equipe pode continuar pelo link abaixo.</p><p><a href=\"%s\">Continuar preparação da implantação</a></p><p>Se os dados já tiverem sido concluídos, nenhuma ação adicional é necessária: a ViaGate poderá seguir com a implantação interna.</p>",
		html.EscapeString(access.Signer.Name),
		html.EscapeString(contractLink),
		html.EscapeString(activationLink),
	)

	textBody := fmt.Sprintf(
		"Seu contrato ViaGate foi assinado com sucesso.\n\nBaixar contrato assinado: %s\n\nContinuar preparação da implantação, se necessário: %s",
		contractLink,
		activationLink,
	)

	return notifications.EnqueueWithOptions(ctx, a.pool, notifications.MessageOptions{
		DedupeKey: "post-signature-customer:" + access.Contract.ID,
		Kind:      "activation_access",
		ToName:    access.Signer.Name,
		ToEmail:   access.Signer.Email,
		Subject:   "Contrato ViaGate assinado com sucesso",
		HTMLBody:  htmlBody,
		TextBody:  textBody,
		Sensitive: true,
	})
}
