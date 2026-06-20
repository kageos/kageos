package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestUpdateTLSCertificateWritesMatchingPEM(t *testing.T) {
	certPEM, keyPEM := mustSelfSignedTLSPEM(t, "app.example.com")
	dir := t.TempDir()
	certFile := filepath.Join(dir, "fullchain.pem")
	keyFile := filepath.Join(dir, "privkey.pem")
	t.Setenv("TLS_MODE", "http")
	t.Setenv("TLS_CERT_FILE", certFile)
	t.Setenv("TLS_KEY_FILE", keyFile)

	settings, err := (&SystemSettingsService{}).UpdateTLSCertificate(dto.UpdateTLSCertificateReq{
		CertificatePEM: certPEM,
		PrivateKeyPEM:  keyPEM,
		Reload:         false,
	}, "system")
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Ready || !settings.CertExists || !settings.KeyExists {
		t.Fatalf("TLS settings should be ready after upload: %#v", settings)
	}
	if got := mustReadTLSFile(t, certFile); !strings.Contains(got, "BEGIN CERTIFICATE") {
		t.Fatalf("cert file was not written as PEM: %q", got)
	}
	if got := mustReadTLSFile(t, keyFile); !strings.Contains(got, "BEGIN PRIVATE KEY") {
		t.Fatalf("key file was not written as PEM: %q", got)
	}
	if settings.Certificate == nil || !settings.Certificate.IsSelfSigned {
		t.Fatalf("expected parsed self-signed certificate info, got %#v", settings.Certificate)
	}
}

func TestUpdateTLSCertificateRejectsMismatchedKey(t *testing.T) {
	certPEM, _ := mustSelfSignedTLSPEM(t, "app.example.com")
	_, otherKeyPEM := mustSelfSignedTLSPEM(t, "other.example.com")
	dir := t.TempDir()
	t.Setenv("TLS_CERT_FILE", filepath.Join(dir, "fullchain.pem"))
	t.Setenv("TLS_KEY_FILE", filepath.Join(dir, "privkey.pem"))

	_, err := (&SystemSettingsService{}).UpdateTLSCertificate(dto.UpdateTLSCertificateReq{
		CertificatePEM: certPEM,
		PrivateKeyPEM:  otherKeyPEM,
	}, "system")
	if err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
}

func mustSelfSignedTLSPEM(t *testing.T, host string) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: host,
		},
		DNSNames:              []string{host},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM)
}

func mustReadTLSFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
