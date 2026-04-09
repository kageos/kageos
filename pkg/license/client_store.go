package license

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type encryptedLicenseStore struct {
	path string
}

func newEncryptedLicenseStore(path string) (*encryptedLicenseStore, error) {
	if path == "" {
		path = getDefaultLicenseKeyPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create license key directory: %w", err)
	}
	return &encryptedLicenseStore{path: path}, nil
}

func getDefaultLicenseKeyPath() string {
	if path := os.Getenv("LICENSE_KEY_PATH"); path != "" {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(homeDir, ".ai-agent-os", "license.key")
	}

	return "./license.key"
}

func (s *encryptedLicenseStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *encryptedLicenseStore) Read() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("license key store is nil")
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read local key file: %w", err)
	}
	return data, nil
}

func (s *encryptedLicenseStore) Write(data []byte) error {
	if s == nil {
		return fmt.Errorf("license key store is nil")
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("failed to save license key to local: %w", err)
	}
	return nil
}

func (s *encryptedLicenseStore) Delete() error {
	if s == nil {
		return fmt.Errorf("license key store is nil")
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local key file: %w", err)
	}
	return nil
}

func (s *encryptedLicenseStore) Exists() bool {
	if s == nil {
		return false
	}
	_, err := os.Stat(s.path)
	return err == nil
}

func (s *encryptedLicenseStore) IsSame(data []byte) (bool, error) {
	current, err := s.Read()
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(current, data), nil
}
