package wecom

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

const wecomSecretCipherPrefix = "v1:"

func encryptWeComSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	gcm, err := newWeComSecretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成企业微信密钥随机数失败: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return wecomSecretCipherPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func decryptWeComSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, wecomSecretCipherPrefix) {
		return value, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, wecomSecretCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("解析企业微信密文失败: %w", err)
	}
	gcm, err := newWeComSecretCipher()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("企业微信密文格式不完整")
	}
	nonce := raw[:gcm.NonceSize()]
	cipherText := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("解密企业微信密文失败: %w", err)
	}
	return string(plain), nil
}

func newWeComSecretCipher() (cipher.AEAD, error) {
	seed := strings.TrimSpace(os.Getenv("KAGEOS_WECOM_SECRET_KEY"))
	if seed == "" {
		seed = strings.TrimSpace(env.User) + ":" + strings.TrimSpace(env.App)
	}
	if strings.Trim(seed, ":") == "" {
		seed = "kageos-system-connector-wecom"
	}
	sum := sha256.Sum256([]byte("kageos-wecom-secret-v1:" + seed))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("初始化企业微信密钥加密器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化企业微信密钥 GCM 失败: %w", err)
	}
	return gcm, nil
}
