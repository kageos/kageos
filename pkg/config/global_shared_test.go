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

func TestSDKConfigEnvVarsExcludeNATSCredentials(t *testing.T) {
	cfg := &SDKConfig{
		NatsURL:    "nats://platform:secret@nats.internal:4222",
		GatewayURL: "http://gateway.internal:9090",
		EnvVars: map[string]string{
			"NATS_URL":                     "nats://override:secret@nats.internal:4222",
			"KAGEOS_NATS_CREDENTIALS_FILE": "/tmp/override",
			"APP_FEATURE_FLAG":             "enabled",
		},
	}

	envVars := cfg.GetEnvVars()
	if _, ok := envVars["NATS_URL"]; ok {
		t.Fatalf("NATS_URL must not be exposed through SDK env vars: %#v", envVars)
	}
	if _, ok := envVars["KAGEOS_NATS_CREDENTIALS_FILE"]; ok {
		t.Fatalf("NATS credentials path must remain runtime-managed: %#v", envVars)
	}
	if got := envVars["GATEWAY_URL"]; got != "http://gateway.internal:9090" {
		t.Fatalf("GATEWAY_URL = %q", got)
	}
	if got := envVars["APP_FEATURE_FLAG"]; got != "enabled" {
		t.Fatalf("APP_FEATURE_FLAG = %q", got)
	}
}
