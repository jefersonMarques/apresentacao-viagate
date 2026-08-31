package httpapp

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) adminContracts(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := a.contractStore.ListAdminContracts(r.Context())
	if err != nil {
		a.logger.Error("load contracts failed", "error", err)
		http.Error(w, "não foi possível carregar os contratos", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.ContractsAdminPage(user, items))
}

func (a *App) adminContractDocument(w http.ResponseWriter, r *http.Request) {
	a.adminContractArtifact(w, r, "contract")
}

func (a *App) adminContractEvidence(w http.ResponseWriter, r *http.Request) {
	a.adminContractArtifact(w, r, "evidence")
}

func (a *App) adminContractPackage(w http.ResponseWriter, r *http.Request) {
	a.adminContractArtifact(w, r, "package")
}

func (a *App) adminContractArtifact(w http.ResponseWriter, r *http.Request, artifact string) {
	user, _ := currentUser(r.Context())
	contractID := chi.URLParam(r, "contractID")
	if !a.canAccessContract(r.Context(), user.ID, contractID) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}

	keys, err := a.contractStore.ArtifactKeysByContractID(r.Context(), contractID)
	if err != nil {
		http.Error(w, "contrato não encontrado", http.StatusNotFound)
		return
	}

	switch artifact {
	case "contract":
		a.redirectPrivateArtifact(w, r, keys.ContractKey, "contrato-viagate.pdf")
	case "evidence":
		if !keys.Finalized || keys.EvidenceKey == "" {
			http.Error(w, "O relatório técnico da assinatura ainda está sendo preparado.", http.StatusConflict)
			return
		}
		a.redirectPrivateArtifact(w, r, keys.EvidenceKey, "relatorio-tecnico-assinatura-viagate.pdf")
	case "package":
		if !keys.Finalized || keys.PackageKey == "" {
			http.Error(w, "Os documentos finais da assinatura ainda estão sendo preparados.", http.StatusConflict)
			return
		}
		a.redirectPrivateArtifact(w, r, keys.PackageKey, "documentos-assinatura-viagate.zip")
	default:
		http.NotFound(w, r)
	}
}

func (a *App) canAccessContract(ctx context.Context, userID, contractID string) bool {
	var ownerID string
	err := a.pool.QueryRow(ctx, `
		select p.created_by::text
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		where c.id=$1 and c.status<>'cancelled'
	`, contractID).Scan(&ownerID)
	if err != nil {
		return false
	}
	if ownerID == userID {
		return true
	}

	for _, permission := range []string{"contract.read_all", "proposal.read_all", "onboarding.review"} {
		allowed, err := a.authStore.HasPermission(ctx, userID, permission)
		if err == nil && allowed {
			return true
		}
	}
	return false
}
