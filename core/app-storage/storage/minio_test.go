package storage

import (
	"context"
	"strings"
	"testing"
	"time"
)

type testMinIOConfig struct {
	endpoint       string
	serverEndpoint string
	useSSL         bool
	cdnDomain      string
}

func (c testMinIOConfig) GetEndpoint() string       { return c.endpoint }
func (c testMinIOConfig) GetAccessKey() string      { return "minioadmin" }
func (c testMinIOConfig) GetSecretKey() string      { return "minioadmin123" }
func (c testMinIOConfig) GetRegion() string         { return "us-east-1" }
func (c testMinIOConfig) GetUseSSL() bool           { return c.useSSL }
func (c testMinIOConfig) GetDefaultBucket() string  { return "ai-agent-os" }
func (c testMinIOConfig) GetCDNDomain() string      { return c.cdnDomain }
func (c testMinIOConfig) GetServerEndpoint() string { return c.serverEndpoint }

func TestGenerateUploadCredentialsUsesHTTPSFromCDNDomain(t *testing.T) {
	storage, err := NewMinIOStorage(testMinIOConfig{
		endpoint:  "127.0.0.1:9000",
		useSSL:    false,
		cdnDomain: "https://files.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	creds, err := storage.GenerateUploadCredentials(context.Background(), "ai-agent-os", "a.txt", "text/plain", time.Hour, UploadSourceBrowser)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(creds.UploadURL, "https://files.example.com/") {
		t.Fatalf("upload URL should follow cdn_domain scheme, got %s", creds.UploadURL)
	}
}

func TestGenerateUploadCredentialsUsesServerEndpointForServerUpload(t *testing.T) {
	storage, err := NewMinIOStorage(testMinIOConfig{
		endpoint:       "127.0.0.1:9000",
		serverEndpoint: "host.containers.internal:9000",
		useSSL:         false,
	})
	if err != nil {
		t.Fatal(err)
	}

	creds, err := storage.GenerateUploadCredentials(context.Background(), "ai-agent-os", "a.txt", "text/plain", time.Hour, UploadSourceServer)
	if err != nil {
		t.Fatal(err)
	}
	if got := creds.SDKConfig["endpoint"]; got != "host.containers.internal:9000" {
		t.Fatalf("SDK endpoint = %v, want host.containers.internal:9000", got)
	}
	if !strings.Contains(creds.ServerUploadURL, "host.containers.internal:9000") {
		t.Fatalf("server upload URL should use server_endpoint, got %s", creds.ServerUploadURL)
	}
}
