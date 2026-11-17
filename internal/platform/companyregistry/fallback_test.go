package companyregistry

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

const validTestCNPJ = "39406000000121"

type stubProvider struct {
    company Company
    err     error
    calls   int
}

func (s *stubProvider) Lookup(_ context.Context, _ string) (Company, error) {
    s.calls++
    return s.company, s.err
}

func TestFallbackProvider(t *testing.T) {
    cases := []struct {
        name          string
        primaryError  error
        fallbackError error
        primaryOK     bool
        expectNotFound bool
        expectUnavailable bool
        fallbackCalls int
    }{
        {
            name: "primary success", primaryOK: true,
            fallbackCalls: 0,
        },
        {
            name: "primary not found", primaryError: ErrNotFound,
            fallbackCalls: 1,
        },
        {
            name: "primary unavailable", primaryError: ErrUnavailable,
            fallbackCalls: 1,
        },
        {
            name: "both not found", primaryError: ErrNotFound,
            fallbackError: ErrNotFound, expectNotFound: true, fallbackCalls: 1,
        },
        {
            name: "primary not found fallback rate limited", primaryError: ErrNotFound,
            fallbackError: ErrUnavailable, expectUnavailable: true, fallbackCalls: 1,
        },
        {
            name: "both unavailable", primaryError: ErrUnavailable,
            fallbackError: ErrUnavailable, expectUnavailable: true, fallbackCalls: 1,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            primary := &stubProvider{company: Company{LegalName: "Primary Company"}, err: tc.primaryError}
            secondary := &stubProvider{company: Company{LegalName: "Fallback Company"}, err: tc.fallbackError}
            provider := NewFallbackProvider(primary, secondary, nil)

            company, err := provider.Lookup(context.Background(), validTestCNPJ)
            if secondary.calls != tc.fallbackCalls {
                t.Fatalf("fallback calls = %d, want %d", secondary.calls, tc.fallbackCalls)
            }
            if !errors.Is(err, ErrNotFound) && tc.expectNotFound {
                t.Fatalf("expected ErrNotFound, got %v", err)
            }
            if !errors.Is(err, ErrUnavailable) && tc.expectUnavailable {
                t.Fatalf("expected ErrUnavailable, got %v", err)
            }
            if tc.expectNotFound || tc.expectUnavailable {
                if company.LegalName != "" {
                    t.Fatalf("unexpected company on failed lookup: %+v", company)
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected lookup error: %v", err)
            }
            wantName := "Fallback Company"
            if tc.primaryOK {
                wantName = "Primary Company"
            }
            if company.LegalName != wantName {
                t.Fatalf("company name = %q, want %q", company.LegalName, wantName)
            }
        })
    }
}

func TestOpenCNPJResponseMapping(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/cnpj/"+validTestCNPJ {
            t.Errorf("unexpected lookup path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{
            "success": true,
            "data": {
                "cnpj": "39406000000121",
                "razaoSocial": "VIA GATEWAY LTDA",
                "nomeFantasia": "VIAGATE",
                "situacaoCadastral": "Ativa",
                "email": "finance@example.com",
                "telefone": "(41) 99999-0000",
                "logradouro": "RUA DE TESTE",
                "numero": "77",
                "complemento": "SALA 1",
                "bairro": "CENTRO",
                "municipio": "CURITIBA",
                "uf": "PR",
                "cep": "80000-000",
                "cnaes": [{"cnae": "6201501", "descricao": "Desenvolvimento de software"}]
            }
        }`))
    }))
    defer server.Close()

    provider := NewOpenCNPJ(2 * time.Second)
    provider.endpoint = server.URL + "/cnpj/"

    company, err := provider.Lookup(context.Background(), validTestCNPJ)
    if err != nil {
        t.Fatalf("OpenCNPJ lookup failed: %v", err)
    }
    if company.CNPJ != validTestCNPJ || company.LegalName != "VIA GATEWAY LTDA" {
        t.Errorf("wrong company identity: %+v", company)
    }
    if company.Street != "RUA DE TESTE" || company.Number != "77" ||
        company.PostalCode != "80000-000" || company.PrimaryCNAE != "Desenvolvimento de software" {
        t.Errorf("wrong company details: %+v", company)
    }
}

func TestOpenCNPJRejectsMismatchedCompany(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        _, _ = w.Write([]byte(`{"success":true,"data":{"cnpj":"00000000000191","razaoSocial":"WRONG COMPANY"}}`))
    }))
    defer server.Close()

    provider := NewOpenCNPJ(2 * time.Second)
    provider.endpoint = server.URL + "/cnpj/"

    _, err := provider.Lookup(context.Background(), validTestCNPJ)
    if !errors.Is(err, ErrUnavailable) {
        t.Fatalf("mismatched CNPJ must fail, got %v", err)
    }
}

func TestRegistryProviderStatusHandling(t *testing.T) {
    for _, providerName := range []string{"brasilapi", "opencnpj"} {
        for _, test := range []struct {
            status int
            expected error
        }{
            {http.StatusNotFound, ErrNotFound},
            {http.StatusTooManyRequests, ErrUnavailable},
            {http.StatusBadGateway, ErrUnavailable},
        } {
            t.Run(providerName+"-"+http.StatusText(test.status), func(t *testing.T) {
                server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
                    w.WriteHeader(test.status)
                }))
                defer server.Close()

                var provider Provider
                if providerName == "brasilapi" {
                    brasilAPI := NewBrasilAPI(time.Second)
                    brasilAPI.endpoint = server.URL + "/cnpj/"
                    provider = brasilAPI
                } else {
                    openCNPJ := NewOpenCNPJ(time.Second)
                    openCNPJ.endpoint = server.URL + "/cnpj/"
                    provider = openCNPJ
                }
                _, err := provider.Lookup(context.Background(), validTestCNPJ)
                if !errors.Is(err, test.expected) {
                    t.Fatalf("status %d returned %v; want %v", test.status, err, test.expected)
                }
            })
        }
    }
}

func TestBrasilAPIRejectsEmptyCompanyName(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        _, _ = w.Write([]byte(`{"cnpj":"39406000000121","razao_social":"  "}`))
    }))
    defer server.Close()
    provider := NewBrasilAPI(time.Second)
    provider.endpoint = server.URL + "/cnpj/"
    _, err := provider.Lookup(context.Background(), validTestCNPJ)
    if !errors.Is(err, ErrUnavailable) {
        t.Fatalf("empty company name must fail, got %v", err)
    }
}

func TestFallbackOnMalformedPrimaryResponse(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        _, _ = w.Write([]byte(strings.Repeat("{", 10)))
    }))
    defer server.Close()
    primary := NewBrasilAPI(time.Second)
    primary.endpoint = server.URL + "/cnpj/"
    secondary := &stubProvider{company: Company{CNPJ: validTestCNPJ, LegalName: "Valid backup"}}
    company, err := NewFallbackProvider(primary, secondary, nil).Lookup(context.Background(), validTestCNPJ)
    if err != nil || company.LegalName != "Valid backup" {
        t.Fatalf("fallback after malformed JSON failed: company=%+v err=%v", company, err)
    }
}
