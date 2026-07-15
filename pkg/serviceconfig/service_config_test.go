package serviceconfig

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
