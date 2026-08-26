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
	var templateVersionID string
	var templateMarkdown string
	err := g.pool.QueryRow(ctx, `
		select v.id::text,v.markdown
		from contract_templates t
		join contract_template_versions v on v.contract_template_id=t.id and v.version_number=t.current_version
		where t.is_active=true and t.is_default=true
		limit 1
	`).Scan(&templateVersionID,&templateMarkdown)
	if err != nil {
		return Generated{}, fmt.Errorf("default contract template: %w",err)
	}

	var proposalVersionID,createdBy string
	var legalName,tradeName,cnpj,street,number,complement,district,city,state,postalCode string
	var repName,repCPF,repEmail,repPhone,repRole string
	var minimumInvoice,setupFee float64
	var acceptedAt time.Time
	var validUntil *time.Time
	err = g.pool.QueryRow(ctx, `
		select pv.id::text,p.created_by::text,
		       o.legal_name,coalesce(o.trade_name,''),o.cnpj,coalesce(o.street,''),coalesce(o.street_number,''),
		       coalesce(o.complement,''),coalesce(o.district,''),coalesce(o.city,''),coalesce(o.state,''),coalesce(o.postal_code,''),
		       o.company_responsible_name,o.company_responsible_cpf,o.company_responsible_email::text,o.company_responsible_phone,coalesce(o.company_responsible_role,''),
		       pv.minimum_invoice,pv.setup_fee,a.accepted_at,p.valid_until
		from onboardings o
		join proposal_acceptances a on a.id=o.proposal_acceptance_id
		join proposal_versions pv on pv.id=a.proposal_version_id
		join proposals p on p.id=pv.proposal_id
		where o.id=$1 and o.status='submitted'
	`,onboardingID).Scan(
		&proposalVersionID,&createdBy,&legalName,&tradeName,&cnpj,&street,&number,&complement,&district,&city,&state,&postalCode,
		&repName,&repCPF,&repEmail,&repPhone,&repRole,&minimumInvoice,&setupFee,&acceptedAt,&validUntil,
	)
	if err != nil { return Generated{},fmt.Errorf("load contract data: %w",err) }

	products := map[string]any{
		"cargo_score": false,
		"cargo_truck": false,
		"prevention": false,
		"monitoring": false,
	}
	rows,err := g.pool.Query(ctx,`select distinct lower(group_name) from proposal_items where proposal_version_id=$1 and is_optional=false`,proposalVersionID)
	if err != nil{return Generated{},err}
	for rows.Next(){
		var group string
		if err:=rows.Scan(&group);err!=nil{rows.Close();return Generated{},err}
		switch {
		case strings.Contains(group,"score"):
			products["cargo_score"]=true
		case strings.Contains(group,"logística") || strings.Contains(group,"logistica") || strings.Contains(group,"truck"):
			products["cargo_truck"]=true
		case strings.Contains(group,"preven"):
			products["prevention"]=true
		case strings.Contains(group,"monitoramento"):
			products["monitoring"]=true
		}
	}
	rows.Close()

	address := strings.TrimSpace(strings.Join(nonEmpty(street,number,complement,district,city,state,postalCode),", "))
	data := Data{
		"client":map[string]any{
			"legal_name":legalName,"trade_name":tradeName,"cnpj":cnpj,"address":address,"city":city,"state":state,
		},
		"representative":map[string]any{
			"name":repName,"cpf":repCPF,"email":repEmail,"phone":repPhone,"role":repRole,
		},
		"proposal":map[string]any{
			"minimum_invoice":fmt.Sprintf("R$ %.2f",minimumInvoice),
			"setup_fee":fmt.Sprintf("R$ %.2f",setupFee),
			"accepted_at":acceptedAt.Format("02/01/2006 15:04"),
			"valid_until":formatDate(validUntil),
		},
		"viagate":map[string]any{
			"legal_name":g.company.LegalName,
			"cnpj":g.company.CNPJ,
		},
		"products":products,
	}

	renderedMarkdown,renderedHTML,err := g.renderer.Render(templateMarkdown,data)
	if err != nil{return Generated{},err}
	pdfBytes,err := g.pdf.Render(ctx,renderedHTML)
	if err != nil{return Generated{},err}
	documentHash := sha256.Sum256(pdfBytes)
	storageKey := fmt.Sprintf("contracts/%s/%d.pdf",onboardingID,time.Now().UTC().UnixNano())
	if err:=g.storage.Put(ctx,storageKey,"application/pdf",bytes.NewReader(pdfBytes),int64(len(pdfBytes)));err!=nil{return Generated{},err}

	contract,err := g.store.CreateGeneratedContract(ctx,onboardingID,proposalVersionID,templateVersionID,renderedMarkdown,renderedHTML,storageKey,documentHash[:],createdBy)
	if err != nil {
		_ = g.storage.Delete(ctx,storageKey)
		return Generated{},err
	}
	signer,err := g.store.AddSigner(ctx,contract.ID,"client",repName,repEmail,repCPF,repRole,1)
	if err != nil{return Generated{},err}
	token,err := g.store.SignerPublicToken(ctx,signer.ID)
	if err != nil{return Generated{},err}
	return Generated{
		ContractID:contract.ID,SignerID:signer.ID,SignerToken:token,SignerName:repName,SignerEmail:repEmail,DocumentSHA256:documentHash[:],
	},nil
}

func nonEmpty(values ...string) []string {
	result:=make([]string,0,len(values))
	for _,value:=range values { if strings.TrimSpace(value)!=""{result=append(result,strings.TrimSpace(value))} }
	return result
}

func formatDate(value *time.Time) string { if value==nil{return ""}; return value.Format("02/01/2006") }
