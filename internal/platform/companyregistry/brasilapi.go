package companyregistry

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/jefersonMarques/apresentacao-viagate/internal/platform/brtaxid"
)

type Company struct {
    CNPJ        string
    LegalName   string
    TradeName   string
    Email       string
    Phone       string
    Street      string
    Number      string
    Complement  string
    District    string
    City        string
    State       string
    PostalCode  string
    Status      string
    PrimaryCNAE string
}

type Provider interface {
    Lookup(ctx context.Context, cnpj string) (Company, error)
}

type BrasilAPI struct {
    client   *http.Client
    endpoint string
}

func NewBrasilAPI(timeout time.Duration) *BrasilAPI {
    return &BrasilAPI{
        client:   &http.Client{Timeout: timeout},
        endpoint: "https://brasilapi.com.br/api/cnpj/v1/",
    }
}

func (b *BrasilAPI) Lookup(ctx context.Context, cnpj string) (Company, error) {
    normalized, err := brtaxid.CleanCNPJ(cnpj)
    if err != nil {
        return Company{}, fmt.Errorf("invalid CNPJ: %w", err)
    }

    endpoint := b.endpoint + url.PathEscape(normalized)
    request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
    if err != nil {
        return Company{}, fmt.Errorf("%w: create BrasilAPI request: %v", ErrUnavailable, err)
    }
    request.Header.Set("Accept", "application/json")

    response, err := b.client.Do(request)
    if err != nil {
        return Company{}, fmt.Errorf("%w: BrasilAPI request: %v", ErrUnavailable, err)
    }
    defer response.Body.Close()

    if response.StatusCode == http.StatusNotFound {
        return Company{}, fmt.Errorf("%w: BrasilAPI returned HTTP 404", ErrNotFound)
    }
    if response.StatusCode != http.StatusOK {
        return Company{}, fmt.Errorf("%w: BrasilAPI returned HTTP %d", ErrUnavailable, response.StatusCode)
    }

    var payload struct {
        CNPJ                 string `json:"cnpj"`
        LegalName            string `json:"razao_social"`
        TradeName            string `json:"nome_fantasia"`
        Email                string `json:"email"`
        Phone                string `json:"ddd_telefone_1"`
        Street               string `json:"logradouro"`
        Number               string `json:"numero"`
        Complement           string `json:"complemento"`
        District             string `json:"bairro"`
        City                 string `json:"municipio"`
        State                string `json:"uf"`
        PostalCode           string `json:"cep"`
        Status               string `json:"descricao_situacao_cadastral"`
        PrimaryCNAE          string `json:"cnae_fiscal_descricao"`
    }

    if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
        return Company{}, fmt.Errorf("%w: decode BrasilAPI response: %v", ErrUnavailable, err)
    }
    if err := validateCompanyData(normalized, payload.CNPJ, payload.LegalName); err != nil {
        return Company{}, fmt.Errorf("%w: BrasilAPI response: %v", ErrUnavailable, err)
    }

    return Company{
        CNPJ:        normalized,
        LegalName:   strings.TrimSpace(payload.LegalName),
        TradeName:   strings.TrimSpace(payload.TradeName),
        Email:       strings.TrimSpace(payload.Email),
        Phone:       strings.TrimSpace(payload.Phone),
        Street:      strings.TrimSpace(payload.Street),
        Number:      strings.TrimSpace(payload.Number),
        Complement:  strings.TrimSpace(payload.Complement),
        District:    strings.TrimSpace(payload.District),
        City:        strings.TrimSpace(payload.City),
        State:       strings.TrimSpace(payload.State),
        PostalCode:  strings.TrimSpace(payload.PostalCode),
        Status:      strings.TrimSpace(payload.Status),
        PrimaryCNAE: strings.TrimSpace(payload.PrimaryCNAE),
    }, nil
}
