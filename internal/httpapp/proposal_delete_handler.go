package httpapp

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

const proposalDeleteConfirmation = "Quero excluir"

func proposalDeleteConfirmed(value string) bool {
	return strings.TrimSpace(value) == proposalDeleteConfirmation
}

func (a *App) softDeleteProposal(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if !access.IsSuperAdmin(user) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	if !proposalDeleteConfirmed(r.FormValue("confirmation")) {
		http.Error(w, "Digite exatamente \"Quero excluir\" para confirmar.", http.StatusBadRequest)
		return
	}

	proposalID := chi.URLParam(r, "id")
	result, err := a.proposalStore.SoftDeleteCascade(r.Context(), proposals.SoftDeleteInput{
		ProposalID:  proposalID,
		ActorUserID: user.ID,
		IPAddress:   requestIP(r),
		UserAgent:   r.UserAgent(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "proposta não encontrada", http.StatusNotFound)
			return
		}
		a.logger.Error("soft delete proposal failed", "proposal_id", proposalID, "user_id", user.ID, "error", err)
		http.Error(w, "não foi possível excluir a proposta", http.StatusInternalServerError)
		return
	}

	a.logger.Info(
		"proposal soft deleted",
		"proposal_id", proposalID,
		"user_id", user.ID,
		"original_status", result.OriginalStatus,
		"contracts", result.Contracts,
		"signed_contracts", result.SignedContracts,
		"activations", result.Activations,
	)
	http.Redirect(w, r, "/admin/proposals?deleted=1", http.StatusSeeOther)
}
