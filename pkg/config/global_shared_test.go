package config

import "testing"

func TestGlobalSharedConfigPublicSiteBaseURLUsesSiteConfig(t *testing.T) {
	t.Setenv(EnvCanonicalBaseURL, "https://env.example.com")
	t.Setenv("KAGEOS_BASE_URL", "https://kageos-env.example.com")

	cfg := &GlobalSharedConfig{
		Site:    SiteConfig{BaseURL: " https://app.example.com/ "},
		Gateway: GatewayConfig{BaseURL: "http://127.0.0.1:9090"},
	}

	if got := cfg.GetPublicSiteBaseURL(); got != "https://app.example.com" {
		t.Fatalf("public site base url = %q", got)
	}
}

func TestGlobalSharedConfigPublicSiteBaseURLFallsBackToCanonicalEnv(t *testing.T) {
	t.Setenv(EnvCanonicalBaseURL, "https://canonical.example.com/")
	t.Setenv("KAGEOS_BASE_URL", "")

	cfg := &GlobalSharedConfig{
		Gateway: GatewayConfig{BaseURL: "http://127.0.0.1:9090"},
	}

	if got := cfg.GetPublicSiteBaseURL(); got != "https://canonical.example.com" {
		t.Fatalf("public site base url = %q", got)
	}
}

func TestGlobalSharedConfigPublicSiteBaseURLCanUseExternalGatewayForOldConfig(t *testing.T) {
	t.Setenv(EnvCanonicalBaseURL, "")
	t.Setenv("KAGEOS_BASE_URL", "")

	cfg := &GlobalSharedConfig{
		Gateway: GatewayConfig{Domain: "kageos.example.com"},
	}

	if got := cfg.GetPublicSiteBaseURL(); got != "https://kageos.example.com" {
		t.Fatalf("public site base url = %q", got)
	}
}

func TestGlobalSharedConfigPublicSiteBaseURLAvoidsLocalGateway(t *testing.T) {
	t.Setenv(EnvCanonicalBaseURL, "")
	t.Setenv("KAGEOS_BASE_URL", "")

	cfg := &GlobalSharedConfig{
		Gateway: GatewayConfig{BaseURL: "http://127.0.0.1:9090"},
	}

	if got := cfg.GetPublicSiteBaseURL(); got != "http://localhost:5173" {
		t.Fatalf("public site base url = %q", got)
	}
}
