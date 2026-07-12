package controlauth

import (
	"bytes"
	"errors"
	"net/http"
	"testing"
	"time"
)

const httpTestSecret = "0123456789abcdef0123456789abcdef"

var httpTestProtectedHeaders = []string{
	"X-Request-User",
	"X-Client-Source",
	"X-Workspace-Role",
}

func TestHTTPRequestSignAndVerify(t *testing.T) {
	body := []byte(`{"message":"hello"}`)
	req, err := http.NewRequest(http.MethodPost, "http://gateway.internal/agent/chat?mode=run", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-User", "alice")
	req.Header.Set("X-Client-Source", "message_action")

	signer := mustHTTPTestSigner(t)
	verifier := mustHTTPTestVerifier(t)
	if err := SignHTTPRequest(req, body, httpTestProtectedHeaders, signer); err != nil {
		t.Fatal(err)
	}
	if !HasHTTPMetadata(req.Header) {
		t.Fatal("signed request does not report authentication metadata")
	}
	if err := VerifyHTTPRequest(req, body, httpTestProtectedHeaders, verifier); err != nil {
		t.Fatalf("verify signed request: %v", err)
	}

	ClearHTTPMetadata(req.Header)
	if HasHTTPMetadata(req.Header) {
		t.Fatal("authentication metadata survived clear")
	}
}

func TestHTTPRequestVerificationRejectsTampering(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*http.Request, []byte) []byte
	}{
		{name: "method", mutate: func(req *http.Request, body []byte) []byte { req.Method = http.MethodPut; return body }},
		{name: "host", mutate: func(req *http.Request, body []byte) []byte { req.Host = "evil.internal"; return body }},
		{name: "path", mutate: func(req *http.Request, body []byte) []byte { req.URL.Path = "/agent/admin"; return body }},
		{name: "query", mutate: func(req *http.Request, body []byte) []byte { req.URL.RawQuery = "mode=admin"; return body }},
		{name: "content-type", mutate: func(req *http.Request, body []byte) []byte { req.Header.Set("Content-Type", "text/plain"); return body }},
		{name: "body", mutate: func(req *http.Request, body []byte) []byte { return []byte(`{"message":"tampered"}`) }},
		{name: "identity", mutate: func(req *http.Request, body []byte) []byte { req.Header.Set("X-Request-User", "mallory"); return body }},
		{name: "extra-identity-value", mutate: func(req *http.Request, body []byte) []byte { req.Header.Add("X-Request-User", "mallory"); return body }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"message":"hello"}`)
			req, err := http.NewRequest(http.MethodPost, "http://gateway.internal/agent/chat?mode=run", bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-User", "alice")
			if err := SignHTTPRequest(req, body, httpTestProtectedHeaders, mustHTTPTestSigner(t)); err != nil {
				t.Fatal(err)
			}
			body = tt.mutate(req, body)
			err = VerifyHTTPRequest(req, body, httpTestProtectedHeaders, mustHTTPTestVerifier(t))
			if !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("verify tampered request error = %v, want ErrInvalidSignature", err)
			}
		})
	}
}

func TestHTTPRequestVerificationRejectsReplayAndPartialMetadata(t *testing.T) {
	body := []byte("payload")
	req, err := http.NewRequest(http.MethodPost, "http://gateway.internal/action", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Request-User", "alice")
	if err := SignHTTPRequest(req, body, httpTestProtectedHeaders, mustHTTPTestSigner(t)); err != nil {
		t.Fatal(err)
	}
	verifier := mustHTTPTestVerifier(t)
	if err := VerifyHTTPRequest(req, body, httpTestProtectedHeaders, verifier); err != nil {
		t.Fatal(err)
	}
	if err := VerifyHTTPRequest(req, body, httpTestProtectedHeaders, verifier); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay error = %v, want ErrReplay", err)
	}

	partial, err := http.NewRequest(http.MethodGet, "http://gateway.internal/action", nil)
	if err != nil {
		t.Fatal(err)
	}
	partial.Header.Set(HTTPNonceHeader, "nonce-only")
	if !HasHTTPMetadata(partial.Header) {
		t.Fatal("partial metadata was not detected")
	}
	if err := VerifyHTTPRequest(partial, nil, httpTestProtectedHeaders, mustHTTPTestVerifier(t)); !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("partial metadata error = %v, want ErrMissingMetadata", err)
	}
}

func TestHTTPRequestVerificationRejectsReorderedOrAmbiguousHeaderValues(t *testing.T) {
	body := []byte("payload")
	req, err := http.NewRequest(http.MethodPost, "http://gateway.internal/action", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header["X-Request-User"] = []string{"alice", "delegated-alice"}
	if err := SignHTTPRequest(req, body, httpTestProtectedHeaders, mustHTTPTestSigner(t)); err != nil {
		t.Fatal(err)
	}
	req.Header["X-Request-User"][0], req.Header["X-Request-User"][1] =
		req.Header["X-Request-User"][1], req.Header["X-Request-User"][0]
	if err := VerifyHTTPRequest(req, body, httpTestProtectedHeaders, mustHTTPTestVerifier(t)); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("reordered values error = %v, want ErrInvalidSignature", err)
	}

	ambiguous, err := http.NewRequest(http.MethodPost, "http://gateway.internal/action", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	ambiguous.Header.Set("X-Request-User", "alice")
	if err := SignHTTPRequest(ambiguous, body, httpTestProtectedHeaders, mustHTTPTestSigner(t)); err != nil {
		t.Fatal(err)
	}
	ambiguous.Header["x-request-user"] = []string{"mallory"}
	if err := VerifyHTTPRequest(ambiguous, body, httpTestProtectedHeaders, mustHTTPTestVerifier(t)); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("ambiguous casing error = %v, want ErrInvalidSignature", err)
	}
}

func mustHTTPTestSigner(t *testing.T) *Signer {
	t.Helper()
	signer, err := NewSigner(httpTestSecret, "http-test")
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func mustHTTPTestVerifier(t *testing.T) *Verifier {
	t.Helper()
	verifier, err := NewVerifier(httpTestSecret, "http-test", VerifierOptions{
		MaxAge:        time.Minute,
		MaxFutureSkew: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}
