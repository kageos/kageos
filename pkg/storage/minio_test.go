package storage

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestResolveUploadURLPrefersServerURLAndPreservesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	rawURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/kageos/a.txt?X-Amz-Signature=secret"
	got := resolveUploadURL(t.Context(), &dto.GetUploadTokenResp{
		UploadURL:       "https://browser.example/kageos/a.txt",
		ServerUploadURL: rawURL,
	})
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/kageos/a.txt" || parsed.RawQuery != "X-Amz-Signature=secret" {
		t.Fatalf("resolved upload URL should preserve path/query, got %s", got)
	}
}

func TestResolveUploadURLKeepsExternalBrowserURL(t *testing.T) {
	rawURL := "https://browser.example/kageos/a.txt?x=1"
	got := resolveUploadURL(t.Context(), &dto.GetUploadTokenResp{UploadURL: rawURL})
	if got != rawURL {
		t.Fatalf("resolveUploadURL() = %q, want %q", got, rawURL)
	}
}

func TestSlowUploadThreshold(t *testing.T) {
	t.Setenv("KAGEOS_SDK_SLOW_UPLOAD_MS", "")
	if got := slowUploadThreshold(); got != 5*time.Second {
		t.Fatalf("default slowUploadThreshold() = %v, want 5s", got)
	}

	t.Setenv("KAGEOS_SDK_SLOW_UPLOAD_MS", "250")
	if got := slowUploadThreshold(); got != 250*time.Millisecond {
		t.Fatalf("slowUploadThreshold() = %v, want 250ms", got)
	}

	t.Setenv("KAGEOS_SDK_SLOW_UPLOAD_MS", "0")
	if got := slowUploadThreshold(); got != 0 {
		t.Fatalf("disabled slowUploadThreshold() = %v, want 0", got)
	}
}
