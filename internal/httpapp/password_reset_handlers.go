package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

const passwordResetTTL = time.Hour

func (a *App) forgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	render(r.Context(), w, http.StatusOK, templates.PasswordResetRequestPage("", ""))
}

func (a *App) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PasswordResetRequestPage("", "Informe um e-mail válido."))
		return
	}
	emailAddress, err := cleanEmail(r.FormValue("email"), true)
	if err != nil {
		render(r.Context(), w, http.StatusOK, templates.PasswordResetRequestPage("Se existir uma conta ativa para este e-mail, enviaremos as instruções de recuperação.", ""))
		return
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		a.logger.Error("generate password reset token failed", "error", err)
		render(r.Context(), w, http.StatusOK, templates.PasswordResetRequestPage("Se existir uma conta ativa para este e-mail, enviaremos as instruções de recuperação.", ""))
		return
	}
	expiresAt := time.Now().Add(passwordResetTTL)
	user, found, err := a.authStore.CreatePasswordReset(r.Context(), emailAddress, hash, expiresAt)
	if err != nil {
		a.logger.Error("create password reset failed", "error", err)
	} else if found {
		link := strings.TrimRight(a.cfg.BaseURL, "/") + "/reset-password/" + plain
		htmlBody := fmt.Sprintf("<p>Olá, %s.</p><p>Recebemos uma solicitação para redefinir sua senha da plataforma ViaGate.</p><p><a href=\"%s\">Criar nova senha</a></p><p>O link expira em 1 hora. Se você não solicitou a alteração, ignore este e-mail.</p>", html.EscapeString(user.Name), html.EscapeString(link))
		if enqueueErr := notifications.EnqueueWithOptions(r.Context(), a.pool, notifications.MessageOptions{
			Kind: "password_reset", ToName: user.Name, ToEmail: user.Email,
			Subject: "Redefinição de senha ViaGate", HTMLBody: htmlBody, TextBody: "Redefina sua senha em " + link,
			ExpiresAt: &expiresAt, Sensitive: true,
		}); enqueueErr != nil {
			a.logger.Error("queue password reset failed", "error", enqueueErr, "user_id", user.ID)
		}
	}
	render(r.Context(), w, http.StatusOK, templates.PasswordResetRequestPage("Se existir uma conta ativa para este e-mail, enviaremos as instruções de recuperação.", ""))
}

func (a *App) resetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	user, err := a.authStore.PasswordResetByToken(r.Context(), hashToken(token))
	if err != nil {
		if err != pgx.ErrNoRows {
			a.logger.Error("load password reset token failed", "error", err)
		}
		http.Error(w, "Este link de recuperação é inválido ou expirou.", http.StatusGone)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.PasswordResetFormPage(user.Name, user.Email, token, ""))
}

func (a *App) resetPassword(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	user, err := a.authStore.PasswordResetByToken(r.Context(), hashToken(token))
	if err != nil {
		http.Error(w, "Este link de recuperação é inválido ou expirou.", http.StatusGone)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	password := r.FormValue("password")
	if password != r.FormValue("confirm_password") {
		render(r.Context(), w, http.StatusBadRequest, templates.PasswordResetFormPage(user.Name, user.Email, token, "As senhas são diferentes."))
		return
	}
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PasswordResetFormPage(user.Name, user.Email, token, err.Error()))
		return
	}
	resetUser, err := a.authStore.ResetPassword(r.Context(), hashToken(token), passwordHash)
	if err != nil {
		a.logger.Error("reset password failed", "error", err)
		http.Error(w, "Não foi possível redefinir a senha. Solicite um novo link.", http.StatusConflict)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user','user.password_reset','user',$1,$2,$3,'{}')
	`, resetUser.ID, requestIP(r), r.UserAgent())
	render(r.Context(), w, http.StatusOK, templates.PasswordResetSuccessPage())
}
