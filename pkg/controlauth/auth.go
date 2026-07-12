// Package controlauth authenticates short-lived control-plane messages.
//
// It deliberately does not provide identity or authorization by itself. Callers
// choose a narrow scope and sign the complete canonical request fields. The
// verifier enforces freshness, integrity, scope separation, and best-effort
// in-process replay protection.
package controlauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Version             = "v1"
	DefaultMaxAge       = 2 * time.Minute
	DefaultFutureSkew   = 15 * time.Second
	minimumSecretLength = 32
	nonceBytes          = 18
	defaultMaxNonces    = 100_000
	maxCleanupInterval  = 30 * time.Second
	minCleanupInterval  = time.Second
)

var (
	ErrInvalidConfig    = errors.New("controlauth: invalid configuration")
	ErrMissingMetadata  = errors.New("controlauth: missing authentication metadata")
	ErrInvalidTimestamp = errors.New("controlauth: invalid timestamp")
	ErrExpired          = errors.New("controlauth: message expired")
	ErrFutureTimestamp  = errors.New("controlauth: message timestamp is too far in the future")
	ErrInvalidSignature = errors.New("controlauth: invalid signature")
	ErrReplay           = errors.New("controlauth: replay detected")
	ErrReplayCapacity   = errors.New("controlauth: replay cache capacity exceeded")
	ErrAmbiguousHeaders = errors.New("controlauth: ambiguous headers")
)

// Metadata is safe to transport in request headers. Timestamp is Unix time in
// milliseconds. Nonce and Signature use unpadded base64url encoding.
type Metadata struct {
	Version   string
	Timestamp int64
	Nonce     string
	Signature string
}

type Signer struct {
	scope       string
	key         []byte
	now         func() time.Time
	nonceReader io.Reader
}

// NewSigner creates a signer for exactly one control-plane scope. A dedicated
// root secret is domain-separated by scope before it is used as an HMAC key.
func NewSigner(secret, scope string) (*Signer, error) {
	key, normalizedScope, err := deriveKey(secret, scope)
	if err != nil {
		return nil, err
	}
	return &Signer{
		scope:       normalizedScope,
		key:         key,
		now:         time.Now,
		nonceReader: rand.Reader,
	}, nil
}

// Sign signs the ordered canonical fields supplied by the caller.
func (s *Signer) Sign(fields ...[]byte) (Metadata, error) {
	if s == nil || len(s.key) == 0 || strings.TrimSpace(s.scope) == "" {
		return Metadata{}, ErrInvalidConfig
	}
	nonceRaw := make([]byte, nonceBytes)
	if _, err := io.ReadFull(s.nonceReader, nonceRaw); err != nil {
		return Metadata{}, fmt.Errorf("controlauth: generate nonce: %w", err)
	}
	metadata := Metadata{
		Version:   Version,
		Timestamp: s.now().UnixMilli(),
		Nonce:     base64.RawURLEncoding.EncodeToString(nonceRaw),
	}
	metadata.Signature = signCanonical(s.key, s.scope, metadata, fields)
	return metadata, nil
}

type VerifierOptions struct {
	MaxAge        time.Duration
	MaxFutureSkew time.Duration
}

type Verifier struct {
	scope         string
	key           []byte
	maxAge        time.Duration
	maxFutureSkew time.Duration
	now           func() time.Time

	mu         sync.Mutex
	seenNonce  map[string]time.Time
	nextSweep  time.Time
	sweepEvery time.Duration
	maxNonces  int
}

// NewVerifier creates a fail-closed verifier for one scope. Replay state is
// intentionally local to the process; durable business state must remain the
// second replay/idempotency boundary across replicas.
func NewVerifier(secret, scope string, opts VerifierOptions) (*Verifier, error) {
	key, normalizedScope, err := deriveKey(secret, scope)
	if err != nil {
		return nil, err
	}
	maxAge := opts.MaxAge
	if maxAge <= 0 {
		maxAge = DefaultMaxAge
	}
	maxFutureSkew := opts.MaxFutureSkew
	if maxFutureSkew <= 0 {
		maxFutureSkew = DefaultFutureSkew
	}
	sweepEvery := maxAge / 4
	if sweepEvery < minCleanupInterval {
		sweepEvery = minCleanupInterval
	}
	if sweepEvery > maxCleanupInterval {
		sweepEvery = maxCleanupInterval
	}
	return &Verifier{
		scope:         normalizedScope,
		key:           key,
		maxAge:        maxAge,
		maxFutureSkew: maxFutureSkew,
		now:           time.Now,
		seenNonce:     make(map[string]time.Time),
		sweepEvery:    sweepEvery,
		maxNonces:     defaultMaxNonces,
	}, nil
}

// Verify validates freshness and integrity, then atomically records the nonce.
func (v *Verifier) Verify(metadata Metadata, fields ...[]byte) error {
	if v == nil || len(v.key) == 0 || strings.TrimSpace(v.scope) == "" {
		return ErrInvalidConfig
	}
	if metadata.Version == "" || metadata.Timestamp == 0 || metadata.Nonce == "" || metadata.Signature == "" {
		return ErrMissingMetadata
	}
	if metadata.Version != Version {
		return fmt.Errorf("%w: unsupported version %q", ErrInvalidSignature, metadata.Version)
	}
	if _, err := base64.RawURLEncoding.DecodeString(metadata.Nonce); err != nil {
		return fmt.Errorf("%w: malformed nonce", ErrInvalidSignature)
	}
	signature, err := base64.RawURLEncoding.DecodeString(metadata.Signature)
	if err != nil || len(signature) != sha256.Size {
		return fmt.Errorf("%w: malformed signature", ErrInvalidSignature)
	}

	now := v.now()
	issuedAt := time.UnixMilli(metadata.Timestamp)
	if issuedAt.After(now.Add(v.maxFutureSkew)) {
		return ErrFutureTimestamp
	}
	if now.Sub(issuedAt) > v.maxAge {
		return ErrExpired
	}
	expectedText := signCanonical(v.key, v.scope, metadata, fields)
	expected, err := base64.RawURLEncoding.DecodeString(expectedText)
	if err != nil || !hmac.Equal(signature, expected) {
		return ErrInvalidSignature
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	if expiresAt, exists := v.seenNonce[metadata.Nonce]; exists {
		if expiresAt.After(now) {
			// Replayed valid messages are the common hostile path on a shared
			// bus. Reject them in O(1) before considering an O(N) cache sweep.
			return ErrReplay
		}
		delete(v.seenNonce, metadata.Nonce)
	}
	if !v.nextSweep.After(now) || len(v.seenNonce) >= v.maxNonces {
		v.sweepExpiredNonces(now)
		v.nextSweep = now.Add(v.sweepEvery)
	}
	if len(v.seenNonce) >= v.maxNonces {
		return ErrReplayCapacity
	}
	v.seenNonce[metadata.Nonce] = issuedAt.Add(v.maxAge + v.maxFutureSkew)
	return nil
}

func (v *Verifier) sweepExpiredNonces(now time.Time) {
	for nonce, expiresAt := range v.seenNonce {
		if !expiresAt.After(now) {
			delete(v.seenNonce, nonce)
		}
	}
}

func deriveKey(secret, scope string) ([]byte, string, error) {
	secret = strings.TrimSpace(secret)
	scope = strings.TrimSpace(scope)
	if len([]byte(secret)) < minimumSecretLength {
		return nil, "", fmt.Errorf("%w: secret must contain at least %d bytes", ErrInvalidConfig, minimumSecretLength)
	}
	if scope == "" {
		return nil, "", fmt.Errorf("%w: scope is required", ErrInvalidConfig)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	writeFrame(mac, []byte("kageos/controlauth/derived-key/v1"))
	writeFrame(mac, []byte(scope))
	return mac.Sum(nil), scope, nil
}

func signCanonical(key []byte, scope string, metadata Metadata, fields [][]byte) string {
	mac := hmac.New(sha256.New, key)
	writeFrame(mac, []byte("kageos/controlauth/message/v1"))
	writeFrame(mac, []byte(metadata.Version))
	writeFrame(mac, []byte(scope))
	writeFrame(mac, []byte(strconv.FormatInt(metadata.Timestamp, 10)))
	writeFrame(mac, []byte(metadata.Nonce))
	for _, field := range fields {
		writeFrame(mac, field)
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func writeFrame(w io.Writer, value []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = w.Write(length[:])
	_, _ = w.Write(value)
}
