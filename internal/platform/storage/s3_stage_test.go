package storage

import "testing"

func TestStageKeyPrefixesLogicalKey(t *testing.T) {
	client := &S3{stage: "prod"}
	got, err := client.StageKey("commercial-assets/user/logo.png")
	if err != nil {
		t.Fatalf("StageKey returned error: %v", err)
	}
	want := "prod/commercial-assets/user/logo.png"
	if got != want {
		t.Fatalf("StageKey = %q, want %q", got, want)
	}
}

func TestStageKeyKeepsMatchingPrefix(t *testing.T) {
	client := &S3{stage: "dev"}
	got, err := client.StageKey("dev/contracts/contract.pdf")
	if err != nil {
		t.Fatalf("StageKey returned error: %v", err)
	}
	if got != "dev/contracts/contract.pdf" {
		t.Fatalf("StageKey = %q", got)
	}
}

func TestStageKeyRejectsCrossEnvironmentKey(t *testing.T) {
	client := &S3{stage: "prod"}
	if _, err := client.StageKey("dev/contracts/contract.pdf"); err == nil {
		t.Fatal("expected cross-environment key to be rejected")
	}
}

func TestStageKeyRejectsTraversal(t *testing.T) {
	client := &S3{stage: "prod"}
	invalid := []string{"../secret", "contracts/../secret", "contracts//secret", `contracts\secret`}
	for _, key := range invalid {
		if _, err := client.StageKey(key); err == nil {
			t.Fatalf("expected key %q to be rejected", key)
		}
	}
}
