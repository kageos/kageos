package secretvault

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

type Vault struct {
	aead   cipher.AEAD
	prefix string
}

type Option func(*Vault)

func WithPrefix(prefix string) Option {
	return func(v *Vault) {
		v.prefix = strings.TrimSpace(prefix)
	}
}

func New(secret string, purpose string, opts ...Option) (*Vault, error) {
	secret = strings.TrimSpace(secret)
	purpose = strings.TrimSpace(purpose)
	if secret == "" {
		return nil, fmt.Errorf("secretvault secret is empty")
	}
	if purpose == "" {
		return nil, fmt.Errorf("secretvault purpose is empty")
	}

	sum := sha256.Sum256([]byte(purpose + ":" + secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	v := &Vault{aead: aead}
	for _, opt := range opts {
		if opt != nil {
			opt(v)
		}
	}
	return v, nil
}

func (v *Vault) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if v == nil || v.aead == nil {
		return "", fmt.Errorf("secretvault is not initialized")
	}
	if v.IsSealed(plaintext) {
		return plaintext, nil
	}

	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := v.aead.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return v.prefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func (v *Vault) Open(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if v == nil || v.aead == nil {
		return "", fmt.Errorf("secretvault is not initialized")
	}

	ciphertext := value
	if v.prefix != "" {
		if !strings.HasPrefix(value, v.prefix) {
			return value, nil
		}
		ciphertext = strings.TrimPrefix(value, v.prefix)
	}

	payload, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	nonceSize := v.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("secretvault cipher payload too short")
	}
	plaintext, err := v.aead.Open(nil, payload[:nonceSize], payload[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (v *Vault) IsSealed(value string) bool {
	return v != nil && v.prefix != "" && strings.HasPrefix(value, v.prefix)
}
