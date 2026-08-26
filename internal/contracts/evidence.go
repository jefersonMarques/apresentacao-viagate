package contracts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
)

type Finalizer struct {
	pool    *pgxpool.Pool
	pdf     *PDFRenderer
	storage *storage.S3
	company config.CompanyConfig
}

type EvidenceReport struct {
	Version          string             `json:"version"`
	ContractID       string             `json:"contract_id"`
	ClientLegalName  string             `json:"client_legal_name"`
	ClientCNPJ       string             `json:"client_cnpj"`
	ProposalVersion  int                `json:"proposal_version"`
	TemplateVersion  int                `json:"template_version"`
	DocumentSHA256   string             `json:"document_sha256"`
	GeneratedAt      time.Time          `json:"generated_at"`
	FullySignedAt    time.Time          `json:"fully_signed_at"`
	IssuedAt         time.Time          `json:"issued_at"`
	CompanyLegalName string             `json:"viagate_legal_name"`
	CompanyCNPJ      string             `json:"viagate_cnpj"`
	Acceptance       EvidenceAcceptance `json:"proposal_acceptance"`
	Signers          []EvidenceSigner   `json:"signers"`
	Events           []EvidenceEvent    `json:"events"`
}

type EvidenceAcceptance struct {
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CPF            string    `json:"cpf"`
	Role           string    `json:"role,omitempty"`
	AcceptedAt     time.Time `json:"accepted_at"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	SessionID      string    `json:"session_id"`
	TextVersion    string    `json:"text_version"`
	Text           string    `json:"text"`
	TextSHA256     string    `json:"text_sha256"`
	ProposalSHA256 string    `json:"proposal_sha256"`
}

type EvidenceSigner struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	CPF            string     `json:"cpf"`
	Role           string     `json:"role,omitempty"`
	Status         string     `json:"status"`
	SignedAt       *time.Time `json:"signed_at,omitempty"`
	SessionID      string     `json:"signature_session_id,omitempty"`
	ConsentVersion string     `json:"signature_consent_version,omitempty"`
	ConsentText    string     `json:"signature_consent_text,omitempty"`
	ConsentSHA256  string     `json:"signature_consent_sha256,omitempty"`
	OTP            bool       `json:"email_otp_verified"`
	Face           bool       `json:"face_verified"`
	Liveness       bool       `json:"liveness_verified"`
}

type EvidenceEvent struct {
	Type       string         `json:"type"`
	SignerID   string         `json:"signer_id,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func NewFinalizer(pool *pgxpool.Pool, pdf *PDFRenderer, storageClient *storage.S3, company config.CompanyConfig) *Finalizer {
	return &Finalizer{pool: pool, pdf: pdf, storage: storageClient, company: company}
}

func (f *Finalizer) Finalize(ctx context.Context, contractID string) error {
	report, contractKey, err := f.loadReport(ctx, contractID)
	if err != nil { return err }

	evidenceJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil { return fmt.Errorf("marshal evidence report: %w", err) }
	evidenceHTML, err := renderEvidenceHTML(report)
	if err != nil { return fmt.Errorf("render evidence report: %w", err) }
	evidencePDF, err := f.pdf.Render(ctx, evidenceHTML)
	if err != nil { return fmt.Errorf("render evidence PDF: %w", err) }

	contractReader, _, _, err := f.storage.Get(ctx, contractKey)
	if err != nil { return err }
	contractPDF, err := io.ReadAll(contractReader)
	_ = contractReader.Close()
	if err != nil { return fmt.Errorf("read contract PDF: %w", err) }
	if sha256Hex(contractPDF)!=report.DocumentSHA256{return fmt.Errorf("stored contract PDF hash does not match signed document hash")}

	manifest := map[string]string{
		"contract.pdf":  sha256Hex(contractPDF),
		"evidence.pdf":  sha256Hex(evidencePDF),
		"evidence.json": sha256Hex(evidenceJSON),
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil { return err }

	packageBytes, err := zipEvidencePackage(contractPDF, evidencePDF, evidenceJSON, manifestJSON)
	if err != nil { return err }
	evidenceHash := sha256.Sum256(evidencePDF)
	packageHash := sha256.Sum256(packageBytes)
	baseKey := fmt.Sprintf("contracts/%s/final", contractID)
	evidenceKey := baseKey + "/evidence.pdf"
	packageKey := baseKey + "/signed-package.zip"

	if err := f.storage.Put(ctx, evidenceKey, "application/pdf", bytes.NewReader(evidencePDF), int64(len(evidencePDF))); err != nil { return err }
	if err := f.storage.Put(ctx, packageKey, "application/zip", bytes.NewReader(packageBytes), int64(len(packageBytes))); err != nil {
		_ = f.storage.Delete(ctx, evidenceKey)
		return err
	}

	command, err := f.pool.Exec(ctx, `
		update contracts
		set evidence_report_storage_key=$2,evidence_report_sha256=$3,
		    final_package_storage_key=$4,final_package_sha256=$5,finalized_at=now(),updated_at=now()
		where id=$1 and status='signed' and finalized_at is null
	`, contractID, evidenceKey, evidenceHash[:], packageKey, packageHash[:])
	if err != nil { return fmt.Errorf("persist evidence artifacts: %w", err) }
	if command.RowsAffected() == 0 {
		_ = f.storage.Delete(ctx, evidenceKey)
		_ = f.storage.Delete(ctx, packageKey)
		return nil
	}

	_, _ = f.pool.Exec(ctx, `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata)
		values('system','contract.evidence_finalized','contract',$1,
		       jsonb_build_object('evidence_sha256',$2,'package_sha256',$3))
	`, contractID, hex.EncodeToString(evidenceHash[:]), hex.EncodeToString(packageHash[:]))
	return nil
}

func (f *Finalizer) loadReport(ctx context.Context, contractID string) (EvidenceReport, string, error) {
	var report EvidenceReport
	var documentHash,acceptanceTextHash,proposalHash []byte
	var contractKey string
	err := f.pool.QueryRow(ctx, `
		select c.id::text,o.legal_name,o.cnpj,pv.version_number,tv.version_number,
		       c.document_sha256,c.pdf_storage_key,c.generated_at,c.fully_signed_at,
		       pa.accepted_by_name,pa.accepted_by_email::text,pa.accepted_by_cpf,coalesce(pa.accepted_by_role,''),
		       pa.accepted_at,coalesce(pa.ip_address::text,''),coalesce(pa.user_agent,''),pa.session_id::text,
		       pa.acceptance_text_version,coalesce(pa.acceptance_text,''),coalesce(pa.acceptance_text_sha256,'\\x'::bytea),pa.proposal_hash
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposal_versions pv on pv.id=c.proposal_version_id
		join contract_template_versions tv on tv.id=c.template_version_id
		where c.id=$1 and c.status='signed' and c.fully_signed_at is not null
	`, contractID).Scan(
		&report.ContractID,&report.ClientLegalName,&report.ClientCNPJ,&report.ProposalVersion,&report.TemplateVersion,
		&documentHash,&contractKey,&report.GeneratedAt,&report.FullySignedAt,
		&report.Acceptance.Name,&report.Acceptance.Email,&report.Acceptance.CPF,&report.Acceptance.Role,
		&report.Acceptance.AcceptedAt,&report.Acceptance.IPAddress,&report.Acceptance.UserAgent,&report.Acceptance.SessionID,
		&report.Acceptance.TextVersion,&report.Acceptance.Text,&acceptanceTextHash,&proposalHash,
	)
	if err != nil { return EvidenceReport{}, "", fmt.Errorf("load signed contract: %w", err) }
	report.Version = "viagate-signature-evidence-v1"
	report.DocumentSHA256 = hex.EncodeToString(documentHash)
	report.Acceptance.TextSHA256=hex.EncodeToString(acceptanceTextHash)
	report.Acceptance.ProposalSHA256=hex.EncodeToString(proposalHash)
	report.IssuedAt = time.Now().UTC()
	report.CompanyLegalName = f.company.LegalName
	report.CompanyCNPJ = f.company.CNPJ

	rows, err := f.pool.Query(ctx, `
		select s.id::text,s.signer_type,s.name,s.email::text,s.cpf,coalesce(s.role,''),s.status::text,s.signed_at,
		       coalesce(s.signature_session_id::text,''),coalesce(s.signature_consent_version,''),coalesce(s.signature_consent_text,''),
		       coalesce(s.signature_consent_sha256,'\\x'::bytea),
		       exists(select 1 from identity_verifications i where i.contract_signer_id=s.id and i.mode='email_otp' and i.status='verified'),
		       exists(select 1 from identity_verifications i where i.contract_signer_id=s.id and i.mode='face' and i.status='verified'),
		       exists(select 1 from identity_verifications i where i.contract_signer_id=s.id and i.mode='liveness' and i.status='verified')
		from contract_signers s where s.contract_id=$1 order by s.sign_order,s.id
	`, contractID)
	if err != nil { return EvidenceReport{}, "", err }
	for rows.Next() {
		var signer EvidenceSigner
		var consentHash []byte
		if err := rows.Scan(&signer.ID,&signer.Type,&signer.Name,&signer.Email,&signer.CPF,&signer.Role,&signer.Status,&signer.SignedAt,&signer.SessionID,&signer.ConsentVersion,&signer.ConsentText,&consentHash,&signer.OTP,&signer.Face,&signer.Liveness); err != nil {
			rows.Close();return EvidenceReport{}, "", err
		}
		signer.ConsentSHA256=hex.EncodeToString(consentHash)
		report.Signers = append(report.Signers, signer)
	}
	if err := rows.Err(); err != nil { rows.Close();return EvidenceReport{}, "", err }
	rows.Close()

	events, err := f.pool.Query(ctx, `
		select e.event_type,coalesce(e.contract_signer_id::text,''),coalesce(e.ip_address::text,''),
		       coalesce(e.user_agent,''),coalesce(e.session_id::text,''),e.created_at,e.metadata
		from signature_events e where e.contract_id=$1 order by e.created_at,e.id
	`, contractID)
	if err != nil { return EvidenceReport{}, "", err }
	defer events.Close()
	for events.Next() {
		var event EvidenceEvent
		var metadataJSON []byte
		if err := events.Scan(&event.Type,&event.SignerID,&event.IPAddress,&event.UserAgent,&event.SessionID,&event.OccurredAt,&metadataJSON); err != nil { return EvidenceReport{}, "", err }
		_ = json.Unmarshal(metadataJSON, &event.Metadata)
		report.Events = append(report.Events, event)
	}
	return report, contractKey, events.Err()
}

func renderEvidenceHTML(report EvidenceReport) (string, error) {
	const page = `
	<h1>Relatório de evidências da assinatura</h1>
	<p><strong>Contrato:</strong> {{.ContractID}}</p>
	<p><strong>Contratante:</strong> {{.ClientLegalName}} · CNPJ {{cnpj .ClientCNPJ}}</p>
	<p><strong>SHA-256 do contrato:</strong><br><code>{{.DocumentSHA256}}</code></p>
	<p><strong>Versão da proposta:</strong> {{.ProposalVersion}} · <strong>Versão do modelo:</strong> {{.TemplateVersion}}</p>
	<p><strong>Contrato gerado em:</strong> {{time .GeneratedAt}} · <strong>Assinatura concluída em:</strong> {{time .FullySignedAt}}</p>
	<h2>Aceite da proposta</h2>
	<p><strong>{{.Acceptance.Name}}</strong> · CPF {{cpf .Acceptance.CPF}} · {{.Acceptance.Email}}</p>
	<p>Aceite em {{time .Acceptance.AcceptedAt}} · IP {{.Acceptance.IPAddress}} · sessão {{.Acceptance.SessionID}}</p>
	<p><strong>Hash da proposta aceita:</strong><br><code>{{.Acceptance.ProposalSHA256}}</code></p>
	<p><strong>Texto de aceite {{.Acceptance.TextVersion}}:</strong> {{.Acceptance.Text}}</p>
	<p><strong>SHA-256 do texto de aceite:</strong><br><code>{{.Acceptance.TextSHA256}}</code></p>
	<h2>Signatários</h2>
	<table><thead><tr><th>Nome</th><th>CPF</th><th>E-mail</th><th>Verificação</th><th>Assinado em</th></tr></thead><tbody>
	{{range .Signers}}<tr><td>{{.Name}}</td><td>{{cpf .CPF}}</td><td>{{.Email}}</td><td>{{if .OTP}}OTP e-mail{{end}}{{if .Face}} · Face{{end}}{{if .Liveness}} · Prova de vida{{end}}</td><td>{{timePtr .SignedAt}}</td></tr>{{end}}
	</tbody></table>
	{{range .Signers}}<p><strong>Consentimento de {{.Name}} ({{.ConsentVersion}}):</strong> {{.ConsentText}}<br><strong>SHA-256:</strong> <code>{{.ConsentSHA256}}</code></p>{{end}}
	<h2>Linha de evidências</h2>
	<table><thead><tr><th>Data/hora</th><th>Evento</th><th>IP</th><th>Sessão</th></tr></thead><tbody>
	{{range .Events}}<tr><td>{{time .OccurredAt}}</td><td>{{.Type}}</td><td>{{.IPAddress}}</td><td>{{.SessionID}}</td></tr>{{end}}
	</tbody></table>
	<p>Relatório emitido em {{time .IssuedAt}} por {{.CompanyLegalName}} · CNPJ {{cnpj .CompanyCNPJ}}.</p>`

	functions := template.FuncMap{
		"time": func(value time.Time) string { return value.UTC().Format("02/01/2006 15:04:05 MST") },
		"timePtr": func(value *time.Time) string { if value == nil { return "—" };return value.UTC().Format("02/01/2006 15:04:05 MST") },
		"cpf": brfields.FormatCPF,
		"cnpj": brfields.FormatCNPJ,
	}
	parsed, err := template.New("evidence").Funcs(functions).Parse(page)
	if err != nil { return "", err }
	var output bytes.Buffer
	if err := parsed.Execute(&output, report); err != nil { return "", err }
	return output.String(), nil
}

func zipEvidencePackage(contractPDF, evidencePDF, evidenceJSON, manifestJSON []byte) ([]byte, error) {
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	files := []struct { name string; data []byte }{
		{name: "contract.pdf", data: contractPDF},
		{name: "evidence.pdf", data: evidencePDF},
		{name: "evidence.json", data: evidenceJSON},
		{name: "manifest.json", data: manifestJSON},
	}
	for _, file := range files {
		entry, err := writer.Create(file.name)
		if err != nil { _ = writer.Close();return nil, err }
		if _, err := entry.Write(file.data); err != nil { _ = writer.Close();return nil, err }
	}
	if err := writer.Close(); err != nil { return nil, err }
	return output.Bytes(), nil
}

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}
