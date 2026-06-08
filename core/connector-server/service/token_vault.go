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

type TokenVault struct {
	aead cipher.AEAD
}

func NewTokenVault(secret string) (*TokenVault, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("connector token encryption secret 不能为空")
	}
	sum := sha256.Sum256([]byte("kageos-connector-token-v1:" + secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &TokenVault{aead: aead}, nil
}

func (v *TokenVault) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := v.aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (v *TokenVault) Open(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := v.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("connector token cipher payload 太短")
	}
	plaintext, err := v.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
