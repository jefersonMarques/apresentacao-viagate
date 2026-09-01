package contracts

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type PDFRenderer struct {
	chromiumPath string
}

func NewPDFRenderer(chromiumPath string) *PDFRenderer {
	return &PDFRenderer{chromiumPath: chromiumPath}
}

func (r *PDFRenderer) Render(ctx context.Context, html string) ([]byte, error) {
	browserPath, err := resolveBrowserExecutable(r.chromiumPath)
	if err != nil {
		return nil, err
	}

	html, err = InjectVerificationBlock(html)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "viagate-contract-*")
	if err != nil {
		return nil, fmt.Errorf("create contract temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	htmlPath := filepath.Join(dir, "contract.html")
	pdfPath := filepath.Join(dir, "contract.pdf")

	document := `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><style>
		@page{size:A4;margin:18mm 16mm}body{font-family:Arial,sans-serif;color:#18212a;font-size:11pt;line-height:1.55}h1,h2,h3{color:#071827;page-break-after:avoid}h1{font-size:20pt}h2{font-size:15pt;margin-top:22pt}p,li{orphans:3;widows:3}table{width:100%;border-collapse:collapse}td,th{border:1px solid #d8dee3;padding:6px;text-align:left}.signature-evidence{page-break-before:always}
		.contract-verification{margin-top:28pt;padding-top:16pt;border-top:1px solid #cfd9df;display:flex;align-items:center;justify-content:space-between;gap:18pt;page-break-inside:avoid}.contract-verification-copy{flex:1;min-width:0}.contract-verification-copy h2{margin:0 0 6pt;font-size:13pt}.contract-verification-copy p{margin:4pt 0;font-size:9pt;color:#526773}.contract-verification-url{word-break:break-all}.contract-verification-qr{width:34mm;height:34mm;object-fit:contain;flex:0 0 34mm}
	</style></head><body>` + html + `</body></html>`

	if err := os.WriteFile(htmlPath, []byte(document), 0o600); err != nil {
		return nil, fmt.Errorf("write contract html: %w", err)
	}

	command := exec.CommandContext(ctx, browserPath,
		"--headless",
		"--disable-gpu",
		"--no-pdf-header-footer",
		"--print-to-pdf="+pdfPath,
		fileURL(htmlPath),
	)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("render contract PDF with %q: %w: %s", browserPath, err, string(output))
	}

	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read generated contract PDF: %w", err)
	}
	if len(pdf) == 0 {
		return nil, fmt.Errorf("generated contract PDF is empty")
	}
	return pdf, nil
}

// RenderURL renders a browser-accessible page entirely on the server. The page
// is responsible for its own @page rules; commercial slides use a fixed 16:9
// page while contracts continue to use Render and the A4 document model above.
func (r *PDFRenderer) RenderURL(ctx context.Context, sourceURL string) ([]byte, error) {
	browserPath, err := resolveBrowserExecutable(r.chromiumPath)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("PDF source URL is required")
	}
	if _, err := url.ParseRequestURI(sourceURL); err != nil {
		return nil, fmt.Errorf("invalid PDF source URL: %w", err)
	}

	dir, err := os.MkdirTemp("", "viagate-commercial-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("create commercial PDF temp dir: %w", err)
	}
	defer os.RemoveAll(dir)
	pdfPath := filepath.Join(dir, "document.pdf")

	command := exec.CommandContext(ctx, browserPath,
		"--headless",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--no-pdf-header-footer",
		"--window-size=1280,720",
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=8000",
		"--print-to-pdf="+pdfPath,
		sourceURL,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("render URL PDF with %q: %w: %s", browserPath, err, string(output))
	}
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("read generated URL PDF: %w", err)
	}
	if len(pdf) == 0 {
		return nil, fmt.Errorf("generated URL PDF is empty")
	}
	return pdf, nil
}

func resolveBrowserExecutable(configuredPath string) (string, error) {
	configuredPath = strings.Trim(strings.TrimSpace(configuredPath), `"'`)
	candidates := browserCandidates(configuredPath)

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		if filepath.IsAbs(candidate) || strings.ContainsAny(candidate, `/\\`) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
			continue
		}

		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf(
		"browser executable not found; CHROMIUM_PATH=%q. Install Chrome, Edge or Chromium, or configure CHROMIUM_PATH with a valid executable path",
		configuredPath,
	)
}

func browserCandidates(configuredPath string) []string {
	candidates := []string{configuredPath}

	switch runtime.GOOS {
	case "windows":
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")
		localAppData := os.Getenv("LOCALAPPDATA")

		candidates = append(candidates,
			filepath.Join(programFiles, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFilesX86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(localAppData, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(programFiles, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(programFilesX86, "Microsoft", "Edge", "Application", "msedge.exe"),
			"chrome.exe",
			"msedge.exe",
			"chromium.exe",
		)
	case "darwin":
		candidates = append(candidates,
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"google-chrome",
			"chromium",
		)
	default:
		candidates = append(candidates,
			"chromium",
			"chromium-browser",
			"google-chrome",
			"google-chrome-stable",
			"microsoft-edge",
			"microsoft-edge-stable",
		)
	}

	return candidates
}

func fileURL(path string) string {
	urlPath := filepath.ToSlash(path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	return (&url.URL{Scheme: "file", Path: urlPath}).String()
}
