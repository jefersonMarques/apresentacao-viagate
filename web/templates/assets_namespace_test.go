package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommercialTemplatesDoNotUseReservedSiteAssetsNamespace(t *testing.T) {
	files, err := filepath.Glob("*.templ")
	if err != nil {
		t.Fatalf("glob templates: %v", err)
	}

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		text := strings.ReplaceAll(string(content), "/v1/assets/", "")
		if strings.Contains(text, `"/assets/`) || strings.Contains(text, `'/assets/`) {
			t.Errorf("%s uses reserved institutional site namespace /assets/; use /commercial-assets/", path)
		}
	}
}
