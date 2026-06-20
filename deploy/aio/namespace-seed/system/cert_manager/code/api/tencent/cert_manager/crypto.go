package cert_manager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/env"
)

const secretCipherPrefix = "v1:"

func encryptSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	gcm, err := newSecretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成密钥随机数失败: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return secretCipherPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, secretCipherPrefix) {
		return value, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, secretCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("解析密文失败: %w", err)
	}
	gcm, err := newSecretCipher()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("密文格式不完整")
	}
	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("解密密文失败: %w", err)
	}
	return string(plain), nil
}

func newSecretCipher() (cipher.AEAD, error) {
	seed := strings.TrimSpace(os.Getenv("KAGEOS_CERT_MANAGER_SECRET_KEY"))
	if seed == "" {
		seed = strings.TrimSpace(env.User) + ":" + strings.TrimSpace(env.App)
	}
	if strings.Trim(seed, ":") == "" {
		seed = "kageos-system-cert-manager-tencent"
	}
	sum := sha256.Sum256([]byte("kageos-cert-manager-tencent-v1:" + seed))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("初始化密钥加密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化密钥 GCM 失败: %w", err)
	}
	return gcm, nil
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 10 {
		return strings.Repeat("*", len(value))
	}
	return value[:6] + strings.Repeat("*", len(value)-10) + value[len(value)-4:]
}
