package main

import (
	"strings"
	"testing"
)

func TestApplyDefaultsKeepsLegacyConfigUpgradableWithControlPlaneSecret(t *testing.T) {
	legacyJWTSecret := strings.Repeat("j", 32)
	cfg := Config{Secrets: SecretsConfig{JWTSecret: legacyJWTSecret}}
	applyDefaults(&cfg)
	if cfg.Secrets.ControlPlaneSecret != legacyJWTSecret {
		t.Fatalf("legacy control-plane secret = %q, want JWT fallback", cfg.Secrets.ControlPlaneSecret)
	}
}
