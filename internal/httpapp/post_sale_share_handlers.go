package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) createPostSaleShare(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	detail, ok := a.postSaleDetail(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	delivery := strings.TrimSpace(r.FormValue("delivery"))
	if delivery != "copy" && delivery != "email" {
		http.Error(w, "Selecione uma forma de compartilhamento.", http.StatusBadRequest)
		return
	}
	if delivery == "email" && detail.Contact.Email == "" {
		http.Error(w, "Informe o e-mail do contato antes de enviar.", http.StatusBadRequest)
		return
	}

	if _, err := a.activationStore.EnsureForContract(r.Context(), detail.ContractID); err != nil {
		a.logger.Error("ensure post-sale activation failed", "error", err)
		http.Error(w, "Não foi possível preparar o formulário.", http.StatusInternalServerError)
		return
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		http.Error(w, "Não foi possível gerar o link.", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.postSaleStore.CreateSharedAccess(
		r.Context(),detail.ContractID,user.ID,detail.Contact.Name,detail.Contact.Email,delivery,hash,expiresAt,
	); err != nil {
		a.logger.Error("create post-sale access failed", "contract_id", detail.ContractID, "error", err)
		if err == pgx.ErrNoRows {
			http.Error(w, "A preparação já foi concluída ou o contrato não está disponível.", http.StatusConflict)
			return
		}
		http.Error(w, "Não foi possível gerar o link. Verifique se há solicitações recentes.", http.StatusConflict)
		return
	}

	link := strings.TrimRight(a.cfg.BaseURL, "/")+"/activation/"+plain
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	message := "Link seguro gerado. Copie ou compartilhe com a pessoa responsável."
	errorMessage := ""
	if delivery == "email" {
		htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>Precisamos completar alguns dados para a implantação da ViaGate.</p><p><a href=%q>Preencher dados da implantação</a></p><p>O link expira em 30 dias.</p>",
		html.EscapeString(detail.Contact.Name), html.EscapeString(link),
		)
		err := notifications.EnqueueWithOptions(r.Context(),a.pool,notifications.MessageOptions{
			Kind:"activation_access",ToName:detail.Contact.Name,ToEmail:detail.Contact.Email,
			Subject:"ViaGate — Dados para implantação",
			HTMLBody:htmlBody,TextBody:"Complete os dados da implantação: "+link,
			Sensitive:true,ExpiresAt:&expiresAt,
		})
		if err != nil {
			a.logger.Error("queue post-sale share email failed", "contract_id", detail.ContractID, "error", err)
			errorMessage = "O e-mail não foi colocado na fila. Copie o link para compartilhar manualmente."
		} else {
			message = "E-mail de convite colocado na fila para envio."
		}
	}
	render(r.Context(), w, http.StatusOK, templates.PostSaleShareLinkPage(user, detail, link, message, errorMessage))
}
