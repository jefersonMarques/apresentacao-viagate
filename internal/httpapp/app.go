package httpapp

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/auth"
	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/internal/pipeline"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/companyregistry"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/email"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type contextKey string

const (
	userContextKey      contextKey = "authenticated_user"
	maxRequestBodyBytes int64      = 20 << 20
)

type App struct {
	cfg               config.Config
	pool              *pgxpool.Pool
	logger            *slog.Logger
	authStore         *auth.Store
	proposalStore     *proposals.Store
	presentationStore *presentations.Store
	pipelineStore     *pipeline.Store
	onboardingStore   *onboarding.Store
	contractStore     *contracts.Store
	contractRenderer  *contracts.Renderer
	contractGenerator *contracts.Generator
	contractFinalizer *contracts.Finalizer
	storage           *storage.S3
	mailer            *email.Brevo
	registry          companyregistry.Provider
}

type Dependencies struct {
	Config            config.Config
	Pool              *pgxpool.Pool
	Logger            *slog.Logger
	AuthStore         *auth.Store
	ProposalStore     *proposals.Store
	PresentationStore *presentations.Store
	PipelineStore     *pipeline.Store
	OnboardingStore   *onboarding.Store
	ContractStore     *contracts.Store
	ContractRenderer  *contracts.Renderer
	ContractGenerator *contracts.Generator
	ContractFinalizer *contracts.Finalizer
	Storage           *storage.S3
	Mailer            *email.Brevo
	Registry          companyregistry.Provider
}

func New(deps Dependencies) *App {
	return &App{
		cfg:               deps.Config,
		pool:              deps.Pool,
		logger:            deps.Logger,
		authStore:         deps.AuthStore,
		proposalStore:     deps.ProposalStore,
		presentationStore: deps.PresentationStore,
		pipelineStore:     deps.PipelineStore,
		onboardingStore:   deps.OnboardingStore,
		contractStore:     deps.ContractStore,
		contractRenderer:  deps.ContractRenderer,
		contractGenerator: deps.ContractGenerator,
		contractFinalizer: deps.ContractFinalizer,
		storage:           deps.Storage,
		mailer:            deps.Mailer,
		registry:          deps.Registry,
	}
}

func (a *App) Routes() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)
	router.Use(a.proxyClientIP)
	router.Use(a.securityHeaders)
	router.Use(a.sameOriginWrites)
	router.Use(a.requestBodyLimit)

	router.Handle("/assets/*", http.StripPrefix("/assets/", http.FileServer(http.Dir("web/assets"))))
	registerV1VisualAssets(router)
	router.Get("/media/{id}", a.commercialAsset)
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	router.Get("/readyz", a.ready)

	router.Get("/login", a.loginPage)
	router.Post("/login", a.login)
	router.Post("/logout", a.logout)
	router.Get("/invite/{token}", a.invitationPage)
	router.Post("/invite/{token}", a.acceptInvitation)

	router.Get("/p/{token}", a.publicProposalPage)
	router.Post("/p/{token}/accept", a.acceptProposal)
	router.Get("/a/{token}", a.publicPresentationPage)
	router.Get("/onboarding/resume/{token}", a.resumeOnboarding)

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
	router.Get("/sign/{token}/contract", a.downloadSignedContract)

	router.Group(func(admin chi.Router) {
		admin.Use(a.authenticated)
		admin.Get("/admin", a.dashboard)
		admin.Get("/admin/profile", a.profilePage)
		admin.Post("/admin/profile", a.updateProfile)
		admin.Post("/admin/assets", a.uploadCommercialAsset)
		admin.Get("/admin/api/cnpj/{cnpj}", a.adminLookupCNPJ)
		admin.Get("/admin/pipeline", a.pipelinePage)
		admin.Get("/admin/pipeline/{proposalID}", a.pipelineDetailPage)

		admin.Get("/admin/proposals", a.adminProposals)
		admin.With(a.permission("proposal.create")).Get("/admin/proposals/new", a.newProposalPage)
		admin.With(a.permission("proposal.create")).Get("/admin/proposals/{id}/edit", a.editProposalPage)
		admin.With(a.permission("proposal.create")).Post("/admin/proposals/save", a.saveProposal)

		admin.Get("/admin/presentations", a.adminPresentations)
		admin.With(a.permission("presentation.create")).Get("/admin/presentations/new", a.newPresentationPage)
		admin.With(a.permission("presentation.create")).Get("/admin/presentations/{id}/edit", a.editPresentationPage)
		admin.With(a.permission("presentation.create")).Post("/admin/presentations/save", a.savePresentation)

		admin.With(a.permission("onboarding.review")).Get("/admin/onboardings", a.adminOnboardings)
		admin.With(a.permission("onboarding.review")).Get("/admin/onboardings/{id}", a.adminOnboardingDetail)
		admin.With(a.permission("onboarding.review")).Post("/admin/onboardings/{id}/review", a.reviewOnboarding)
		admin.With(a.permission("onboarding.review")).Post("/admin/onboardings/{id}/contract/retry", a.retryOnboardingContract)
		admin.With(a.permission("onboarding.review")).Get("/admin/onboardings/{id}/documents/{documentID}", a.adminOnboardingDocument)

		admin.Get("/admin/contracts/{contractID}/document", a.adminContractDocument)
		admin.Get("/admin/contracts/{contractID}/evidence", a.adminContractEvidence)
		admin.Get("/admin/contracts/{contractID}/package", a.adminContractPackage)
		admin.With(a.permission("contract.read_all")).Get("/admin/contracts", a.adminContracts)
		admin.With(a.permission("contract.template.manage")).Get("/admin/contracts/templates", a.contractTemplatesPage)
		admin.With(a.permission("contract.template.manage")).Post("/admin/contracts/templates", a.saveContractTemplate)

		admin.With(a.permission("user.manage")).Get("/admin/users", a.usersPage)
		admin.With(a.permission("user.manage")).Post("/admin/users/invite", a.inviteUser)
		admin.With(a.permission("user.manage")).Post("/admin/users/{id}/access", a.updateUserAccess)
	})

	return router
}

func registerV1VisualAssets(router chi.Router) {
	router.Handle("/v1/assets/*", http.StripPrefix("/v1/assets/", http.FileServer(http.Dir("assets"))))
	router.Get("/v1/presentation-content.html", serveV1PresentationContent)
	files := map[string]string{
		"/v1/styles.css":                     "styles.css",
		"/v1/script.js":                      "script.js",
		"/v1/enhancements.css":               "enhancements.css",
		"/v1/insurers.css":                   "insurers.css",
		"/v1/hero-v2.css":                    "hero-v2.css",
		"/v1/executive-v2.css":               "executive-v2.css",
		"/v1/executive-v2.js":                "executive-v2.js",
		"/v1/presentation-fixes.css":          "presentation-fixes.css",
		"/v1/presentation-story.css":          "presentation-story.css",
		"/v1/presentation-story.js":           "presentation-story.js",
		"/v1/presentation-contact.js":         "presentation-contact.js",
		"/v1/presentation-mode.css":           "presentation-mode.css",
		"/v1/presentation-mode.js":            "presentation-mode.js",
		"/v1/presentation-personalization.js": "presentation-personalization.js",
		"/v1/presentation-social-links.js":    "presentation-social-links.js",
		"/v1/presentation-host-bridge.js":     "presentation-host-bridge.js",
		"/v1/presentation-bootstrap.js":       "presentation-bootstrap.js",
		"/v1/viewer-active.css":               "viewer-active.css",
		"/v1/proposal.css":                    "proposal/proposal.css",
		"/v1/proposal-view.css":               "proposal/proposal-view.css",
		"/v1/proposal-premium.css":            "proposal/proposal-premium.css",
		"/v1/proposal-social.css":             "proposal/proposal-social.css",
		"/v1/proposal-experience.css":         "proposal/proposal-experience.css",
	}
	for route, path := range files {
		router.Get(route, serveProjectFile(path))
	}
}

func serveV1PresentationContent(w http.ResponseWriter, r *http.Request) {
	content, err := os.ReadFile("presentation-content.html")
	if err != nil {
		http.Error(w, "apresentação indisponível", http.StatusInternalServerError)
		return
	}
	html := string(content)
	html = strings.Replace(html, `<base href="/apresentacao/" />`, "", 1)
	html = strings.Replace(html, `<base href="/apresentacao/">`, "", 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

func serveProjectFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, path) }
}

func (a *App) proxyClientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.cfg.TrustProxyHeaders {
			next.ServeHTTP(w, r)
			return
		}
		candidate := ""
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			candidate = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
		if candidate == "" {
			candidate = strings.TrimSpace(r.Header.Get("X-Real-IP"))
		}
		if ip := net.ParseIP(candidate); ip != nil {
			r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/v1/") {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; script-src 'self' 'unsafe-inline' https://unpkg.com; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
		} else {
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; script-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		}
		if a.cfg.Environment == "production" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) sameOriginWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		source := r.Header.Get("Origin")
		if source == "" {
			source = r.Header.Get("Referer")
		}
		if source == "" && a.cfg.Environment == "production" {
			http.Error(w, "origem da requisição ausente", http.StatusForbidden)
			return
		}
		if source != "" {
			parsed, err := url.Parse(source)
			if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
				http.Error(w, "origem da requisição não permitida", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) requestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(a.cfg.Session.CookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		user, err := a.authStore.SessionUser(r.Context(), hashToken(cookie.Value))
		if err != nil {
			clearCookie(w, a.cfg.Session.CookieName)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) permission(code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := currentUser(r.Context())
			if !ok {
				http.Error(w, "não autenticado", http.StatusUnauthorized)
				return
			}
			allowed, err := a.authStore.HasPermission(r.Context(), user.ID, code)
			if err != nil || !allowed {
				http.Error(w, "acesso negado", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func currentUser(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(domain.User)
	return user, ok
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
}
