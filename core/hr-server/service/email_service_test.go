package service

import (
	"strings"
	"testing"

	appconfig "github.com/kageos/kageos/pkg/config"
)

func TestForgotPasswordEmailContainsVerificationCodeNotLink(t *testing.T) {
	emailService := &EmailService{
		config: &appconfig.EmailConfig{
			Verification: appconfig.EmailVerificationConfig{CodeExpire: 600},
		},
	}

	body := emailService.getBody("123456", "forgot_password")
	if !strings.Contains(body, ">123456<") {
		t.Fatal("forgot-password email should contain the verification code")
	}
	if strings.Contains(body, `href="123456"`) {
		t.Fatal("forgot-password verification code must not be rendered as a link")
	}
	if !strings.Contains(body, "10 分钟") {
		t.Fatal("forgot-password email should show the configured expiry")
	}
}
