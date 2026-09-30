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

func (a *App) continueOnboardingContract(ctx context.Context, onboardingID, status, source string) (contracts.DeliveryAccess, error) {
	switch status {
	case "submitted":
		if _, err := a.onboardingStore.AutoApprove(ctx, onboardingID, source); err != nil {
			return contracts.DeliveryAccess{}, fmt.Errorf("auto approve onboarding: %w", err)
		}
	case "approved":
		// O contrato pode ter falhado em uma tentativa anterior; ensureContractDelivery
		// é idempotente e retoma a partir do estado persistido.
	default:
		return contracts.DeliveryAccess{}, fmt.Errorf("onboarding cannot continue contract delivery in status %s", status)
	}

	access, _, err := a.ensureContractDelivery(ctx, onboardingID)
	if err != nil {
		return contracts.DeliveryAccess{}, err
	}
	return access, nil
}

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
	signatureLink := strings.TrimRight(a.cfg.BaseURL, "/") + proposalPath

	activationPath, err := a.issueActivationOwnerPath(
		ctx,
		access.ContractID,
		access.SignerID,
		access.SignerName,
		access.SignerEmail,
	)
	if err != nil {
		return access, created, fmt.Errorf("prepare activation access: %w", err)
	}
	activationLink := strings.TrimRight(a.cfg.BaseURL, "/") + activationPath

	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Seu contrato ViaGate está pronto para conferência e assinatura.</p><p><a href=\"%s\">Revisar e assinar contrato</a></p><p>Enquanto a assinatura estiver pendente, você já pode preparar a implantação ou encaminhar essa etapa para alguém da sua equipe.</p><p><a href=\"%s\">Preparar dados da implantação</a></p><p>A implantação interna só será liberada quando o contrato estiver assinado e os dados operacionais estiverem concluídos.</p>",
		html.EscapeString(access.SignerName),
		html.EscapeString(signatureLink),
		html.EscapeString(activationLink),
	)
	if err := notifications.EnqueueUnique(
		ctx,
		a.pool,
		"contract-signature:"+access.ContractID+":"+access.SignerID,
		access.SignerName,
		access.SignerEmail,
		"Seu contrato ViaGate está pronto para assinatura",
		htmlBody,
		"Seu contrato ViaGate está pronto. Assinatura: "+signatureLink+"\nPreparação da implantação: "+activationLink,
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
