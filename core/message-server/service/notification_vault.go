package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type NotificationSecretVault struct {
	aead cipher.AEAD
}

func NewNotificationSecretVault(secret string) (*NotificationSecretVault, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("notification encryption secret is empty")
	}
	sum := sha256.Sum256([]byte("kageos-message-notification-v1:" + secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &NotificationSecretVault{aead: aead}, nil
}

func (v *NotificationSecretVault) Seal(plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", nil
	}
	if v == nil || v.aead == nil {
		return "", fmt.Errorf("notification secret vault is not initialized")
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := v.aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (v *NotificationSecretVault) Open(ciphertext string) (string, error) {
	if strings.TrimSpace(ciphertext) == "" {
		return "", nil
	}
	if v == nil || v.aead == nil {
		return "", fmt.Errorf("notification secret vault is not initialized")
	}
	payload, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := v.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("notification cipher payload too short")
	}
	plaintext, err := v.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
