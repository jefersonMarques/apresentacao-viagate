package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/database"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fail("configuration", err)
	}
	if cfg.Environment != "production" {
		fail("configuration", fmt.Errorf("APP_ENV must be production for this preflight"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fail("postgres", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		fail("postgres", err)
	}
	fmt.Println("ok postgres")

	storageClient, err := storage.NewS3(ctx, cfg.S3)
	if err != nil {
		fail("s3", err)
	}
	if err := storageClient.Check(ctx); err != nil {
		fail("s3", err)
	}
	fmt.Println("ok s3")

	pdfRenderer := contracts.NewPDFRenderer(cfg.ChromiumPath)
	if err := pdfRenderer.Check(); err != nil {
		fail("browser", err)
	}
	fmt.Println("ok browser")

	fmt.Println("production preflight passed")
}

func fail(component string, err error) {
	fmt.Fprintf(os.Stderr, "preflight failed: %s: %v\n", component, err)
	os.Exit(1)
}
