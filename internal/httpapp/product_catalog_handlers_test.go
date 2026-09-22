package httpapp

import (
	"net/url"
	"testing"
)

func TestNormalizedCategoryCodesRemovesBlankAndDuplicateValues(t *testing.T) {
	got := normalizedCategoryCodes([]string{"score", " score ", "", "logistics", "score"})
	want := []string{"score", "logistics"}
	if len(got) != len(want) {
		t.Fatalf("got %d category codes, want %d: %#v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("category code %d = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParseCatalogSortOrder(t *testing.T) {
	if got, err := parseCatalogSortOrder(""); err != nil || got != 0 {
		t.Fatalf("empty sort order = %d, %v; want 0, nil", got, err)
	}
	if got, err := parseCatalogSortOrder(" 25 "); err != nil || got != 25 {
		t.Fatalf("sort order = %d, %v; want 25, nil", got, err)
	}
	if _, err := parseCatalogSortOrder("abc"); err == nil {
		t.Fatal("invalid sort order must return an error")
	}
}

func TestParseProductDependencyInputsKeepsGroupsSeparated(t *testing.T) {
	form := url.Values{
		"dependency_mode_2":    {"all"},
		"dependency_product_2": {"product-c", "product-d"},
		"dependency_mode_0":    {"any"},
		"dependency_product_0": {"product-a", "product-b"},
		"dependency_mode_1":    {"any"},
	}

	got := parseProductDependencyInputs(form)
	if len(got) != 2 {
		t.Fatalf("dependency groups = %d, want 2", len(got))
	}
	if got[0].MatchMode != "any" || len(got[0].RequiredProductIDs) != 2 {
		t.Fatalf("unexpected first dependency group: %#v", got[0])
	}
	if got[1].MatchMode != "all" || len(got[1].RequiredProductIDs) != 2 {
		t.Fatalf("unexpected second dependency group: %#v", got[1])
	}
	if got[1].RequiredProductIDs[0] != "product-c" {
		t.Fatalf("unexpected dependency ordering: %#v", got[1].RequiredProductIDs)
	}
}
