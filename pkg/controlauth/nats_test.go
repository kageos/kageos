package controlauth

import (
	"errors"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestNATSMessageSignAndVerify(t *testing.T) {
	msg := nats.NewMsg("timer.v1.cmd.execution.requested.agent.session")
	msg.Data = []byte(`{"execution_id":42}`)
	msg.Header.Set("X-Trace-Id", "trace-1")
	if err := SignNATSMessage(msg, mustSigner(t, "timer.scheduler.v1")); err != nil {
		t.Fatal(err)
	}
	if err := VerifyNATSMessage(msg, mustVerifier(t, "timer.scheduler.v1", VerifierOptions{})); err != nil {
		t.Fatalf("VerifyNATSMessage() error = %v", err)
	}
}

func TestNATSMessageVerifyRejectsUnsignedAndTamperedMessages(t *testing.T) {
	unsigned := nats.NewMsg("timer.v1.cmd.execution.started")
	unsigned.Data = []byte(`{"execution_id":42}`)
	if err := VerifyNATSMessage(unsigned, mustVerifier(t, "timer.worker.v1", VerifierOptions{})); !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("unsigned error = %v, want ErrMissingMetadata", err)
	}

	tests := []struct {
		name   string
		tamper func(*nats.Msg)
	}{
		{name: "subject", tamper: func(msg *nats.Msg) { msg.Subject = "timer.v1.cmd.execution.finished" }},
		{name: "reply", tamper: func(msg *nats.Msg) { msg.Reply = "_INBOX.attacker" }},
		{name: "payload", tamper: func(msg *nats.Msg) { msg.Data = []byte(`{"execution_id":99}`) }},
		{name: "business header", tamper: func(msg *nats.Msg) { msg.Header.Set("X-Request-User", "attacker") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := nats.NewMsg("timer.v1.cmd.execution.started")
			msg.Reply = "_INBOX.worker"
			msg.Data = []byte(`{"execution_id":42}`)
			msg.Header.Set("X-Request-User", "system")
			if err := SignNATSMessage(msg, mustSigner(t, "timer.worker.v1")); err != nil {
				t.Fatal(err)
			}
			tt.tamper(msg)
			if err := VerifyNATSMessage(msg, mustVerifier(t, "timer.worker.v1", VerifierOptions{})); !errors.Is(err, ErrInvalidSignature) {
				t.Fatalf("VerifyNATSMessage() error = %v, want ErrInvalidSignature", err)
			}
		})
	}
}

func TestNATSMessageHeaderValueOrderIsAuthenticated(t *testing.T) {
	msg := nats.NewMsg("timer.v1.cmd.execution.started")
	msg.Header["X-Request-User"] = []string{"system", "attacker"}
	if err := SignNATSMessage(msg, mustSigner(t, "timer.worker.v1")); err != nil {
		t.Fatal(err)
	}
	msg.Header["X-Request-User"] = []string{"attacker", "system"}
	if err := VerifyNATSMessage(msg, mustVerifier(t, "timer.worker.v1", VerifierOptions{})); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("VerifyNATSMessage() after value reorder error = %v, want ErrInvalidSignature", err)
	}
}

func TestNATSMessageRejectsDuplicateCanonicalHeaderNames(t *testing.T) {
	msg := nats.NewMsg("timer.v1.cmd.execution.started")
	msg.Header["X-Request-User"] = []string{"system"}
	msg.Header["x-request-user"] = []string{"attacker"}
	if err := SignNATSMessage(msg, mustSigner(t, "timer.worker.v1")); !errors.Is(err, ErrAmbiguousHeaders) {
		t.Fatalf("SignNATSMessage() error = %v, want ErrAmbiguousHeaders", err)
	}
}

func TestNATSMessageRejectsDuplicateAuthenticationHeaderNames(t *testing.T) {
	msg := nats.NewMsg("timer.v1.cmd.execution.started")
	if err := SignNATSMessage(msg, mustSigner(t, "timer.worker.v1")); err != nil {
		t.Fatal(err)
	}
	msg.Header["x-kageos-control-signature"] = append([]string(nil), msg.Header[NATSSignatureHeader]...)
	if err := VerifyNATSMessage(msg, mustVerifier(t, "timer.worker.v1", VerifierOptions{})); !errors.Is(err, ErrAmbiguousHeaders) {
		t.Fatalf("VerifyNATSMessage() error = %v, want ErrAmbiguousHeaders", err)
	}
}
