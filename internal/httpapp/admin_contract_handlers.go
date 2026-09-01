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
	if artifact == "evidence" || artifact == "package" {
		allowed, err := a.authStore.HasPermission(r.Context(), user.ID, "contract.evidence.read")
		if err != nil || !allowed {
			http.Error(w, "acesso negado", http.StatusForbidden)
			return
		}
	}

	keys, err := a.contractStore.ArtifactKeysByContractID(r.Context(), contractID)
	if err != nil {
		http.Error(w, "contrato não encontrado", http.StatusNotFound)
		return
	}

	switch artifact {
	case "contract":
		a.auditContractArtifactAccess(r, user.ID, contractID, artifact)
		a.redirectPrivateArtifact(w, r, keys.ContractKey, "contrato-viagate.pdf")
	case "evidence":
		if !keys.Finalized || keys.EvidenceKey == "" {
			http.Error(w, "O relatório técnico da assinatura ainda está sendo preparado.", http.StatusConflict)
			return
		}
		a.auditContractArtifactAccess(r, user.ID, contractID, artifact)
		a.redirectPrivateArtifact(w, r, keys.EvidenceKey, "relatorio-tecnico-assinatura-viagate.pdf")
	case "package":
		if !keys.Finalized || keys.PackageKey == "" {
			http.Error(w, "Os documentos finais da assinatura ainda estão sendo preparados.", http.StatusConflict)
			return
		}
		a.auditContractArtifactAccess(r, user.ID, contractID, artifact)
		a.redirectPrivateArtifact(w, r, keys.PackageKey, "documentos-assinatura-viagate.zip")
	default:
		http.NotFound(w, r)
	}
}

func (a *App) auditContractArtifactAccess(r *http.Request, userID, contractID, artifact string) {
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user','contract.artifact_downloaded','contract',$2,$3,$4,jsonb_build_object('artifact',$5::text))
	`, userID, contractID, requestIP(r), r.UserAgent(), artifact)
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

	readAll, err := a.authStore.HasPermission(ctx, userID, "contract.read_all")
	if err == nil && readAll {
		return true
	}
	if ownerID != userID {
		return false
	}
	readOwn, err := a.authStore.HasPermission(ctx, userID, "contract.read_own")
	return err == nil && readOwn
}
