package emailx

import (
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/config"
)

type smtpDataWriterStub struct {
	closeErr error
}

func (w *smtpDataWriterStub) Write(data []byte) (int, error) { return len(data), nil }
func (w *smtpDataWriterStub) Close() error                   { return w.closeErr }

func TestWriteSMTPDataReturnsFinalServerRejection(t *testing.T) {
	err := writeSMTPData(&smtpDataWriterStub{closeErr: errors.New("550 rejected")}, []byte("message"))
	if err == nil || !strings.Contains(err.Error(), "550 rejected") {
		t.Fatalf("writeSMTPData error = %v, want final SMTP rejection", err)
	}
}

func TestSendHTMLRejectsHeaderInjectionBeforeConnecting(t *testing.T) {
	sender := NewSender(config.EmailSMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	})
	err := sender.SendHTML("user@example.com", "hello\r\nBcc: attacker@example.com", "body")
	if err == nil || !strings.Contains(err.Error(), "非法换行符") {
		t.Fatalf("SendHTML error = %v, want header injection rejection", err)
	}
}
