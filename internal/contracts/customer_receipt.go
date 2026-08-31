package contracts

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func (f *Finalizer) CustomerSignatureReceipt(ctx context.Context, contractID string) ([]byte, error) {
	report, _, err := f.loadReport(ctx, contractID)
	if err != nil {
		return nil, err
	}

	html, err := renderCustomerSignatureReceiptHTML(report)
	if err != nil {
		return nil, fmt.Errorf("render customer signature receipt: %w", err)
	}
	pdf, err := f.pdf.Render(ctx, html)
	if err != nil {
		return nil, fmt.Errorf("render customer signature receipt PDF: %w", err)
	}
	return pdf, nil
}

func renderCustomerSignatureReceiptHTML(report EvidenceReport) (string, error) {
	const page = `
	<h1>Comprovante de assinatura eletrônica</h1>
	<p>Este comprovante confirma a conclusão da assinatura eletrônica do contrato entre as partes abaixo.</p>

	<h2>Contratante</h2>
	<p><strong>{{.ClientLegalName}}</strong><br>CNPJ {{cnpj .ClientCNPJ}}</p>

	<h2>Responsável</h2>
	<p><strong>{{.Acceptance.Name}}</strong><br>CPF {{cpf .Acceptance.CPF}} · {{.Acceptance.Email}}{{if .Acceptance.Role}} · {{.Acceptance.Role}}{{end}}</p>

	<h2>Confirmações</h2>
	<p><strong>Proposta aceita em:</strong> {{time .Acceptance.AcceptedAt}}</p>
	<p><strong>Contrato assinado em:</strong> {{time .FullySignedAt}}</p>
	<p><strong>Confirmação de identidade:</strong> realizada por código enviado ao e-mail informado pelo responsável.</p>

	<p>A ViaGate mantém de forma segura os registros necessários para comprovar a autenticidade da assinatura e a integridade da versão do documento aceita pelas partes.</p>

	<p>Comprovante emitido em {{time .IssuedAt}} por {{.CompanyLegalName}} · CNPJ {{cnpj .CompanyCNPJ}}.</p>`

	functions := template.FuncMap{
		"time": func(value time.Time) string {
			return value.Format("02/01/2006 às 15:04")
		},
		"cpf":  brfields.FormatCPF,
		"cnpj": brfields.FormatCNPJ,
	}
	parsed, err := template.New("customer-signature-receipt").Funcs(functions).Parse(page)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, report); err != nil {
		return "", err
	}
	return output.String(), nil
}
