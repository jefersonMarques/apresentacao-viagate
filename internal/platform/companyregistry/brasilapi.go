package companyregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type Company struct {
	CNPJ          string
	LegalName     string
	TradeName     string
	Email         string
	Phone         string
	Street        string
	Number        string
	Complement    string
	District      string
	City          string
	State         string
	PostalCode    string
	Status        string
	PrimaryCNAE   string
}

type Provider interface {
	Lookup(ctx context.Context, cnpj string) (Company, error)
}

type BrasilAPI struct {
	client *http.Client
}

func NewBrasilAPI(timeout time.Duration) *BrasilAPI {
	return &BrasilAPI{client: &http.Client{Timeout: timeout}}
}

func (b *BrasilAPI) Lookup(ctx context.Context, cnpj string) (Company, error) {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(cnpj, "")
	if len(digits) != 14 {
		return Company{}, fmt.Errorf("invalid CNPJ")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://brasilapi.com.br/api/cnpj/v1/"+digits, nil)
	if err != nil {
		return Company{}, fmt.Errorf("create CNPJ request: %w", err)
	}

	resp, err := b.client.Do(req)
	if err != nil {
		return Company{}, fmt.Errorf("lookup CNPJ: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Company{}, fmt.Errorf("CNPJ not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Company{}, fmt.Errorf("CNPJ provider returned status %d", resp.StatusCode)
	}

	var raw struct {
		CNPJ               string `json:"cnpj"`
		RazaoSocial        string `json:"razao_social"`
		NomeFantasia       string `json:"nome_fantasia"`
		Email              string `json:"email"`
		DDDTelefone1       string `json:"ddd_telefone_1"`
		Logradouro         string `json:"logradouro"`
		Numero             string `json:"numero"`
		Complemento        string `json:"complemento"`
		Bairro             string `json:"bairro"`
		Municipio          string `json:"municipio"`
		UF                 string `json:"uf"`
		CEP                string `json:"cep"`
		DescricaoSituacao  string `json:"descricao_situacao_cadastral"`
		CNAEFiscalDescricao string `json:"cnae_fiscal_descricao"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return Company{}, fmt.Errorf("decode CNPJ response: %w", err)
	}

	return Company{
		CNPJ:        raw.CNPJ,
		LegalName:   strings.TrimSpace(raw.RazaoSocial),
		TradeName:   strings.TrimSpace(raw.NomeFantasia),
		Email:       strings.TrimSpace(raw.Email),
		Phone:       strings.TrimSpace(raw.DDDTelefone1),
		Street:      strings.TrimSpace(raw.Logradouro),
		Number:      strings.TrimSpace(raw.Numero),
		Complement:  strings.TrimSpace(raw.Complemento),
		District:    strings.TrimSpace(raw.Bairro),
		City:        strings.TrimSpace(raw.Municipio),
		State:       strings.TrimSpace(raw.UF),
		PostalCode:  strings.TrimSpace(raw.CEP),
		Status:      strings.TrimSpace(raw.DescricaoSituacao),
		PrimaryCNAE: strings.TrimSpace(raw.CNAEFiscalDescricao),
	}, nil
}
