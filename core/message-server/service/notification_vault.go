package service

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/secretvault"
)

type NotificationSecretVault struct {
	vault *secretvault.Vault
}

func NewNotificationSecretVault(secret string) (*NotificationSecretVault, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("notification encryption secret is empty")
	}
	vault, err := secretvault.New(secret, "kageos-message-notification-v1")
	if err != nil {
		return nil, err
	}
	return &NotificationSecretVault{vault: vault}, nil
}

func (v *NotificationSecretVault) Seal(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", nil
	}
	if v == nil || v.vault == nil {
		return "", fmt.Errorf("notification secret vault is not initialized")
	}
	return v.vault.Seal(plaintext)
}

func (v *NotificationSecretVault) Open(ciphertext string) (string, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return "", nil
	}
	if v == nil || v.vault == nil {
		return "", fmt.Errorf("notification secret vault is not initialized")
	}
	return v.vault.Open(ciphertext)
}
