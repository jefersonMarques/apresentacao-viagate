package httpapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

const maxCommercialImageBytes int64 = 2 << 20

var commercialImageCategories = map[string]bool{
	"salesperson_photo": true,
	"client_logo":       true,
}

var commercialImageExtensions = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

var commercialMediaPathPattern = regexp.MustCompile(`^/media/[0-9a-fA-F-]{36}$`)

func (a *App) uploadCommercialAsset(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if !commercialImageCategories[category] {
		http.Error(w, "categoria de imagem inválida", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCommercialImageBytes+(256<<10))
	if err := r.ParseMultipartForm(maxCommercialImageBytes); err != nil {
		http.Error(w, "a imagem deve ter no máximo 2 MB", http.StatusRequestEntityTooLarge)
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "selecione uma imagem", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxCommercialImageBytes+1))
	if err != nil || len(content) == 0 || int64(len(content)) > maxCommercialImageBytes {
		http.Error(w, "imagem inválida ou maior que 2 MB", http.StatusBadRequest)
		return
	}
	mimeType := http.DetectContentType(content[:min(512, len(content))])
	extension, allowed := commercialImageExtensions[mimeType]
	if !allowed {
		http.Error(w, "formato não permitido; envie JPG, PNG ou WEBP", http.StatusUnsupportedMediaType)
		return
	}

	assetID, err := newUUID()
	if err != nil {
		http.Error(w, "não foi possível preparar o upload", http.StatusInternalServerError)
		return
	}
	hash := sha256.Sum256(content)
	storageKey := fmt.Sprintf("commercial-assets/%s/%s/%s.%s", user.ID, category, assetID, extension)
	if err := a.storage.Put(r.Context(), storageKey, mimeType, bytes.NewReader(content), int64(len(content))); err != nil {
		a.logger.Error("upload commercial asset failed", "user_id", user.ID, "category", category, "error", err)
		http.Error(w, "não foi possível armazenar a imagem", http.StatusInternalServerError)
		return
	}

	_, err = a.pool.Exec(r.Context(), `
		insert into commercial_assets(id,owner_user_id,category,storage_key,original_filename,mime_type,size_bytes,sha256)
		values($1,$2,$3,$4,$5,$6,$7,$8)
	`, assetID, user.ID, category, storageKey, sanitizeFilename(header.Filename), mimeType, len(content), hash[:])
	if err != nil {
		_ = a.storage.Delete(r.Context(), storageKey)
		a.logger.Error("register commercial asset failed", "user_id", user.ID, "category", category, "error", err)
		http.Error(w, "não foi possível registrar a imagem", http.StatusInternalServerError)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'commercial_asset.uploaded','commercial_asset',$2,
		       jsonb_build_object('category',$3::text,'mime_type',$4::text,'size_bytes',$5::bigint))
	`, user.ID, assetID, category, mimeType, len(content))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id":  assetID,
		"url": "/media/" + assetID,
	})
}

func (a *App) commercialAsset(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "id")
	var storageKey, mimeType string
	var size int64
	if err := a.pool.QueryRow(r.Context(), `
		select storage_key,mime_type,size_bytes from commercial_assets where id=$1
	`, assetID).Scan(&storageKey, &mimeType, &size); err != nil {
		http.NotFound(w, r)
		return
	}

	body, objectSize, storedType, err := a.storage.Get(r.Context(), storageKey)
	if err != nil {
		a.logger.Error("load commercial asset failed", "asset_id", assetID, "error", err)
		http.Error(w, "imagem indisponível", http.StatusServiceUnavailable)
		return
	}
	defer body.Close()
	if storedType != "" {
		mimeType = storedType
	}
	if objectSize > 0 {
		size = objectSize
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, body)
}

func cleanCommercialImageURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if commercialMediaPathPattern.MatchString(value) {
		return value, nil
	}
	return cleanProfileURL(value)
}

func (a *App) adminLookupCNPJ(w http.ResponseWriter, r *http.Request) {
	cnpj, err := cleanCNPJ(chi.URLParam(r, "cnpj"))
	if err != nil {
		http.Error(w, "CNPJ inválido", http.StatusBadRequest)
		return
	}
	company, err := a.registry.Lookup(r.Context(), cnpj)
	if err != nil {
		http.Error(w, "CNPJ não encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"cnpj":         company.CNPJ,
		"legal_name":   company.LegalName,
		"trade_name":   company.TradeName,
		"email":        company.Email,
		"phone":        company.Phone,
		"street":       company.Street,
		"number":       company.Number,
		"complement":   company.Complement,
		"district":     company.District,
		"city":         company.City,
		"state":        company.State,
		"postal_code":  company.PostalCode,
		"status":       company.Status,
		"primary_cnae": company.PrimaryCNAE,
	})
}
