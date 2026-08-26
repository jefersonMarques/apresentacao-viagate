package contracts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type PDFRenderer struct {
	chromiumPath string
}

func NewPDFRenderer(chromiumPath string) *PDFRenderer {
	return &PDFRenderer{chromiumPath: chromiumPath}
}

func (r *PDFRenderer) Render(ctx context.Context, html string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "viagate-contract-*")
	if err != nil {
		return nil, fmt.Errorf("create contract temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	htmlPath := filepath.Join(dir, "contract.html")
	pdfPath := filepath.Join(dir, "contract.pdf")

	document := `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><style>
		@page{size:A4;margin:18mm 16mm}body{font-family:Arial,sans-serif;color:#18212a;font-size:11pt;line-height:1.55}h1,h2,h3{color:#071827;page-break-after:avoid}h1{font-size:20pt}h2{font-size:15pt;margin-top:22pt}p,li{orphans:3;widows:3}table{width:100%;border-collapse:collapse}td,th{border:1px solid #d8dee3;padding:6px;text-align:left}.signature-evidence{page-break-before:always}
	</style></head><body>` + html + `</body></html>`

	if err := os.WriteFile(htmlPath, []byte(document), 0o600); err != nil {
		return nil, fmt.Errorf("write contract html: %w", err)
	}

	command := exec.CommandContext(ctx, r.chromiumPath,
		"--headless",
		"--disable-gpu",
		"--no-pdf-header-footer",
		"--print-to-pdf="+pdfPath,
		"file://"+htmlPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("render contract PDF: %w: %s", err, string(output))
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
