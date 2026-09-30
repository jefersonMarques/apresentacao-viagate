package httpapp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"time"

	onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
)

const maxInsurancePolicyBytes int64 = 15 << 20

type policyUploadError struct {
	Status  int
	Message string
}

func (e *policyUploadError) Error() string {
	return e.Message
}

func policyUploadResponse(err error) (int, string) {
	if uploadErr, ok := err.(*policyUploadError); ok {
		return uploadErr.Status, uploadErr.Message
	}
	return http.StatusInternalServerError, "Não foi possível enviar a apólice."
}

func (a *App) storeInsurancePolicyFromRequest(r *http.Request, onboardingID string) error {
	if err := r.ParseMultipartForm(maxInsurancePolicyBytes); err != nil {
		return &policyUploadError{Status: http.StatusRequestEntityTooLarge, Message: "Arquivo excede o limite de 15 MB."}
	}

	file, header, err := r.FormFile("document")
	if err != nil {
		return &policyUploadError{Status: http.StatusBadRequest, Message: "Selecione a apólice."}
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxInsurancePolicyBytes+1))
	if err != nil || len(content) == 0 || int64(len(content)) > maxInsurancePolicyBytes {
		return &policyUploadError{Status: http.StatusBadRequest, Message: "Arquivo inválido."}
	}

	mimeType := http.DetectContentType(content[:min(512, len(content))])
	allowed := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
	}
	if !allowed[mimeType] {
		return &policyUploadError{Status: http.StatusUnsupportedMediaType, Message: "Formato não permitido; envie PDF, JPG ou PNG."}
	}

	hash := sha256.Sum256(content)
	key := fmt.Sprintf("onboarding/%s/insurance_policy/%d-%s", onboardingID, time.Now().UTC().UnixNano(), sanitizeFilename(header.Filename))
	if err := a.storage.Put(r.Context(), key, mimeType, bytes.NewReader(content), int64(len(content))); err != nil {
		a.logger.Error("upload policy to S3 failed", "onboarding_id", onboardingID, "error", err)
		return err
	}

	document := onboardingpkg.Document{
		DocumentType:     "insurance_policy",
		StorageKey:       key,
		OriginalFilename: header.Filename,
		MIMEType:         mimeType,
		SizeBytes:        int64(len(content)),
		SHA256:           hash[:],
	}
	if err := a.onboardingStore.AddDocument(r.Context(), onboardingID, document); err != nil {
		_ = a.storage.Delete(r.Context(), key)
		return err
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values(
			'customer','document.uploaded','onboarding',$1,$2,$3,
			jsonb_build_object('type','insurance_policy','sha256',$4,'filename',$5)
		)
	`, onboardingID, requestIP(r), r.UserAgent(), fmt.Sprintf("%x", hash[:]), header.Filename)

	return nil
}
