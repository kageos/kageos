package controlauth

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/textproto"
	"sort"
	"strconv"
	"strings"
)

const (
	// HTTPWorkspaceActionScope authenticates the message-server request that
	// starts an agent workspace action through the API gateway.
	HTTPWorkspaceActionScope = "message-server/workspace-action-http/v1"
	// HTTPGatewayAgentBackendScope authenticates the gateway's second hop to
	// agent-server. It is deliberately distinct from the message-server scope,
	// so a first-hop signature can never be replayed directly against Agent.
	HTTPGatewayAgentBackendScope = "api-gateway/agent-backend-http/v1"
	// HTTPAgentDelegatedAPIScope authenticates platform API calls delegated by
	// an already-authenticated Agent request. Gateway policy still constrains
	// the exact methods and paths this scope may reach.
	HTTPAgentDelegatedAPIScope = "agent/delegated-api-http/v1"
	// HTTPGatewayDelegatedBackendScope authenticates the gateway's rewritten
	// second hop for a verified Agent-delegated general API request. It is
	// deliberately separate from the Agent->Gateway scope so an inbound
	// signature cannot be replayed directly against a backend service.
	HTTPGatewayDelegatedBackendScope = "api-gateway/delegated-backend-http/v1"
	// HTTPAgentDelegatedTimerScope is separate because task management has a
	// narrower route policy than Agent's general workspace API calls.
	HTTPAgentDelegatedTimerScope = "agent/delegated-timer-http/v1"
	// HTTPGatewayTimerBackendScope authenticates the gateway's rewritten second
	// hop for an Agent-delegated timer request.
	HTTPGatewayTimerBackendScope = "api-gateway/timer-backend-http/v1"

	HTTPVersionHeader   = "X-Kageos-Control-Version"
	HTTPTimestampHeader = "X-Kageos-Control-Timestamp"
	HTTPNonceHeader     = "X-Kageos-Control-Nonce"
	HTTPSignatureHeader = "X-Kageos-Control-Signature"
)

var httpAuthHeaders = map[string]struct{}{
	strings.ToLower(HTTPVersionHeader):   {},
	strings.ToLower(HTTPTimestampHeader): {},
	strings.ToLower(HTTPNonceHeader):     {},
	strings.ToLower(HTTPSignatureHeader): {},
}

// SignHTTPRequest signs the complete HTTP routing identity, body digest, and
// the selected trusted headers. body must contain the exact bytes that will be
// sent on req.Body. headerNames should include every identity or provenance
// header the receiver may trust after verification.
func SignHTTPRequest(req *http.Request, body []byte, headerNames []string, signer *Signer) error {
	if req == nil {
		return fmt.Errorf("controlauth: http request is nil")
	}
	if signer == nil {
		return ErrInvalidConfig
	}
	ClearHTTPMetadata(req.Header)
	fields, err := canonicalHTTPRequestFields(req, body, headerNames)
	if err != nil {
		return err
	}
	metadata, err := signer.Sign(fields...)
	if err != nil {
		return err
	}
	req.Header.Set(HTTPVersionHeader, metadata.Version)
	req.Header.Set(HTTPTimestampHeader, strconv.FormatInt(metadata.Timestamp, 10))
	req.Header.Set(HTTPNonceHeader, metadata.Nonce)
	req.Header.Set(HTTPSignatureHeader, metadata.Signature)
	return nil
}

// VerifyHTTPRequest validates and consumes the request nonce. Callers must
// read and restore req.Body themselves and pass those exact bytes here.
func VerifyHTTPRequest(req *http.Request, body []byte, headerNames []string, verifier *Verifier) error {
	if req == nil {
		return fmt.Errorf("controlauth: http request is nil")
	}
	if verifier == nil {
		return ErrInvalidConfig
	}
	metadata, err := httpMetadata(req.Header)
	if err != nil {
		return err
	}
	fields, err := canonicalHTTPRequestFields(req, body, headerNames)
	if err != nil {
		return err
	}
	return verifier.Verify(metadata, fields...)
}

// HasHTTPMetadata reports whether any internal authentication metadata is
// present. A partial set is treated as an attempted signed request and should
// fail closed instead of falling back to external authentication.
func HasHTTPMetadata(header http.Header) bool {
	for name, values := range header {
		if _, ok := httpAuthHeaders[strings.ToLower(name)]; ok && len(values) > 0 {
			return true
		}
	}
	return false
}

// ClearHTTPMetadata prevents an authenticated internal request from carrying
// reusable control authentication metadata to downstream services.
func ClearHTTPMetadata(header http.Header) {
	for name := range header {
		if _, ok := httpAuthHeaders[strings.ToLower(name)]; ok {
			delete(header, name)
		}
	}
}

func httpMetadata(header http.Header) (Metadata, error) {
	version, err := singleHTTPHeader(header, HTTPVersionHeader)
	if err != nil {
		return Metadata{}, err
	}
	timestampText, err := singleHTTPHeader(header, HTTPTimestampHeader)
	if err != nil {
		return Metadata{}, err
	}
	nonce, err := singleHTTPHeader(header, HTTPNonceHeader)
	if err != nil {
		return Metadata{}, err
	}
	signature, err := singleHTTPHeader(header, HTTPSignatureHeader)
	if err != nil {
		return Metadata{}, err
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return Metadata{}, ErrInvalidTimestamp
	}
	return Metadata{Version: version, Timestamp: timestamp, Nonce: nonce, Signature: signature}, nil
}

func singleHTTPHeader(header http.Header, name string) (string, error) {
	var values []string
	matchingKeys := 0
	for rawName, rawValues := range header {
		if !strings.EqualFold(rawName, name) {
			continue
		}
		matchingKeys++
		values = append(values, rawValues...)
	}
	if matchingKeys != 1 || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", ErrMissingMetadata
	}
	return values[0], nil
}

func canonicalHTTPRequestFields(req *http.Request, body []byte, headerNames []string) ([][]byte, error) {
	bodyDigest := sha256.Sum256(body)
	protectedHeaders := append([]string(nil), headerNames...)
	protectedHeaders = append(protectedHeaders, "Content-Type")
	selectedHeaders, err := canonicalSelectedHTTPHeaders(req.Header, protectedHeaders)
	if err != nil {
		return nil, err
	}
	path, rawQuery := "", ""
	if req.URL != nil {
		path = req.URL.EscapedPath()
		rawQuery = req.URL.RawQuery
	}
	return [][]byte{
		[]byte(strings.ToUpper(strings.TrimSpace(req.Method))),
		[]byte(canonicalHTTPHost(req)),
		[]byte(path),
		[]byte(rawQuery),
		bodyDigest[:],
		selectedHeaders,
	}, nil
}

func canonicalHTTPHost(req *http.Request) string {
	if host := strings.TrimSpace(req.Host); host != "" {
		return strings.ToLower(host)
	}
	if req.URL != nil {
		return strings.ToLower(strings.TrimSpace(req.URL.Host))
	}
	return ""
}

func canonicalSelectedHTTPHeaders(header http.Header, headerNames []string) ([]byte, error) {
	type entry struct {
		name   string
		values []string
	}

	seen := make(map[string]struct{}, len(headerNames))
	entries := make([]entry, 0, len(headerNames))
	for _, rawName := range headerNames {
		name := strings.ToLower(textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(rawName)))
		if name == "" {
			continue
		}
		if _, authHeader := httpAuthHeaders[name]; authHeader {
			continue
		}
		if _, duplicate := seen[name]; duplicate {
			continue
		}
		seen[name] = struct{}{}
		var values []string
		matchingKeys := 0
		for actualName, actualValues := range header {
			if !strings.EqualFold(actualName, rawName) {
				continue
			}
			matchingKeys++
			values = append(values, actualValues...)
		}
		if matchingKeys > 1 {
			return nil, fmt.Errorf("%w: ambiguous HTTP header %s", ErrInvalidSignature, rawName)
		}
		entries = append(entries, entry{name: name, values: values})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	var out bytes.Buffer
	for _, item := range entries {
		writeHTTPFrame(&out, []byte(item.name))
		var count [8]byte
		binary.BigEndian.PutUint64(count[:], uint64(len(item.values)))
		_, _ = out.Write(count[:])
		for _, value := range item.values {
			writeHTTPFrame(&out, []byte(value))
		}
	}
	return out.Bytes(), nil
}

func writeHTTPFrame(out *bytes.Buffer, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = out.Write(length[:])
	_, _ = out.Write(value)
}
