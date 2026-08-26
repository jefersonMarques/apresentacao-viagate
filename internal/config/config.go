package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment string
	Address     string
	BaseURL     string
	DatabaseURL string
	Session     SessionConfig
	S3          S3Config
	Brevo       BrevoConfig
	Registry    RegistryConfig
	Bootstrap   BootstrapConfig
}

type SessionConfig struct {
	CookieName       string
	TTL              time.Duration
	InviteTTL        time.Duration
	SignatureOTPTTL  time.Duration
}

type S3Config struct {
	Region          string
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
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
	cfg := Config{
		Environment: env("APP_ENV", "development"),
		Address:     env("APP_ADDR", ":8080"),
		BaseURL:     env("APP_BASE_URL", "http://localhost:8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Session: SessionConfig{
			CookieName:      env("SESSION_COOKIE_NAME", "viagate_session"),
			TTL:             hours("SESSION_TTL_HOURS", 12),
			InviteTTL:       hours("INVITE_TTL_HOURS", 48),
			SignatureOTPTTL: minutes("SIGNATURE_OTP_TTL_MINUTES", 10),
		},
		S3: S3Config{
			Region:          env("S3_REGION", "us-east-1"),
			Bucket:          os.Getenv("S3_BUCKET"),
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			UsePathStyle:    boolean("S3_USE_PATH_STYLE", false),
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

	return cfg, nil
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
