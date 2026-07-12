package config

import (
	"strings"
	"testing"
)

func TestControlPlaneConfigRequiresDedicatedStrongSecret(t *testing.T) {
	if _, err := (ControlPlaneConfig{}).GetSecret(); err == nil {
		t.Fatal("GetSecret() with empty secret should fail closed")
	}
	if _, err := (ControlPlaneConfig{Secret: "short"}).GetSecret(); err == nil {
		t.Fatal("GetSecret() with short secret should fail closed")
	}
	want := strings.Repeat("c", 32)
	got, err := (ControlPlaneConfig{Secret: " " + want + " "}).GetSecret()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("GetSecret() = %q, want %q", got, want)
	}
}

func TestGlobalSharedConfigLegacyControlPlaneSecretFallback(t *testing.T) {
	legacyJWTSecret := strings.Repeat("j", 32)
	cfg := &GlobalSharedConfig{JWT: JWTConfig{Secret: legacyJWTSecret}}
	got, err := cfg.ResolveControlPlaneSecret()
	if err != nil {
		t.Fatal(err)
	}
	if got != legacyJWTSecret {
		t.Fatalf("ResolveControlPlaneSecret() = %q, want legacy JWT secret", got)
	}

	cfg.ControlPlane.Secret = "explicit-but-short"
	if _, err := cfg.ResolveControlPlaneSecret(); err == nil {
		t.Fatal("explicit weak control_plane.secret must fail instead of falling back")
	}
}
