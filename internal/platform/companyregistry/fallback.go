package companyregistry

import (
    "errors"
    "fmt"
    "log/slog"
    "strings"
    "context"

    "github.com/jefersonMarques/apresentacao-viagate/internal/platform/brtaxid"
)

var (
    ErrNotFound    = errors.New("company not found")
    ErrUnavailable = errors.New("company registry unavailable")
)

type FallbackProvider struct {
    primary   Provider
    secondary Provider
    logger    *slog.Logger
}

func NewFallbackProvider(primary, secondary Provider, logger *slog.Logger) *FallbackProvider {
    return &FallbackProvider{
        primary:   primary,
        secondary: secondary,
        logger:    logger,
    }
}

func (f *FallbackProvider) Lookup(ctx context.Context, cnpj string) (Company, error) {
    company, primaryError := f.primary.Lookup(ctx, cnpj)
    if primaryError == nil {
        return company, nil
    }
    if ctx.Err() != nil {
        return Company{}, ctx.Err()
    }

    company, secondaryError := f.secondary.Lookup(ctx, cnpj)
    if secondaryError == nil {
        if f.logger != nil {
            f.logger.Info("company registry fallback succeeded", "provider", "opencnpj")
        }
        return company, nil
    }

    if errors.Is(primaryError, ErrNotFound) && errors.Is(secondaryError, ErrNotFound) {
        return Company{}, fmt.Errorf("%w: BrasilAPI and OpenCNPJ returned 404", ErrNotFound)
    }

    return Company{}, fmt.Errorf(
        "%w: BrasilAPI: %v; OpenCNPJ: %v",
        ErrUnavailable, primaryError, secondaryError,
    )
}

func validateCompanyData(requestedCNPJ, responseCNPJ, legalName string) error {
    if strings.TrimSpace(legalName) == "" {
        return fmt.Errorf("missing company legal name")
    }

    if strings.TrimSpace(responseCNPJ) == "" {
        return nil
    }

    normalized, err := brtaxid.CleanCNPJ(responseCNPJ)
    if err != nil {
        return fmt.Errorf("invalid CNPJ in provider response: %w", err)
    }
    if normalized != requestedCNPJ {
        return fmt.Errorf("provider response CNPJ differs from request")
    }
    return nil
}
