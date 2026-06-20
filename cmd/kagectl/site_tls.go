package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"strings"
	"time"
)

const tlsModeAuto = "auto"

func applyInitialSitePolicy(site *SiteConfig, tlsMode string) error {
	if site == nil {
		return nil
	}
	tlsMode = strings.ToLower(strings.TrimSpace(tlsMode))
	if tlsMode == "" {
		tlsMode = tlsModeAuto
	}
	if err := validateInitTLSMode(tlsMode); err != nil {
		return err
	}
	if strings.TrimSpace(site.BaseURL) == "" {
		if tlsMode != tlsModeAuto {
			site.TLSMode = tlsMode
		}
		return nil
	}

	base, host, err := normalizeSiteBaseURL(site.BaseURL, tlsMode)
	if err != nil {
		return err
	}
	site.BaseURL = base

	if tlsMode != tlsModeAuto {
		site.TLSMode = tlsMode
		switch tlsMode {
		case "http":
			site.BaseURL = setBaseURLScheme(site.BaseURL, "http")
		case "https", "redirect", "external":
			site.BaseURL = setBaseURLScheme(site.BaseURL, "https")
		}
		if tlsMode != "https" && tlsMode != "redirect" {
			site.AllowSelfSignedBootstrap = false
			return nil
		}
		if siteTLSCertsEmpty(*site) && shouldSelfSignForHost(host) {
			site.AllowSelfSignedBootstrap = true
		}
		return nil
	}

	if shouldSelfSignForHost(host) {
		site.BaseURL = setBaseURLScheme(site.BaseURL, "https")
		site.TLSMode = "redirect"
		if siteTLSCertsEmpty(*site) {
			site.AllowSelfSignedBootstrap = true
		}
		return nil
	}

	site.BaseURL = setBaseURLScheme(site.BaseURL, "http")
	site.TLSMode = "http"
	site.AllowSelfSignedBootstrap = false
	return nil
}

func validateInitTLSMode(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", tlsModeAuto, "http", "https", "redirect", "external":
		return nil
	default:
		return fmt.Errorf("--tls-mode requires auto, http, https, redirect, or external")
	}
}

func normalizeSiteBaseURL(raw string, tlsMode string) (baseURL string, host string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", nil
	}
	parsed, hadScheme, err := parseSiteBaseURL(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse site.base_url: %w", err)
	}
	host = strings.Trim(strings.ToLower(parsed.Hostname()), "[]")
	if host == "" {
		return "", "", fmt.Errorf("site.base_url host is required")
	}
	if !hadScheme {
		scheme := "http"
		if tlsMode == "https" || tlsMode == "redirect" || tlsMode == "external" || (tlsMode == tlsModeAuto && shouldSelfSignForHost(host)) {
			scheme = "https"
		}
		parsed.Scheme = scheme
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), host, nil
}

func parseSiteBaseURL(raw string) (*url.URL, bool, error) {
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return nil, false, err
		}
		return parsed, true, nil
	}
	parsed, err := url.Parse("//" + raw)
	if err != nil {
		return nil, false, err
	}
	return parsed, false, nil
}

func setBaseURLScheme(raw string, scheme string) string {
	parsed, _, err := parseSiteBaseURL(raw)
	if err != nil {
		return raw
	}
	parsed.Scheme = scheme
	return parsed.String()
}

func siteTLSCertsEmpty(site SiteConfig) bool {
	return strings.TrimSpace(site.TLSCertPEMB64) == "" && strings.TrimSpace(site.TLSKeyPEMB64) == ""
}

func siteTLSCertsIncomplete(site SiteConfig) bool {
	certEmpty := strings.TrimSpace(site.TLSCertPEMB64) == ""
	keyEmpty := strings.TrimSpace(site.TLSKeyPEMB64) == ""
	return certEmpty != keyEmpty
}

func shouldSelfSignForHost(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return false
	}
	if net.ParseIP(host) != nil {
		return false
	}
	return strings.Contains(host, ".")
}

func generateSelfSignedTLSPEM(baseURL string) ([]byte, []byte, error) {
	host, err := hostFromBaseURL(baseURL)
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate self-signed TLS key: %w", err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("generate self-signed TLS serial: %w", err)
	}
	now := time.Now()
	commonName := host
	if len(commonName) > 64 {
		commonName = "Kageos Bootstrap TLS"
	}
	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Kageos Bootstrap"},
		},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create self-signed TLS certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal self-signed TLS key: %w", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}

func hostFromBaseURL(raw string) (string, error) {
	parsed, _, err := parseSiteBaseURL(raw)
	if err != nil {
		return "", err
	}
	host := strings.Trim(parsed.Hostname(), "[]")
	if host == "" {
		return "", fmt.Errorf("site.base_url host is required")
	}
	return host, nil
}
