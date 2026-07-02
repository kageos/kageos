package serverx

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestStartHTTPServerListensBeforeReturning(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	srv, err := StartHTTPServer(context.Background(), "127.0.0.1:0", mux)
	if err != nil {
		t.Fatalf("StartHTTPServer() error = %v", err)
	}
	defer func() {
		if err := srv.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	resp, err := http.Get("http://" + srv.Addr() + "/health")
	if err != nil {
		t.Fatalf("GET health after StartHTTPServer returned: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("health response = status %d body %q, want 200 ok", resp.StatusCode, string(body))
	}
}

func TestStartHTTPServerFailsWhenPortIsBusy(t *testing.T) {
	srv, err := StartHTTPServer(context.Background(), "127.0.0.1:0", http.NewServeMux())
	if err != nil {
		t.Fatalf("StartHTTPServer() error = %v", err)
	}
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()

	_, err = StartHTTPServer(context.Background(), srv.Addr(), http.NewServeMux())
	if err == nil {
		t.Fatal("StartHTTPServer() on busy port succeeded, want error")
	}
	if !strings.Contains(err.Error(), "address already in use") && !strings.Contains(err.Error(), "bind") {
		t.Fatalf("StartHTTPServer() busy port error = %v, want bind error", err)
	}
}

func TestHTTPServerShutdownAcceptsCancelledContext(t *testing.T) {
	srv, err := StartHTTPServer(context.Background(), "127.0.0.1:0", http.NewServeMux())
	if err != nil {
		t.Fatalf("StartHTTPServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown(cancelled context) error = %v", err)
	}

	select {
	case err := <-srv.Err():
		if err != nil {
			t.Fatalf("serve error after shutdown = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not stop after Shutdown")
	}
}
