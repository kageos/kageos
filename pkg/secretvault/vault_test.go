package secretvault

import (
	"strings"
	"testing"
)

func TestVaultSealsAndOpensWithPrefix(t *testing.T) {
	vault, err := New("secret", "test-purpose", WithPrefix("test:v1:"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sealed, err := vault.Seal("plain-secret")
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if sealed == "plain-secret" || !strings.HasPrefix(sealed, "test:v1:") {
		t.Fatalf("sealed value = %q, want prefixed ciphertext", sealed)
	}
	if !vault.IsSealed(sealed) {
		t.Fatalf("IsSealed(%q) = false, want true", sealed)
	}

	opened, err := vault.Open(sealed)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if opened != "plain-secret" {
		t.Fatalf("Open() = %q, want plain-secret", opened)
	}

	sealedAgain, err := vault.Seal(sealed)
	if err != nil {
		t.Fatalf("Seal(already sealed) error = %v", err)
	}
	if sealedAgain != sealed {
		t.Fatalf("Seal(already sealed) = %q, want unchanged %q", sealedAgain, sealed)
	}
}

func TestVaultOpensUnprefixedLegacyPlaintext(t *testing.T) {
	vault, err := New("secret", "test-purpose", WithPrefix("test:v1:"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	opened, err := vault.Open("legacy-secret")
	if err != nil {
		t.Fatalf("Open(legacy) error = %v", err)
	}
	if opened != "legacy-secret" {
		t.Fatalf("Open(legacy) = %q, want unchanged plaintext", opened)
	}
	if vault.IsSealed("legacy-secret") {
		t.Fatalf("IsSealed(legacy) = true, want false")
	}
}

func TestVaultRejectsWrongPurpose(t *testing.T) {
	vault, err := New("secret", "test-purpose", WithPrefix("test:v1:"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sealed, err := vault.Seal("plain-secret")
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	other, err := New("secret", "other-purpose", WithPrefix("test:v1:"))
	if err != nil {
		t.Fatalf("New(other) error = %v", err)
	}
	if _, err := other.Open(sealed); err == nil {
		t.Fatal("Open() with wrong purpose succeeded, want error")
	}
}
