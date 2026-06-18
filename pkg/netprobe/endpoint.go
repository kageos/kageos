package netprobe

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	tcpEndpointResolvers sync.Map
	httpURLResolvers     sync.Map
)

type cachedResolution struct {
	once  sync.Once
	value string
	err   error
}

// EndpointCandidates returns local runtime alternatives for host:port endpoints.
// The original endpoint is always first. External hosts are returned unchanged.
func EndpointCandidates(endpoint string) []string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil
	}
	host, port, err := splitEndpointHostPort(endpoint)
	if err != nil || !isLocalRuntimeHost(host) {
		return []string{endpoint}
	}
	candidates := make([]string, 0, 4)
	for _, candidateHost := range localRuntimeHosts(host) {
		candidates = appendUnique(candidates, net.JoinHostPort(candidateHost, port))
	}
	return candidates
}

// URLCandidates returns local runtime alternatives for URLs while preserving
// scheme, auth, path, query, and fragment. The original URL is always first.
func URLCandidates(rawURL string) []string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" || !isLocalRuntimeHost(parsed.Hostname()) {
		return []string{rawURL}
	}
	port := parsed.Port()
	if port == "" {
		port = defaultPortForScheme(parsed.Scheme)
	}
	if port == "" {
		return []string{rawURL}
	}
	candidates := make([]string, 0, 4)
	for _, candidateHost := range localRuntimeHosts(parsed.Hostname()) {
		next := *parsed
		next.Host = net.JoinHostPort(candidateHost, port)
		candidates = appendUnique(candidates, next.String())
	}
	return candidates
}

func ResolveTCPEndpointCached(ctx context.Context, label, endpoint string, timeout time.Duration) (string, error) {
	key := label + "\x00" + strings.TrimSpace(endpoint)
	entryValue, _ := tcpEndpointResolvers.LoadOrStore(key, &cachedResolution{})
	entry := entryValue.(*cachedResolution)
	entry.once.Do(func() {
		entry.value, entry.err = ResolveTCPEndpoint(ctx, endpoint, timeout)
		if entry.err != nil {
			entry.value = strings.TrimSpace(endpoint)
		}
	})
	return entry.value, entry.err
}

// ResolveHTTPURLHostCached resolves only the host:port part of a local URL and
// preserves the original scheme, path, query, auth, and fragment. It is useful
// for presigned object URLs where probing the full URL would download content.
func ResolveHTTPURLHostCached(ctx context.Context, label, rawURL string, timeout time.Duration) (string, error) {
	parsed, endpoint, err := localURLTCPEndpoint(rawURL)
	if err != nil {
		return strings.TrimSpace(rawURL), err
	}
	if endpoint == "" {
		return strings.TrimSpace(rawURL), nil
	}
	resolvedEndpoint, err := ResolveTCPEndpointCached(ctx, label, endpoint, timeout)
	if err != nil {
		return strings.TrimSpace(rawURL), err
	}
	next := *parsed
	next.Host = resolvedEndpoint
	return next.String(), nil
}

func ResolveTCPEndpoint(ctx context.Context, endpoint string, timeout time.Duration) (string, error) {
	candidates := EndpointCandidates(endpoint)
	if len(candidates) == 0 {
		return "", fmt.Errorf("empty endpoint")
	}
	var failures []string
	for _, candidate := range candidates {
		if err := probeTCP(ctx, candidate, timeout); err == nil {
			return candidate, nil
		} else {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
		}
	}
	return candidates[0], fmt.Errorf("all endpoint candidates failed: %s", strings.Join(failures, "; "))
}

func localURLTCPEndpoint(rawURL string) (*url.URL, string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, "", fmt.Errorf("empty URL")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	if parsed.Host == "" || !isLocalRuntimeHost(parsed.Hostname()) {
		return parsed, "", nil
	}
	port := parsed.Port()
	if port == "" {
		port = defaultPortForScheme(parsed.Scheme)
	}
	if port == "" {
		return parsed, "", nil
	}
	return parsed, net.JoinHostPort(parsed.Hostname(), port), nil
}

func ResolveHTTPBaseURLCached(ctx context.Context, label, rawURL, healthPath string, timeout time.Duration) (string, error) {
	key := label + "\x00" + strings.TrimSpace(rawURL) + "\x00" + healthPath
	entryValue, _ := httpURLResolvers.LoadOrStore(key, &cachedResolution{})
	entry := entryValue.(*cachedResolution)
	entry.once.Do(func() {
		entry.value, entry.err = ResolveHTTPBaseURL(ctx, rawURL, healthPath, timeout)
		if entry.err != nil {
			entry.value = strings.TrimSpace(rawURL)
		}
	})
	return entry.value, entry.err
}

func ResolveHTTPBaseURL(ctx context.Context, rawURL, healthPath string, timeout time.Duration) (string, error) {
	candidates := URLCandidates(rawURL)
	if len(candidates) == 0 {
		return "", fmt.Errorf("empty URL")
	}
	client := &http.Client{Timeout: timeout}
	var failures []string
	for _, candidate := range candidates {
		target, err := joinURL(candidate, healthPath)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return candidate, nil
		}
		failures = append(failures, fmt.Sprintf("%s: HTTP %d", candidate, resp.StatusCode))
	}
	return candidates[0], fmt.Errorf("all URL candidates failed: %s", strings.Join(failures, "; "))
}

func probeTCP(ctx context.Context, endpoint string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var dialer net.Dialer
	conn, err := dialer.DialContext(probeCtx, "tcp", endpoint)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func splitEndpointHostPort(endpoint string) (string, string, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err == nil {
		return strings.Trim(host, "[]"), port, nil
	}
	if strings.Count(endpoint, ":") == 1 {
		parts := strings.SplitN(endpoint, ":", 2)
		return parts[0], parts[1], nil
	}
	return "", "", err
}

func localRuntimeHosts(original string) []string {
	hosts := []string{strings.Trim(original, "[]"), "127.0.0.1", "host.containers.internal", "host.docker.internal", "localhost"}
	return uniqueStrings(hosts)
}

func isLocalRuntimeHost(host string) bool {
	host = strings.ToLower(strings.Trim(strings.TrimSpace(host), "[]"))
	if host == "localhost" || host == "host.containers.internal" || host == "host.docker.internal" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	case "nats":
		return "4222"
	default:
		return ""
	}
}

func joinURL(baseURL, path string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	return parsed.String(), nil
}

func appendUnique(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = appendUnique(out, value)
	}
	return out
}
