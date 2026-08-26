package httpapp

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/auth"
	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/companyregistry"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/email"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type contextKey string

const userContextKey contextKey = "authenticated_user"

type App struct {
	cfg              config.Config
	pool             *pgxpool.Pool
	logger           *slog.Logger
	authStore        *auth.Store
	proposalStore    *proposals.Store
	onboardingStore  *onboarding.Store
	contractStore    *contracts.Store
	contractRenderer *contracts.Renderer
	contractGenerator *contracts.Generator
	storage          *storage.S3
	mailer           *email.Brevo
	registry         companyregistry.Provider
}

type Dependencies struct {
	Config            config.Config
	Pool              *pgxpool.Pool
	Logger            *slog.Logger
	AuthStore         *auth.Store
	ProposalStore     *proposals.Store
	OnboardingStore   *onboarding.Store
	ContractStore     *contracts.Store
	ContractRenderer  *contracts.Renderer
	ContractGenerator *contracts.Generator
	Storage           *storage.S3
	Mailer            *email.Brevo
	Registry          companyregistry.Provider
}

func New(deps Dependencies) *App {
	return &App{
		cfg: deps.Config,
		pool: deps.Pool,
		logger: deps.Logger,
		authStore: deps.AuthStore,
		proposalStore: deps.ProposalStore,
		onboardingStore: deps.OnboardingStore,
		contractStore: deps.ContractStore,
		contractRenderer: deps.ContractRenderer,
		contractGenerator: deps.ContractGenerator,
		storage: deps.Storage,
		mailer: deps.Mailer,
		registry: deps.Registry,
	}
}

func (a *App) Routes() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(a.securityHeaders)
	router.Use(a.sameOriginWrites)

	router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	router.Get("/login", a.loginPage)
	router.Post("/login", a.login)
	router.Post("/logout", a.logout)
	router.Get("/invite/{token}", a.invitationPage)
	router.Post("/invite/{token}", a.acceptInvitation)

	router.Get("/p/{token}", a.publicProposalPage)
	router.Post("/p/{token}/accept", a.acceptProposal)

	router.Group(func(customer chi.Router) {
		customer.Use(a.customerSessionRequired)
		customer.Get("/onboarding/{id}", a.onboardingPage)
		customer.Post("/onboarding/{id}", a.saveOnboarding)
		customer.Post("/onboarding/{id}/documents", a.uploadOnboardingDocument)
		customer.Post("/onboarding/{id}/submit", a.submitOnboarding)
		customer.Get("/api/cnpj/{cnpj}", a.lookupCNPJ)
	})

	router.Get("/sign/{token}", a.signaturePage)
	router.Post("/sign/{token}/otp", a.sendSignatureOTP)
	router.Post("/sign/{token}/confirm", a.confirmSignature)

	router.Group(func(admin chi.Router) {
		admin.Use(a.authenticated)
		admin.Get("/admin", a.dashboard)
		admin.Get("/admin/proposals", a.adminProposals)
		admin.Get("/admin/presentations", a.adminPresentations)

		admin.With(a.permission("user.manage")).Get("/admin/users", a.usersPage)
		admin.With(a.permission("user.manage")).Post("/admin/users/invite", a.inviteUser)
		admin.With(a.permission("contract.template.manage")).Get("/admin/contracts/templates", a.contractTemplatesPage)
		admin.With(a.permission("contract.template.manage")).Post("/admin/contracts/templates", a.saveContractTemplate)
	})

	return router
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; script-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if a.cfg.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w,r)
	})
}

func (a *App) sameOriginWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w,r)
			return
		}

		source := r.Header.Get("Origin")
		if source == "" { source = r.Header.Get("Referer") }
		if source != "" {
			parsed, err := url.Parse(source)
			if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
				http.Error(w,"origem da requisição não permitida",http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w,r)
	})
}

func (a *App) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		cookie,err:=r.Cookie(a.cfg.Session.CookieName)
		if err!=nil || cookie.Value=="" { http.Redirect(w,r,"/login",http.StatusSeeOther); return }
		user,err:=a.authStore.SessionUser(r.Context(),hashToken(cookie.Value))
		if err!=nil { clearCookie(w,a.cfg.Session.CookieName); http.Redirect(w,r,"/login",http.StatusSeeOther); return }
		ctx:=context.WithValue(r.Context(),userContextKey,user)
		next.ServeHTTP(w,r.WithContext(ctx))
	})
}

func (a *App) permission(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
			user,ok:=currentUser(r.Context())
			if !ok { http.Error(w,"não autenticado",http.StatusUnauthorized); return }
			allowed,err:=a.authStore.HasPermission(r.Context(),user.ID,code)
			if err!=nil || !allowed { http.Error(w,"acesso negado",http.StatusForbidden); return }
			next.ServeHTTP(w,r)
		})
	}
}

func currentUser(ctx context.Context) (domain.User,bool) {
	user,ok:=ctx.Value(userContextKey).(domain.User)
	return user,ok
}

func clearCookie(w http.ResponseWriter,name string) {
	http.SetCookie(w,&http.Cookie{Name:name,Value:"",Path:"/",MaxAge:-1,HttpOnly:true,SameSite:http.SameSiteStrictMode})
}
