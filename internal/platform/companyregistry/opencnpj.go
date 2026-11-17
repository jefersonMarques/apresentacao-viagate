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

type OpenCNPJ struct {
    client   *http.Client
    endpoint string
}

func NewOpenCNPJ(timeout time.Duration) *OpenCNPJ {
    return &OpenCNPJ{
        client:   &http.Client{Timeout: timeout},
        endpoint: "https://kitana.opencnpj.com/cnpj/",
    }
}

func (o *OpenCNPJ) Lookup(ctx context.Context, cnpj string) (Company, error) {
    normalized, err := brtaxid.CleanCNPJ(cnpj)
    if err != nil {
        return Company{}, fmt.Errorf("invalid CNPJ: %w", err)
    }

    request, err := http.NewRequestWithContext(
        ctx, http.MethodGet, o.endpoint+url.PathEscape(normalized), nil,
    )
    if err != nil {
        return Company{}, fmt.Errorf("%w: create OpenCNPJ request: %v", ErrUnavailable, err)
    }
    request.Header.Set("Accept", "application/json")

    response, err := o.client.Do(request)
    if err != nil {
        return Company{}, fmt.Errorf("%w: OpenCNPJ request: %v", ErrUnavailable, err)
    }
    defer response.Body.Close()

    if response.StatusCode == http.StatusNotFound {
        return Company{}, fmt.Errorf("%w: OpenCNPJ returned HTTP 404", ErrNotFound)
    }
    if response.StatusCode != http.StatusOK {
        return Company{}, fmt.Errorf("%w: OpenCNPJ returned HTTP %d", ErrUnavailable, response.StatusCode)
    }

    var payload struct {
        Success bool   `json:"success"`
        Data    *struct {
            CNPJ        string `json:"cnpj"`
            LegalName   string `json:"razaoSocial"`
            TradeName   string `json:"nomeFantasia"`
            Email       string `json:"email"`
            Phone       string `json:"telefone"`
            Street      string `json:"logradouro"`
            Number      string `json:"numero"`
            Complement  string `json:"complemento"`
            District    string `json:"bairro"`
            City        string `json:"municipio"`
            State       string `json:"uf"`
            PostalCode  string `json:"cep"`
            Status      string `json:"situacaoCadastral"`
            CNAEs       []struct {
                Description string `json:"descricao"`
            } `json:"cnaes"`
        } `json:"data"`
    }

    if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
        return Company{}, fmt.Errorf("%w: decode OpenCNPJ response: %v", ErrUnavailable, err)
    }
    if !payload.Success || payload.Data == nil {
        return Company{}, fmt.Errorf("%w: OpenCNPJ returned an unsuccessful response", ErrUnavailable)
    }
    companyData := payload.Data
    if err := validateCompanyData(normalized, companyData.CNPJ, companyData.LegalName); err != nil {
        return Company{}, fmt.Errorf("%w: OpenCNPJ response: %v", ErrUnavailable, err)
    }

    primaryCNAE := ""
    if len(companyData.CNAEs) > 0 {
        primaryCNAE = strings.TrimSpace(companyData.CNAEs[0].Description)
    }

    return Company{
        CNPJ:        normalized,
        LegalName:   strings.TrimSpace(companyData.LegalName),
        TradeName:   strings.TrimSpace(companyData.TradeName),
        Email:       strings.TrimSpace(companyData.Email),
        Phone:       strings.TrimSpace(companyData.Phone),
        Street:      strings.TrimSpace(companyData.Street),
        Number:      strings.TrimSpace(companyData.Number),
        Complement:  strings.TrimSpace(companyData.Complement),
        District:    strings.TrimSpace(companyData.District),
        City:        strings.TrimSpace(companyData.City),
        State:       strings.TrimSpace(companyData.State),
        PostalCode:  strings.TrimSpace(companyData.PostalCode),
        Status:      strings.TrimSpace(companyData.Status),
        PrimaryCNAE: primaryCNAE,
    }, nil
}
