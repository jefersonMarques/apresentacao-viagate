package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	appconfig "github.com/jefersonMarques/apresentacao-viagate/internal/config"
)

type Brevo struct {
	apiKey      string
	senderEmail string
	senderName  string
	client      *http.Client
}

type Message struct {
	ToName   string
	ToEmail  string
	Subject  string
	HTMLBody string
	TextBody string
}

func NewBrevo(cfg appconfig.BrevoConfig) *Brevo {
	return &Brevo{
		apiKey:      cfg.APIKey,
		senderEmail: cfg.SenderEmail,
		senderName:  cfg.SenderName,
		client:      &http.Client{Timeout: 8 * time.Second},
	}
}

func (b *Brevo) Send(ctx context.Context, message Message) error {
	if strings.TrimSpace(b.apiKey) == "" {
		return fmt.Errorf("BREVO_API_KEY is not configured")
	}
	if strings.TrimSpace(b.senderEmail) == "" {
		return fmt.Errorf("BREVO_SENDER_EMAIL is not configured")
	}
	if strings.TrimSpace(message.ToEmail) == "" {
		return fmt.Errorf("recipient email is empty")
	}

	payload := map[string]any{
		"sender": map[string]string{"name": b.senderName, "email": b.senderEmail},
		"to": []map[string]string{{"name": message.ToName, "email": message.ToEmail}},
		"subject":     message.Subject,
		"htmlContent": message.HTMLBody,
		"textContent": message.TextBody,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal brevo payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create brevo request: %w", err)
	}
	req.Header.Set("api-key", b.apiKey)
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("send brevo email: %w", err)
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
	if readErr != nil {
		return fmt.Errorf("read brevo response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail := strings.TrimSpace(string(responseBody))
		if detail == "" {
			detail = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("brevo returned status %d: %s", resp.StatusCode, detail)
	}

	return nil
}
