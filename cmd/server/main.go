package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/auth"
	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/httpapp"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/companyregistry"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/database"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/email"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func main(){
	logger:=slog.New(slog.NewJSONHandler(os.Stdout,&slog.HandlerOptions{Level:slog.LevelInfo}))
	cfg,err:=config.Load();if err!=nil{logger.Error("invalid configuration","error",err);os.Exit(1)}

	ctx,cancel:=signal.NotifyContext(context.Background(),syscall.SIGINT,syscall.SIGTERM);defer cancel()
	pool,err:=database.Open(ctx,cfg.DatabaseURL);if err!=nil{logger.Error("database unavailable","error",err);os.Exit(1)};defer pool.Close()
	storageClient,err:=storage.NewS3(ctx,cfg.S3);if err!=nil{logger.Error("S3 unavailable","error",err);os.Exit(1)}
	mailer:=email.NewBrevo(cfg.Brevo)
	registry:=companyregistry.NewBrasilAPI(cfg.Registry.Timeout)

	authStore:=auth.NewStore(pool)
	proposalStore:=proposals.NewStore(pool)
	presentationStore:=presentations.NewStore(pool)
	onboardingStore:=onboarding.NewStore(pool)
	contractStore:=contracts.NewStore(pool)
	renderer:=contracts.NewRenderer()
	pdfRenderer:=contracts.NewPDFRenderer(cfg.ChromiumPath)
	generator:=contracts.NewGenerator(pool,contractStore,renderer,pdfRenderer,storageClient)

	if err:=bootstrapAdmin(ctx,cfg,authStore,pool,logger);err!=nil{logger.Error("bootstrap admin failed","error",err);os.Exit(1)}

	worker:=notifications.NewWorker(pool,mailer,logger)
	go worker.Run(ctx)

	app:=httpapp.New(httpapp.Dependencies{
		Config:cfg,Pool:pool,Logger:logger,AuthStore:authStore,ProposalStore:proposalStore,PresentationStore:presentationStore,
		OnboardingStore:onboardingStore,ContractStore:contractStore,ContractRenderer:renderer,ContractGenerator:generator,
		Storage:storageClient,Mailer:mailer,Registry:registry,
	})
	server:=&http.Server{
		Addr:cfg.Address,Handler:app.Routes(),ReadHeaderTimeout:5*time.Second,ReadTimeout:30*time.Second,WriteTimeout:60*time.Second,IdleTimeout:90*time.Second,
	}
	go func(){
		logger.Info("commercial platform started","addr",cfg.Address,"env",cfg.Environment)
		if err:=server.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{logger.Error("http server failed","error",err);cancel()}
	}()
	<-ctx.Done()
	shutdownCtx,shutdownCancel:=context.WithTimeout(context.Background(),15*time.Second);defer shutdownCancel()
	if err:=server.Shutdown(shutdownCtx);err!=nil{logger.Error("graceful shutdown failed","error",err)}
}

func bootstrapAdmin(ctx context.Context,cfg config.Config,store *auth.Store,pool interface{Exec(context.Context,string,...any)(any,error)},logger *slog.Logger) error {
	if cfg.Bootstrap.AdminEmail==""{return nil}
	name:=strings.TrimSpace(cfg.Bootstrap.AdminName);if name==""{name="Super Admin"}
	token,hash,err:=security.RandomToken(32);if err!=nil{return err}
	created,err:=store.BootstrapAdmin(ctx,strings.ToLower(strings.TrimSpace(cfg.Bootstrap.AdminEmail)),name,hash,time.Now().Add(cfg.Session.InviteTTL));if err!=nil{return err}
	if !created{return nil}
	link:=strings.TrimRight(cfg.BaseURL,"/")+"/invite/"+token
	logger.Warn("bootstrap super admin invitation created","email",cfg.Bootstrap.AdminEmail,"activation_url",link)
	// O primeiro link também é registrado no log para recuperação operacional caso o Brevo ainda não esteja configurado.
	_ = fmt.Sprintf("%v",pool)
	return nil
}
