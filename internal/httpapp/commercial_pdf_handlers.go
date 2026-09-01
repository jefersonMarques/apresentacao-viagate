package httpapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) downloadProposalPDF(w http.ResponseWriter, r *http.Request, items []domain.Proposal, proposalID string) {
	var token string
	for _, item := range items {
		if item.ID == proposalID {
			token = item.PublicToken
			break
		}
	}
	if token == "" {
		http.Error(w, "Publique a proposta antes de gerar o PDF.", http.StatusConflict)
		return
	}

	proposal, err := a.proposalStore.PublishedByToken(r.Context(), token)
	if err != nil {
		a.logger.Error("load published proposal for PDF failed", "proposal_id", proposalID, "error", err)
		http.Error(w, "Não foi possível carregar a versão publicada da proposta.", http.StatusInternalServerError)
		return
	}

	key := "commercial-pdf/proposals/" + proposal.VersionID + ".pdf"
	filename := commercialPDFFilename("proposta", proposal.ClientTradeName, proposal.ClientName, fmt.Sprintf("v%d", proposal.VersionNumber))
	a.serveCommercialPDF(w, r, key, filename, func(ctx context.Context) (string, error) {
		var buffer bytes.Buffer
		if err := templates.PublicProposalViewerPage(proposal, "").Render(ctx, &buffer); err != nil {
			return "", fmt.Errorf("render proposal HTML: %w", err)
		}
		return a.prepareProposalPDFDocument(buffer.String()), nil
	})
}

func (a *App) downloadPresentationPDF(w http.ResponseWriter, r *http.Request, items []presentations.Presentation, presentationID string) {
	var token string
	for _, item := range items {
		if item.ID == presentationID {
			token = item.PublicToken
			break
		}
	}
	if token == "" {
		http.Error(w, "Publique a apresentação antes de gerar o PDF.", http.StatusConflict)
		return
	}

	presentation, err := a.presentationStore.PublishedByToken(r.Context(), token)
	if err != nil {
		a.logger.Error("load published presentation for PDF failed", "presentation_id", presentationID, "error", err)
		http.Error(w, "Não foi possível carregar a versão publicada da apresentação.", http.StatusInternalServerError)
		return
	}

	key := "commercial-pdf/presentations/" + presentation.VersionID + ".pdf"
	filename := commercialPDFFilename("apresentacao", presentationClientFilename(presentation), fmt.Sprintf("v%d", presentation.VersionNumber))
	a.serveCommercialPDF(w, r, key, filename, func(ctx context.Context) (string, error) {
		return a.preparePresentationPDFDocument(presentation)
	})
}

func (a *App) serveCommercialPDF(w http.ResponseWriter, r *http.Request, key, filename string, buildDocument func(context.Context) (string, error)) {
	if a.storage != nil {
		body, size, _, err := a.storage.Get(r.Context(), key)
		if err == nil {
			defer body.Close()
			writePDFHeaders(w, filename, size)
			if _, copyErr := io.Copy(w, body); copyErr != nil {
				a.logger.Warn("stream cached commercial PDF failed", "key", key, "error", copyErr)
			}
			return
		}
	}

	document, err := buildDocument(r.Context())
	if err != nil {
		a.logger.Error("build commercial PDF document failed", "key", key, "error", err)
		http.Error(w, "Não foi possível preparar o PDF.", http.StatusInternalServerError)
		return
	}

	renderer := contracts.NewPDFRenderer(a.cfg.ChromiumPath)
	pdf, err := renderer.RenderDocument(r.Context(), document)
	if err != nil {
		a.logger.Error("render commercial PDF failed", "key", key, "error", err)
		http.Error(w, "Não foi possível gerar o PDF. Verifique a instalação do Chrome/Edge/Chromium no servidor.", http.StatusInternalServerError)
		return
	}

	if a.storage != nil {
		if err := a.storage.Put(r.Context(), key, "application/pdf", bytes.NewReader(pdf), int64(len(pdf))); err != nil {
			a.logger.Warn("cache commercial PDF failed", "key", key, "error", err)
		}
	}
	writePDFHeaders(w, filename, int64(len(pdf)))
	_, _ = w.Write(pdf)
}

func (a *App) prepareProposalPDFDocument(document string) string {
	baseURL := strings.TrimRight(a.cfg.BaseURL, "/") + "/"
	document = strings.Replace(document, "<head>", `<head><base href="`+html.EscapeString(baseURL)+`"/>`, 1)
	document = strings.Replace(document, "</head>", `<link rel="stylesheet" href="/assets/commercial-pdf.css"/></head>`, 1)
	pdfRuntime := `<script>window.__VIAGATE_PDF_MODE__=true;</script><script src="/assets/commercial-pdf.js"></script>`
	document = strings.Replace(document, "</body>", pdfRuntime+"</body>", 1)
	return document
}

func (a *App) preparePresentationPDFDocument(p presentations.PublicPresentation) (string, error) {
	content, err := os.ReadFile("presentation-content.html")
	if err != nil {
		return "", fmt.Errorf("read presentation content: %w", err)
	}
	baseURL := strings.TrimRight(a.cfg.BaseURL, "/")
	document := string(content)
	baseTag := `<base href="` + html.EscapeString(baseURL) + `/v1/" />`
	document = strings.Replace(document, `<base href="/apresentacao/" />`, baseTag, 1)
	document = strings.Replace(document, `<base href="/apresentacao/">`, baseTag, 1)
	document = strings.Replace(document, "</head>", `<link rel="stylesheet" href="`+html.EscapeString(baseURL)+`/assets/commercial-pdf.css"/></head>`, 1)

	contact := map[string]any{
		"name": p.SalespersonName,
		"role": p.SalespersonJobTitle,
		"email": p.SalespersonEmail,
		"phone": templates.Phone(p.SalespersonPhone),
		"whatsapp": digitsOnly(p.SalespersonPhone),
		"linkedin": p.SalespersonLinkedIn,
		"instagram": p.SalespersonInstagram,
		"photoUrl": p.SalespersonPhotoURL,
	}
	settings := map[string]any{
		"showContactSlide": p.ShowContactSlide,
		"showClientIdentity": p.ShowClientIdentity,
		"client": map[string]string{
			"company_name": presentationClientFilename(p),
			"contact_name": p.ContactName,
			"logo_url": p.ClientLogoURL,
		},
	}
	contactJSON, _ := json.Marshal(contact)
	settingsJSON, _ := json.Marshal(settings)
	modulesJSON, _ := json.Marshal(p.SelectedModules)

	configuration := `<script>window.__VIAGATE_PDF_MODE__=true;window.presentationContact=Object.freeze(` + string(contactJSON) + `);window.presentationSettings=Object.freeze(` + string(settingsJSON) + `);</script>` +
		`<script src="` + html.EscapeString(baseURL) + `/assets/commercial-pdf.js"></script>`
	document = strings.Replace(document, "<body>", "<body>"+configuration, 1)

	bootstrap := `<script>(function(){
const selected=new Set(` + string(modulesJSON) + `);
const slideModules={'slide-03':['score'],'slide-04':['score'],'slide-05':['score'],'slide-06':['score'],'slide-07':['authentication','prevention'],'slide-08':['authentication'],'slide-10':['logistics','monitoring'],'slide-11':['logistics','monitoring']};
function filterSlides(){if(selected.size===0)return;Object.entries(slideModules).forEach(function(entry){const slide=document.getElementById(entry[0]);if(slide&&!entry[1].some(function(module){return selected.has(module);})){slide.remove();}});}
function waitBridge(attempt){if(typeof window.hostStartPresentation==='function'){window.hostStartPresentation(true);document.documentElement.classList.add('commercial-pdf-mode');document.body.classList.add('commercial-pdf-mode');window.dispatchEvent(new Event('commercial-pdf-ready'));return;}if(attempt<160)window.setTimeout(function(){waitBridge(attempt+1);},50);}
function initialize(attempt){if(!document.querySelector('.exec-analysis-metrics-slide')&&attempt<160){window.setTimeout(function(){initialize(attempt+1);},50);return;}filterSlides();const script=document.createElement('script');script.src='` + html.EscapeString(baseURL) + `/v1/presentation-bootstrap.js';script.addEventListener('load',function(){waitBridge(0);},{once:true});document.body.appendChild(script);}
window.addEventListener('load',function(){initialize(0);},{once:true});
})();</script>`
	document = strings.Replace(document, "</body>", bootstrap+"</body>", 1)
	return document, nil
}

func writePDFHeaders(w http.ResponseWriter, filename string, size int64) {
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Cache-Control", "private, no-store")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
}

func commercialPDFFilename(parts ...string) string {
	joined := strings.Join(parts, "-")
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(joined) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		name = "viagate"
	}
	return name + ".pdf"
}

func presentationClientFilename(p presentations.PublicPresentation) string {
	if value := strings.TrimSpace(p.ClientTradeName); value != "" {
		return value
	}
	if value := strings.TrimSpace(p.ClientLegalName); value != "" {
		return value
	}
	return "institucional"
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
