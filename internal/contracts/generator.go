package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	appconfig "github.com/jefersonMarques/apresentacao-viagate/internal/config"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/storage"
)

type Generator struct {
	pool     *pgxpool.Pool
	store    *Store
	renderer *Renderer
	pdf      *PDFRenderer
	storage  *storage.S3
	company  appconfig.CompanyConfig
}

type Generated struct {
	ContractID     string
	SignerID       string
	SignerToken    string
	SignerName     string
	SignerEmail    string
	DocumentSHA256 []byte
}

func NewGenerator(pool *pgxpool.Pool, store *Store, renderer *Renderer, pdf *PDFRenderer, storageClient *storage.S3, company appconfig.CompanyConfig) *Generator {
	return &Generator{pool: pool, store: store, renderer: renderer, pdf: pdf, storage: storageClient, company: company}
}

func (g *Generator) GenerateForOnboarding(ctx context.Context, onboardingID string) (Generated, error) {
	var proposalVersionID, createdBy, assignedTemplateVersionID string
	var legalName, tradeName, cnpj, street, number, complement, district, city, state, postalCode string
	var operationType, insurer, policyStartDate, policyEndDate, brokerCompany, brokerProducer string
	var repName, repCPF, repEmail, repPhone, repRole string
	var pricingModel string
	var minimumInvoice, setupFee float64
	var acceptedAt time.Time
	var validUntilSnapshot string

	err := g.pool.QueryRow(ctx, `
		select pv.id::text,p.created_by::text,
		       coalesce(pv.content #>> '{proposal,contract_template_version_id}',''),
		       o.legal_name,coalesce(o.trade_name,''),o.cnpj,coalesce(o.street,''),coalesce(o.street_number,''),
		       coalesce(o.complement,''),coalesce(o.district,''),coalesce(o.city,''),coalesce(o.state,''),coalesce(o.postal_code,''),
		       coalesce(o.operation_type,''),coalesce(o.insurer,''),coalesce(o.policy_start_date::text,''),coalesce(o.policy_end_date::text,''),
		       coalesce(o.broker_company,''),coalesce(o.broker_producer,''),
		       o.company_responsible_name,o.company_responsible_cpf,o.company_responsible_email::text,o.company_responsible_phone,coalesce(o.company_responsible_role,''),
		       pv.pricing_model,pv.minimum_invoice,pv.setup_fee,a.accepted_at,coalesce(pv.content #>> '{proposal,valid_until}','')
		from onboardings o
		join proposal_acceptances a on a.id=o.proposal_acceptance_id
		join proposal_versions pv on pv.id=a.proposal_version_id
		join proposals p on p.id=pv.proposal_id
		where o.id=$1 and o.status='approved'
	`, onboardingID).Scan(
		&proposalVersionID, &createdBy, &assignedTemplateVersionID,
		&legalName, &tradeName, &cnpj, &street, &number, &complement, &district, &city, &state, &postalCode,
		&operationType, &insurer, &policyStartDate, &policyEndDate, &brokerCompany, &brokerProducer,
		&repName, &repCPF, &repEmail, &repPhone, &repRole,
		&pricingModel, &minimumInvoice, &setupFee, &acceptedAt, &validUntilSnapshot,
	)
	if err != nil {
		return Generated{}, fmt.Errorf("load approved contract data: %w", err)
	}

	var templateVersionID string
	var templateMarkdown string
	if assignedTemplateVersionID != "" {
		err = g.pool.QueryRow(ctx, `
			select id::text,markdown
			from contract_template_versions
			where id=$1
		`, assignedTemplateVersionID).Scan(&templateVersionID, &templateMarkdown)
		if err != nil {
			return Generated{}, fmt.Errorf("assigned contract template version: %w", err)
		}
	} else {
		err = g.pool.QueryRow(ctx, `
			select v.id::text,v.markdown
			from contract_templates t
			join contract_template_versions v on v.contract_template_id=t.id and v.version_number=t.current_version
			where t.is_active=true and t.is_default=true
			limit 1
		`).Scan(&templateVersionID, &templateMarkdown)
		if err != nil {
			return Generated{}, fmt.Errorf("proposal has no assigned contract template and no default template is available: %w", err)
		}
	}

	templateMarkdown = ensureProposalFinancialTerms(templateMarkdown)

	pricingData, err := loadProposalPricing(ctx, g.pool, proposalVersionID)
	if err != nil {
		return Generated{}, err
	}

	formattedPostalCode := brfields.FormatPostalCode(postalCode)
	address := strings.TrimSpace(strings.Join(nonEmpty(street, number, complement, district, city, state, formattedPostalCode), ", "))
	validUntilDisplay := contractDate(validUntilSnapshot)
	data := Data{
		"client": map[string]any{
			"legal_name": legalName,
			"trade_name": tradeName,
			"cnpj":       brfields.FormatCNPJ(cnpj),
			"address":    address,
			"city":       city,
			"state":      state,
		},
		"representative": map[string]any{
			"name":  repName,
			"cpf":   brfields.FormatCPF(repCPF),
			"email": repEmail,
			"phone": brfields.FormatPhone(repPhone),
			"role":  repRole,
		},
		"proposal": map[string]any{
			"pricing_model":   contractPricingModelLabel(pricingModel),
			"pricing_table":   pricingData.PricingTable,
			"minimum_invoice": formatBRL(minimumInvoice),
			"setup_fee":       formatBRL(setupFee),
			"accepted_at":     acceptedAt.Format("02/01/2006 15:04"),
			"valid_until":     validUntilDisplay,
		},
		"pricing": pricingData.Prices,
		"operation": map[string]any{
			"type": contractOperationTypeLabel(operationType),
		},
		"insurance": map[string]any{
			"insurer":           insurer,
			"policy_start_date": contractDate(policyStartDate),
			"policy_end_date":   contractDate(policyEndDate),
			"broker_company":    brokerCompany,
			"broker_producer":   brokerProducer,
		},
		"viagate": map[string]any{
			"legal_name": g.company.LegalName,
			"cnpj":       brfields.FormatCNPJ(g.company.CNPJ),
		},
		"products": pricingData.Products,
	}

	renderedMarkdown, renderedHTML, err := g.renderer.Render(templateMarkdown, data)
	if err != nil {
		return Generated{}, err
	}
	pdfBytes, err := g.pdf.Render(ctx, renderedHTML)
	if err != nil {
		return Generated{}, err
	}
	documentHash := sha256.Sum256(pdfBytes)
	storageKey := fmt.Sprintf("contracts/%s/%d.pdf", onboardingID, time.Now().UTC().UnixNano())
	if err := g.storage.Put(ctx, storageKey, "application/pdf", bytes.NewReader(pdfBytes), int64(len(pdfBytes))); err != nil {
		return Generated{}, err
	}

	contract, err := g.store.CreateGeneratedContract(ctx, onboardingID, proposalVersionID, templateVersionID, renderedMarkdown, renderedHTML, storageKey, documentHash[:], createdBy)
	if err != nil {
		_ = g.storage.Delete(ctx, storageKey)
		return Generated{}, err
	}

	signer, err := g.store.AddSigner(ctx, contract.ID, "client", repName, repEmail, repCPF, repRole, 1)
	if err != nil {
		cleanupErr := g.cleanupGeneratedContract(ctx, contract.ID, storageKey)
		return Generated{}, joinGenerationError("add contract signer", err, cleanupErr)
	}

	token, err := g.store.SignerPublicToken(ctx, signer.ID)
	if err != nil {
		cleanupErr := g.cleanupGeneratedContract(ctx, contract.ID, storageKey)
		return Generated{}, joinGenerationError("load contract signer token", err, cleanupErr)
	}

	return Generated{
		ContractID:     contract.ID,
		SignerID:       signer.ID,
		SignerToken:    token,
		SignerName:     repName,
		SignerEmail:    repEmail,
		DocumentSHA256: documentHash[:],
	}, nil
}

func (g *Generator) cleanupGeneratedContract(ctx context.Context, contractID, storageKey string) error {
	var cleanupErrors []error
	if _, err := g.pool.Exec(ctx, `delete from contracts where id=$1 and status='generated'`, contractID); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("delete generated contract: %w", err))
	}
	if err := g.storage.Delete(ctx, storageKey); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("delete generated contract PDF: %w", err))
	}
	return errors.Join(cleanupErrors...)
}

func joinGenerationError(operation string, err, cleanupErr error) error {
	primary := fmt.Errorf("%s: %w", operation, err)
	if cleanupErr == nil {
		return primary
	}
	return errors.Join(primary, cleanupErr)
}

func contractDate(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return parsed.Format("02/01/2006")
}

func contractPricingModelLabel(value string) string {
	for _, model := range catalog.PricingModels {
		if model.ID == value {
			return model.Title
		}
	}
	if strings.TrimSpace(value) == "" {
		return "Não informado"
	}
	return "Condição comercial personalizada"
}

func contractOperationTypeLabel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "normal":
		return "Normal"
	case "avulsa":
		return "Avulsa"
	case "avulso":
		return "Avulsa"
	default:
		if strings.TrimSpace(value) == "" {
			return "Não informada"
		}
		return strings.TrimSpace(value)
	}
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}
