package contracts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
			"pricing_model":   pricingModel,
			"pricing_table":   pricingData.PricingTable,
			"minimum_invoice": formatBRL(minimumInvoice),
			"setup_fee":       formatBRL(setupFee),
			"accepted_at":     acceptedAt.Format("02/01/2006 15:04"),
			"valid_until":     validUntilDisplay,
		},
		"pricing": pricingData.Prices,
		"operation": map[string]any{
			"type": operationType,
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
		return Generated{}, err
	}
	token, err := g.store.SignerPublicToken(ctx, signer.ID)
	if err != nil {
		return Generated{}, err
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

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}
