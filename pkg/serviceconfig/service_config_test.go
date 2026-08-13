package serviceconfig

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kageos/kageos/pkg/config"
)

func TestGetGatewayURLAutoResolvesLocalRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("GATEWAY_URL", strings.Replace(server.URL, "127.0.0.1", "localhost", 1))

	got := GetGatewayURL()
	if got == "" {
		t.Fatal("GetGatewayURL returned empty URL")
	}
}

func TestGetGatewayURLKeepsExternalHostWithoutProbe(t *testing.T) {
	t.Setenv("GATEWAY_URL", "https://gateway.example.com")

	if got := GetGatewayURL(); got != "https://gateway.example.com" {
		t.Fatalf("GetGatewayURL = %q, want external URL unchanged", got)
	}
}

func TestGatewayRuntimeURLRequiresExplicitEnvironmentOverride(t *testing.T) {
	t.Setenv("GATEWAY_URL", "")

	if got, ok := gatewayRuntimeURL(); ok || got != "" {
		t.Fatalf("gatewayRuntimeURL() = %q, %v; want empty, false", got, ok)
	}

	t.Setenv("GATEWAY_URL", "  http://localhost:9090  ")
	if got, ok := gatewayRuntimeURL(); !ok || got != "http://localhost:9090" {
		t.Fatalf("gatewayRuntimeURL() = %q, %v; want trimmed URL, true", got, ok)
	}
}

func TestGetGatewayURLUsesAuthoritativeConfigWithoutRuntimeOverride(t *testing.T) {
	t.Setenv("GATEWAY_URL", "")

	want := config.GetGatewayURL()
	if got := GetGatewayURL(); got != want {
		t.Fatalf("GetGatewayURL() = %q, want configured URL %q", got, want)
	}
}

func TestInvalidateGatewayURLForcesReprobe(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("GATEWAY_URL", strings.Replace(server.URL, "127.0.0.1", "localhost", 1))
	_ = GetGatewayURL()
	firstHits := hits.Load()
	_ = GetGatewayURL()
	if hits.Load() != firstHits {
		t.Fatal("successful gateway resolution was not cached")
	}

	InvalidateGatewayURL()
	_ = GetGatewayURL()
	if hits.Load() <= firstHits {
		t.Fatal("gateway invalidation did not force a new probe")
	}
}
