package contracts

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
)

func TestBuildRasterPDFCreatesVariableHeightImagePages(t *testing.T) {
	pageA := testRasterPage(t, 120, 68)
	pageB := testRasterPage(t, 120, 180)

	pdf, err := buildRasterPDF([]rasterPDFPage{pageA, pageB})
	if err != nil {
		t.Fatalf("build raster PDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatalf("unexpected PDF header: %q", pdf[:min(len(pdf), 12)])
	}

	text := string(pdf)
	if !strings.Contains(text, "/Count 2") {
		t.Fatalf("expected two pages in PDF")
	}
	if strings.Count(text, "/Subtype /Image") != 2 {
		t.Fatalf("expected one image object per page")
	}
	if strings.Count(text, "/MediaBox") != 2 {
		t.Fatalf("expected one media box per page")
	}
}

func testRasterPage(t *testing.T, width, height int) rasterPDFPage {
	t.Helper()
	imageData := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			imageData.Set(x, y, color.RGBA{R: 20, G: 60, B: 90, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, imageData, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode test JPEG: %v", err)
	}
	return rasterPDFPage{
		JPEG:        buffer.Bytes(),
		PixelWidth:  width,
		PixelHeight: height,
	}
}
