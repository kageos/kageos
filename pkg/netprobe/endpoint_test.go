package netprobe

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEndpointCandidatesForLocalRuntimeHosts(t *testing.T) {
	got := EndpointCandidates("host.containers.internal:4222")
	for _, want := range []string{
		"host.containers.internal:4222",
		"127.0.0.1:4222",
		"host.docker.internal:4222",
		"localhost:4222",
	} {
		if !containsString(got, want) {
			t.Fatalf("EndpointCandidates missing %q: %#v", want, got)
		}
	}
}

func TestEndpointCandidatesKeepExternalHost(t *testing.T) {
	got := EndpointCandidates("nats.example.com:4222")
	if len(got) != 1 || got[0] != "nats.example.com:4222" {
		t.Fatalf("EndpointCandidates external = %#v", got)
	}
}

func TestURLCandidatesPreserveAuthAndPath(t *testing.T) {
	got := URLCandidates("nats://aos:secret@host.containers.internal:4222/path?q=1")
	want := "nats://aos:secret@127.0.0.1:4222/path?q=1"
	if !containsString(got, want) {
		t.Fatalf("URLCandidates missing %q: %#v", want, got)
	}
}

func TestResolveHTTPBaseURLReturnsReachableCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	rawURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	got, err := ResolveHTTPBaseURL(t.Context(), rawURL, "/health", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected resolved URL")
	}
}

func TestResolveHTTPURLHostCachedPreservesURLParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rawURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/kageos/a.png?X-Amz-Signature=secret#frag"
	got, err := ResolveHTTPURLHostCached(t.Context(), t.Name(), rawURL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/kageos/a.png" || parsed.RawQuery != "X-Amz-Signature=secret" || parsed.Fragment != "frag" {
		t.Fatalf("resolved URL did not preserve path/query/fragment: %s", got)
	}
}

func TestResolveHTTPURLHostCachedKeepsExternalURL(t *testing.T) {
	rawURL := "https://files.example.com/kageos/a.png?x=1"
	got, err := ResolveHTTPURLHostCached(t.Context(), t.Name(), rawURL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got != rawURL {
		t.Fatalf("ResolveHTTPURLHostCached external = %q, want %q", got, rawURL)
	}
}

func TestResolveHTTPBaseURLCachedProbesOnce(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	got, err := ResolveHTTPBaseURLCached(t.Context(), t.Name(), server.URL, "/health", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected resolved URL")
	}
	firstHits := atomic.LoadInt32(&hits)
	if firstHits == 0 {
		t.Fatal("expected first call to probe")
	}

	got, err = ResolveHTTPBaseURLCached(t.Context(), t.Name(), server.URL, "/health", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected cached URL")
	}
	if secondHits := atomic.LoadInt32(&hits); secondHits != firstHits {
		t.Fatalf("expected cached call to skip probing, hits before=%d after=%d", firstHits, secondHits)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
