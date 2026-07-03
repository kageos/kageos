package service

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/secretvault"
)

type TokenVault struct {
	vault *secretvault.Vault
}

func NewTokenVault(secret string) (*TokenVault, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("connector token encryption secret 不能为空")
	}
	vault, err := secretvault.New(secret, "kageos-connector-token-v1")
	if err != nil {
		return nil, err
	}
	return &TokenVault{vault: vault}, nil
}

func (v *TokenVault) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return v.vault.Seal(plaintext)
}

func (v *TokenVault) Open(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return v.vault.Open(ciphertext)
}
