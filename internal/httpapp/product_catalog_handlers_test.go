package httpapp

import "testing"

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
