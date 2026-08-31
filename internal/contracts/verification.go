package contracts

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"os"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

const contractVerificationMarker = "<!-- VIAGATE_CONTRACT_VERIFICATION -->"

type VerificationRecord struct {
	ContractID     string
	ClientName     string
	ClientCNPJ     string
	SignerName     string
	Status         string
	DocumentSHA256 []byte
	GeneratedAt    *time.Time
	FullySignedAt  *time.Time
}

func VerificationToken(renderedHTML string) string {
	hash := sha256.Sum256([]byte(renderedHTML))
	return hex.EncodeToString(hash[:])
}

func VerificationCode(token string) string {
	token = strings.ToUpper(strings.TrimSpace(token))
	if len(token) < 12 {
		return "VG-" + token
	}
	return "VG-" + token[:4] + "-" + token[4:8] + "-" + token[8:12]
}

func VerificationURL(token string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return baseURL + "/verify/" + token
}

func InjectVerificationBlock(renderedHTML string) (string, error) {
	if !strings.Contains(renderedHTML, contractVerificationMarker) {
		return renderedHTML, nil
	}

	token := VerificationToken(renderedHTML)
	verificationURL := VerificationURL(token)
	png, err := qrcode.Encode(verificationURL, qrcode.Medium, 220)
	if err != nil {
		return "", fmt.Errorf("generate contract verification QR code: %w", err)
	}
	qrDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)

	block := fmt.Sprintf(`
<section class="contract-verification">
  <div class="contract-verification-copy">
    <h2>Autenticidade do documento</h2>
    <p>Este contrato possui uma identificação criptográfica única registrada antes da assinatura. Escaneie o QR Code para confirmar a situação da assinatura e verificar se uma cópia do PDF corresponde exatamente ao documento registrado pela ViaGate.</p>
    <p><strong>Código de verificação:</strong> %s</p>
    <p class="contract-verification-url">%s</p>
  </div>
  <img class="contract-verification-qr" src="%s" alt="QR Code para verificar a autenticidade deste contrato"/>
</section>`,
		html.EscapeString(VerificationCode(token)),
		html.EscapeString(verificationURL),
		html.EscapeString(qrDataURI),
	)

	return strings.Replace(renderedHTML, contractVerificationMarker, block, 1), nil
}

func (s *Store) VerificationByToken(ctx context.Context, token string) (VerificationRecord, error) {
	var record VerificationRecord
	err := s.pool.QueryRow(ctx, `
		select c.id::text,
		       o.legal_name,
		       o.cnpj,
		       coalesce((
		           select cs.name
		           from contract_signers cs
		           where cs.contract_id=c.id and cs.signer_type='client'
		           order by cs.sign_order,cs.id
		           limit 1
		       ),''),
		       c.status::text,
		       c.document_sha256,
		       c.generated_at,
		       c.fully_signed_at
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		where encode(digest(c.rendered_html,'sha256'),'hex')=$1
		limit 1
	`, strings.ToLower(strings.TrimSpace(token))).Scan(
		&record.ContractID,
		&record.ClientName,
		&record.ClientCNPJ,
		&record.SignerName,
		&record.Status,
		&record.DocumentSHA256,
		&record.GeneratedAt,
		&record.FullySignedAt,
	)
	return record, err
}

func (record VerificationRecord) IsSigned() bool {
	return record.Status == "signed" && record.FullySignedAt != nil
}

func (record VerificationRecord) DocumentHashHex() string {
	return hex.EncodeToString(record.DocumentSHA256)
}

func DocumentMatches(record VerificationRecord, content []byte) bool {
	hash := sha256.Sum256(content)
	return strings.EqualFold(hex.EncodeToString(hash[:]), record.DocumentHashHex())
}
