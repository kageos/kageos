package controlauth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const testControlSecret = "0123456789abcdef0123456789abcdef"

func TestSignAndVerify(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	signer := mustSigner(t, "timer.scheduler.v1")
	signer.now = func() time.Time { return now }
	verifier := mustVerifier(t, "timer.scheduler.v1", VerifierOptions{})
	verifier.now = func() time.Time { return now }

	metadata, err := signer.Sign([]byte("subject"), []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify(metadata, []byte("subject"), []byte("payload")); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := verifier.Verify(metadata, []byte("subject"), []byte("payload")); !errors.Is(err, ErrReplay) {
		t.Fatalf("second Verify() error = %v, want ErrReplay", err)
	}
}

func TestVerifyRejectsTamperingAndWrongScope(t *testing.T) {
	signer := mustSigner(t, "timer.scheduler.v1")
	metadata, err := signer.Sign([]byte("timer.v1.cmd.execution.requested.agent.session"), []byte(`{"id":1}`))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		scope    string
		fields   [][]byte
		metadata Metadata
	}{
		{name: "subject", scope: "timer.scheduler.v1", fields: [][]byte{[]byte("timer.v1.cmd.execution.finished"), []byte(`{"id":1}`)}, metadata: metadata},
		{name: "payload", scope: "timer.scheduler.v1", fields: [][]byte{[]byte("timer.v1.cmd.execution.requested.agent.session"), []byte(`{"id":2}`)}, metadata: metadata},
		{name: "scope", scope: "timer.worker.v1", fields: [][]byte{[]byte("timer.v1.cmd.execution.requested.agent.session"), []byte(`{"id":1}`)}, metadata: metadata},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := mustVerifier(t, tt.scope, VerifierOptions{})
			if err := verifier.Verify(tt.metadata, tt.fields...); !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("Verify() error = %v, want ErrInvalidSignature", err)
			}
		})
	}
}

func TestVerifyRejectsExpiredAndFutureMessages(t *testing.T) {
	issuedAt := time.Unix(1_750_000_000, 0)
	signer := mustSigner(t, "timer.scheduler.v1")
	signer.now = func() time.Time { return issuedAt }
	metadata, err := signer.Sign([]byte("subject"))
	if err != nil {
		t.Fatal(err)
	}

	expired := mustVerifier(t, "timer.scheduler.v1", VerifierOptions{MaxAge: time.Minute})
	expired.now = func() time.Time { return issuedAt.Add(time.Minute + time.Millisecond) }
	if err := expired.Verify(metadata, []byte("subject")); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired Verify() error = %v, want ErrExpired", err)
	}

	future := mustVerifier(t, "timer.scheduler.v1", VerifierOptions{MaxFutureSkew: 5 * time.Second})
	future.now = func() time.Time { return issuedAt.Add(-5*time.Second - time.Millisecond) }
	if err := future.Verify(metadata, []byte("subject")); !errors.Is(err, ErrFutureTimestamp) {
		t.Fatalf("future Verify() error = %v, want ErrFutureTimestamp", err)
	}
}

func TestReplayFastPathDoesNotSweepEntireNonceCache(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	signer := mustSigner(t, "timer.scheduler.v1")
	signer.now = func() time.Time { return now }
	metadata, err := signer.Sign([]byte("subject"))
	if err != nil {
		t.Fatal(err)
	}
	verifier := mustVerifier(t, "timer.scheduler.v1", VerifierOptions{})
	verifier.now = func() time.Time { return now }
	verifier.seenNonce[metadata.Nonce] = now.Add(time.Minute)
	verifier.seenNonce["expired-unrelated-entry"] = now.Add(-time.Second)

	if err := verifier.Verify(metadata, []byte("subject")); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay Verify() error = %v, want ErrReplay", err)
	}
	if _, exists := verifier.seenNonce["expired-unrelated-entry"]; !exists {
		t.Fatal("replay fast path unexpectedly swept unrelated nonce entries")
	}

	newMetadata, err := signer.Sign([]byte("subject"))
	if err != nil {
		t.Fatal(err)
	}
	if err := verifier.Verify(newMetadata, []byte("subject")); err != nil {
		t.Fatalf("new Verify() error = %v", err)
	}
	if _, exists := verifier.seenNonce["expired-unrelated-entry"]; exists {
		t.Fatal("new nonce path did not sweep expired entries")
	}
}

func TestVerifierFailsClosedWhenReplayCacheIsFull(t *testing.T) {
	now := time.Unix(1_750_000_000, 0)
	signer := mustSigner(t, "timer.scheduler.v1")
	signer.now = func() time.Time { return now }
	metadata, err := signer.Sign([]byte("subject"))
	if err != nil {
		t.Fatal(err)
	}
	verifier := mustVerifier(t, "timer.scheduler.v1", VerifierOptions{})
	verifier.now = func() time.Time { return now }
	verifier.maxNonces = 1
	verifier.nextSweep = now.Add(time.Minute)
	verifier.seenNonce["still-live"] = now.Add(time.Minute)

	if err := verifier.Verify(metadata, []byte("subject")); !errors.Is(err, ErrReplayCapacity) {
		t.Fatalf("full cache Verify() error = %v, want ErrReplayCapacity", err)
	}
}

func TestNewSignerRequiresDedicatedStrongSecretAndScope(t *testing.T) {
	if _, err := NewSigner("short", "timer.scheduler.v1"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewSigner(short) error = %v, want ErrInvalidConfig", err)
	}
	if _, err := NewSigner(strings.Repeat("x", 32), ""); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("NewSigner(empty scope) error = %v, want ErrInvalidConfig", err)
	}
}

func mustSigner(t *testing.T, scope string) *Signer {
	t.Helper()
	signer, err := NewSigner(testControlSecret, scope)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func mustVerifier(t *testing.T, scope string, opts VerifierOptions) *Verifier {
	t.Helper()
	verifier, err := NewVerifier(testControlSecret, scope, opts)
	if err != nil {
		t.Fatal(err)
	}
	return verifier
}
