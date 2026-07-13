package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/kageos/kageos/core/app-runtime/model"
)

func (s *AppDatabaseService) prepareDatabaseCredentials(record *model.AppDatabase) (appDatabasePasswords, error) {
	if record == nil {
		return appDatabasePasswords{}, fmt.Errorf("app database record is nil")
	}
	changed := false
	if strings.TrimSpace(record.DatabaseUser) == "" && record.ID > 0 {
		record.DatabaseUser = runtimeDatabaseUserName(s.cfg.UserPrefix, base62Encode(uint64(record.ID)))
		changed = true
	}
	runtimePasswordMissing := strings.TrimSpace(record.PasswordCiphertext) == "" || strings.TrimSpace(record.PasswordNonce) == ""
	runtimePassword, err := s.ensureEncryptedPassword(&record.PasswordCiphertext, &record.PasswordNonce)
	if err != nil {
		return appDatabasePasswords{}, err
	}
	if strings.TrimSpace(record.MigrationDatabaseUser) == "" && record.ID > 0 {
		record.MigrationDatabaseUser = migrationDatabaseUserName(s.cfg.UserPrefix, base62Encode(uint64(record.ID)))
		changed = true
	}
	migrationPasswordMissing := strings.TrimSpace(record.MigrationPasswordCiphertext) == "" || strings.TrimSpace(record.MigrationPasswordNonce) == ""
	migrationPassword, err := s.ensureEncryptedPassword(&record.MigrationPasswordCiphertext, &record.MigrationPasswordNonce)
	if err != nil {
		return appDatabasePasswords{}, err
	}
	if changed || runtimePasswordMissing || migrationPasswordMissing {
		if err := s.db.Save(record).Error; err != nil {
			return appDatabasePasswords{}, err
		}
	}
	return appDatabasePasswords{runtime: runtimePassword, migration: migrationPassword}, nil
}

func (s *AppDatabaseService) ensureEncryptedPassword(ciphertext, nonce *string) (string, error) {
	if ciphertext == nil || nonce == nil {
		return "", fmt.Errorf("password fields are nil")
	}
	if strings.TrimSpace(*ciphertext) != "" && strings.TrimSpace(*nonce) != "" {
		return s.decryptPassword(*ciphertext, *nonce)
	}
	password, err := randomToken(32)
	if err != nil {
		return "", err
	}
	encrypted, nonceText, err := s.encryptPassword(password)
	if err != nil {
		return "", err
	}
	*ciphertext = encrypted
	*nonce = nonceText
	return password, nil
}

func (s *AppDatabaseService) encryptPassword(password string) (string, string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(password), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), base64.RawURLEncoding.EncodeToString(nonce), nil
}

func (s *AppDatabaseService) decryptPassword(ciphertextText, nonceText string) (string, error) {
	block, err := aes.NewCipher(s.secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(ciphertextText)
	if err != nil {
		return "", err
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceText)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
