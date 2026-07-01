package service

import (
	"strings"
	"testing"
)

func TestOAuthTokenEndpointErrorHidesBody(t *testing.T) {
	err := oauthTokenEndpointError(400, []byte(`{"error":"invalid_grant","access_token":"secret-token","client_secret":"secret-client"}`))
	if err == nil {
		t.Fatal("expected error")
	}

	got := err.Error()
	for _, leaked := range []string{"secret-token", "secret-client", "access_token", "client_secret", "invalid_grant"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("oauthTokenEndpointError() leaked %q in %q", leaked, got)
		}
	}
	if !strings.Contains(got, "400") || !strings.Contains(got, "hidden") {
		t.Fatalf("oauthTokenEndpointError() = %q, want status and hidden marker", got)
	}
}
