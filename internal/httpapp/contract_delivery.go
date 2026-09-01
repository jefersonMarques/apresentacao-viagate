package httpapp

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
)

func (a *App) ensureContractDelivery(ctx context.Context, onboardingID string) (contracts.DeliveryAccess, bool, error) {
	access, err := a.contractStore.DeliveryByOnboarding(ctx, onboardingID)
	created := false
	if err == pgx.ErrNoRows {
		generated, generationErr := a.contractGenerator.GenerateForOnboarding(ctx, onboardingID)
		if generationErr != nil {
			return contracts.DeliveryAccess{}, false, generationErr
		}
		created = true
		access = contracts.DeliveryAccess{
			ContractID:     generated.ContractID,
			ContractStatus: "generated",
			SignerID:       generated.SignerID,
			SignerToken:    generated.SignerToken,
			SignerName:     generated.SignerName,
			SignerEmail:    generated.SignerEmail,
			SignerStatus:   "pending",
		}
	} else if err != nil {
		return contracts.DeliveryAccess{}, false, err
	}

	if access.ContractStatus == "signed" || access.ContractStatus == "partially_signed" || access.ContractStatus == "sent" {
		return access, created, nil
	}
	if access.ContractStatus != "generated" {
		return access, created, fmt.Errorf("contract cannot be delivered in status %s", access.ContractStatus)
	}

	proposalPath, err := a.proposalPublicPathByOnboarding(ctx, onboardingID)
	if err != nil {
		return access, created, fmt.Errorf("resolve proposal journey link: %w", err)
	}
	link := strings.TrimRight(a.cfg.BaseURL, "/") + proposalPath
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Os dados da operação foram recebidos e o seu contrato ViaGate está pronto para conferência e assinatura.</p><p><a href=\"%s\">Revisar e assinar contrato</a></p><p>Este é o mesmo link da proposta. Ao abri-lo, você será levado diretamente ao contrato e verá o PDF exato que será assinado.</p>",
		html.EscapeString(access.SignerName),
		html.EscapeString(link),
	)
	if err := notifications.EnqueueUnique(
		ctx,
		a.pool,
		"contract-signature:"+access.ContractID+":"+access.SignerID,
		access.SignerName,
		access.SignerEmail,
		"Seu contrato ViaGate está pronto para assinatura",
		htmlBody,
		"Seu contrato ViaGate está pronto. Revise e assine usando o mesmo link da proposta: "+link,
	); err != nil {
		return access, created, err
	}
	if err := a.contractStore.MarkSent(ctx, access.ContractID); err != nil {
		return access, created, err
	}
	access.ContractStatus = "sent"

	if _, err := a.pool.Exec(ctx, `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata)
		values('system','contract.sent','contract',$1,jsonb_build_object('signer_id',$2::uuid,'channel','email'))
	`, access.ContractID, access.SignerID); err != nil {
		a.logger.Error("record contract sent audit failed", "contract_id", access.ContractID, "signer_id", access.SignerID, "error", err)
	}
	return access, created, nil
}
