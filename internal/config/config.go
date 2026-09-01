package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment             string
	Address                 string
	BaseURL                 string
	DatabaseURL             string
	ChromiumPath            string
	TrustProxyHeaders       bool
	RequireOnboardingReview bool
	Company                 CompanyConfig
	Session                 SessionConfig
	S3                      S3Config
	Brevo                   BrevoConfig
	Registry                RegistryConfig
	Bootstrap               BootstrapConfig
}

type CompanyConfig struct {
	LegalName string
	CNPJ      string
}

type SessionConfig struct {
	CookieName      string
	TTL             time.Duration
	InviteTTL       time.Duration
	SignatureOTPTTL time.Duration
}

type S3Config struct {
	Stage                string
	Region               string
	Bucket               string
	Endpoint             string
	AccessKeyID          string
	SecretAccessKey      string
	UsePathStyle         bool
	ServerSideEncryption string
	KMSKeyID             string
	DownloadTTL          time.Duration
}

type BrevoConfig struct {
	APIKey      string
	SenderEmail string
	SenderName  string
}

type RegistryConfig struct {
	Provider string
	Timeout  time.Duration
}

type BootstrapConfig struct {
	AdminEmail string
	AdminName  string
}

func Load() (Config, error) {
	environment := strings.ToLower(strings.TrimSpace(env("APP_ENV", "development")))
	storageStage := "dev"
	if environment == "production" {
		storageStage = "prod"
	}

	cfg := Config{
		Environment:             environment,
		Address:                 env("APP_ADDR", ":8080"),
		BaseURL:                 env("APP_BASE_URL", "http://localhost:8080"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		ChromiumPath:            env("CHROMIUM_PATH", "chromium"),
		TrustProxyHeaders:       boolean("TRUST_PROXY_HEADERS", false),
		RequireOnboardingReview: boolean("REQUIRE_ONBOARDING_REVIEW", true),
		Company: CompanyConfig{
			LegalName: os.Getenv("VIAGATE_LEGAL_NAME"),
			CNPJ:      os.Getenv("VIAGATE_CNPJ"),
		},
		Session: SessionConfig{
			CookieName:      env("SESSION_COOKIE_NAME", "viagate_session"),
			TTL:             hours("SESSION_TTL_HOURS", 12),
			InviteTTL:       hours("INVITE_TTL_HOURS", 48),
			SignatureOTPTTL: minutes("SIGNATURE_OTP_TTL_MINUTES", 10),
		},
		S3: S3Config{
			Stage:                strings.ToLower(strings.TrimSpace(env("S3_STAGE", storageStage))),
			Region:               strings.TrimSpace(env("S3_REGION", "us-east-1")),
			Bucket:               strings.TrimSpace(os.Getenv("S3_BUCKET")),
			Endpoint:             strings.TrimSpace(os.Getenv("S3_ENDPOINT")),
			AccessKeyID:          os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey:      os.Getenv("S3_SECRET_ACCESS_KEY"),
			UsePathStyle:         boolean("S3_USE_PATH_STYLE", false),
			ServerSideEncryption: env("S3_SERVER_SIDE_ENCRYPTION", "AES256"),
			KMSKeyID:             os.Getenv("S3_KMS_KEY_ID"),
			DownloadTTL:          minutes("S3_DOWNLOAD_TTL_MINUTES", 5),
		},
		Brevo: BrevoConfig{
			APIKey:      os.Getenv("BREVO_API_KEY"),
			SenderEmail: env("BREVO_SENDER_EMAIL", "naoresponda@viagate.com.br"),
			SenderName:  env("BREVO_SENDER_NAME", "ViaGate"),
		},
		Registry: RegistryConfig{
			Provider: env("COMPANY_REGISTRY_PROVIDER", "brasilapi"),
			Timeout:  seconds("COMPANY_REGISTRY_TIMEOUT_SECONDS", 4),
		},
		Bootstrap: BootstrapConfig{
			AdminEmail: os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
			AdminName:  os.Getenv("BOOTSTRAP_ADMIN_NAME"),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.S3.Bucket == "" {
		return Config{}, fmt.Errorf("S3_BUCKET is required")
	}
	if cfg.S3.Stage != "dev" && cfg.S3.Stage != "prod" {
		return Config{}, fmt.Errorf("S3_STAGE must be dev or prod")
	}
	if cfg.Environment == "production" && cfg.S3.Stage != "prod" {
		return Config{}, fmt.Errorf("S3_STAGE must be prod when APP_ENV=production")
	}
	if cfg.Environment != "production" && cfg.S3.Stage != "dev" {
		return Config{}, fmt.Errorf("S3_STAGE must be dev outside production")
	}
	if cfg.S3.ServerSideEncryption != "none" && cfg.S3.ServerSideEncryption != "AES256" && cfg.S3.ServerSideEncryption != "aws:kms" {
		return Config{}, fmt.Errorf("S3_SERVER_SIDE_ENCRYPTION must be none, AES256 or aws:kms")
	}
	if cfg.S3.ServerSideEncryption == "aws:kms" && cfg.S3.KMSKeyID == "" {
		return Config{}, fmt.Errorf("S3_KMS_KEY_ID is required when using aws:kms")
	}
	if cfg.Environment == "production" && cfg.S3.ServerSideEncryption == "none" && cfg.S3.Endpoint == "" {
		return Config{}, fmt.Errorf("S3 server-side encryption is required for AWS S3 in production")
	}
	if cfg.Environment == "production" && (cfg.Company.LegalName == "" || cfg.Company.CNPJ == "") {
		return Config{}, fmt.Errorf("VIAGATE_LEGAL_NAME and VIAGATE_CNPJ are required in production")
	}
	if cfg.Environment == "production" && cfg.Brevo.APIKey == "" {
		return Config{}, fmt.Errorf("BREVO_API_KEY is required in production")
	}
	if cfg.Environment == "production" {
		if err := validateProductionBaseURL(cfg.BaseURL); err != nil {
			return Config{}, err
		}
	}

	return cfg, nil
}

func validateProductionBaseURL(value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return fmt.Errorf("APP_BASE_URL must be an absolute public URL in production")
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("APP_BASE_URL must use https in production")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("APP_BASE_URL cannot point to localhost in production")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("APP_BASE_URL cannot contain query parameters or fragments")
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func integer(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func boolean(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func hours(key string, fallback int) time.Duration {
	return time.Duration(integer(key, fallback)) * time.Hour
}

func minutes(key string, fallback int) time.Duration {
	return time.Duration(integer(key, fallback)) * time.Minute
}

func seconds(key string, fallback int) time.Duration {
	return time.Duration(integer(key, fallback)) * time.Second
}
