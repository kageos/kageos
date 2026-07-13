package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
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
func (c testMinIOConfig) GetDefaultBucket() string  { return "kageos" }
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

	creds, err := storage.GenerateUploadCredentials(context.Background(), "kageos", "a.txt", "text/plain", time.Hour, UploadSourceBrowser)
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

	creds, err := storage.GenerateUploadCredentials(context.Background(), "kageos", "a.txt", "text/plain", time.Hour, UploadSourceServer)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(creds.ServerUploadURL, "host.containers.internal:9000") {
		t.Fatalf("server upload URL should use server_endpoint, got %s", creds.ServerUploadURL)
	}
}

func TestIsMinIOTimeSkewError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "sdk error code",
			err:  minio.ErrorResponse{Code: "RequestTimeTooSkewed"},
			want: true,
		},
		{
			name: "wrapped sdk error code",
			err:  fmt.Errorf("wrapped: %w", minio.ErrorResponse{Code: "RequestTimeTooSkewed"}),
			want: true,
		},
		{
			name: "minio message",
			err:  errors.New("The difference between the request time and the server's time is too large."),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("access denied"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMinIOTimeSkewError(tt.err); got != tt.want {
				t.Fatalf("isMinIOTimeSkewError() = %v, want %v", got, tt.want)
			}
		})
	}
}
