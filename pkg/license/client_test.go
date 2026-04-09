package license

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestEncryptedLicenseStoreReadWriteDelete(t *testing.T) {
	t.Parallel()

	store, err := newEncryptedLicenseStore(filepath.Join(t.TempDir(), "license.key"))
	if err != nil {
		t.Fatalf("newEncryptedLicenseStore: %v", err)
	}

	data := []byte("encrypted-license")
	if err := store.Write(data); err != nil {
		t.Fatalf("store.Write: %v", err)
	}

	readData, err := store.Read()
	if err != nil {
		t.Fatalf("store.Read: %v", err)
	}
	if string(readData) != string(data) {
		t.Fatalf("unexpected store content: got %q want %q", string(readData), string(data))
	}

	same, err := store.IsSame(data)
	if err != nil {
		t.Fatalf("store.IsSame: %v", err)
	}
	if !same {
		t.Fatal("expected store content to match")
	}

	if err := store.Delete(); err != nil {
		t.Fatalf("store.Delete: %v", err)
	}
	if store.Exists() {
		t.Fatal("expected store file to be deleted")
	}
}

func TestClientApplyLicenseKeyMessageUpdatesManagerAndStore(t *testing.T) {
	t.Parallel()

	client := newTestLicenseClient(t)
	license := validTestLicense()
	encrypted := mustEncryptLicense(t, client.encryptionKey, license)

	msg := &LicenseKeyMessage{
		EncryptedLicense: base64.StdEncoding.EncodeToString(encrypted),
		Algorithm:        "aes-256-gcm",
		Timestamp:        time.Now().Unix(),
	}

	if err := client.applyLicenseKeyMessage(context.Background(), msg, false); err != nil {
		t.Fatalf("applyLicenseKeyMessage: %v", err)
	}

	got := client.manager.GetLicense()
	if got == nil || got.ID != license.ID {
		t.Fatalf("unexpected manager license: %+v", got)
	}

	storeData, err := client.keyStore.Read()
	if err != nil {
		t.Fatalf("keyStore.Read: %v", err)
	}
	if string(storeData) != string(encrypted) {
		t.Fatalf("unexpected stored encrypted license")
	}
}

func TestHandleDeactivateClearsManagerAndDeletesKeyFile(t *testing.T) {
	t.Parallel()

	client := newTestLicenseClient(t)
	license := validTestLicense()
	client.manager.setLicense(license)

	encrypted := mustEncryptLicense(t, client.encryptionKey, license)
	if err := client.keyStore.Write(encrypted); err != nil {
		t.Fatalf("keyStore.Write: %v", err)
	}

	client.handleDeactivate(context.Background())

	if got := client.manager.GetLicense(); got != nil {
		t.Fatalf("expected manager license cleared, got %+v", got)
	}
	if client.keyStore.Exists() {
		t.Fatal("expected encrypted license file to be deleted")
	}
}

func TestDecodeLicensePayloadHelpers(t *testing.T) {
	t.Parallel()

	keyPayload := []byte(`{"encrypted_license":"abc","algorithm":"aes-256-gcm","timestamp":1}`)
	keyMsg, err := decodeLicenseKeyPayload(keyPayload)
	if err != nil {
		t.Fatalf("decodeLicenseKeyPayload: %v", err)
	}
	if keyMsg.EncryptedLicense != "abc" {
		t.Fatalf("unexpected key payload: %+v", keyMsg)
	}

	instructionPayload := []byte(`{"action":"refresh","encrypted_license":"xyz","timestamp":2}`)
	instructionMsg, err := decodeLicenseInstructionPayload(instructionPayload)
	if err != nil {
		t.Fatalf("decodeLicenseInstructionPayload: %v", err)
	}
	if instructionMsg.Action != "refresh" || instructionMsg.EncryptedLicense != "xyz" {
		t.Fatalf("unexpected instruction payload: %+v", instructionMsg)
	}
}

func newTestLicenseClient(t *testing.T) *Client {
	t.Helper()

	store, err := newEncryptedLicenseStore(filepath.Join(t.TempDir(), "license.key"))
	if err != nil {
		t.Fatalf("newEncryptedLicenseStore: %v", err)
	}

	return &Client{
		natsConn:      &nats.Conn{},
		transport:     nil,
		encryptionKey: []byte("0123456789abcdef0123456789abcdef"),
		keyStore:      store,
		manager:       &Manager{},
	}
}

func validTestLicense() *License {
	return &License{
		ID:        "lic-test",
		Edition:   string(EditionEnterprise),
		Customer:  "Test Customer",
		IssuedAt:  FlexibleTime{Time: time.Now().Add(-time.Hour)},
		ExpiresAt: FlexibleTime{Time: time.Now().Add(24 * time.Hour)},
		Features: Features{
			OperateLog: true,
		},
	}
}

func mustEncryptLicense(t *testing.T, encryptionKey []byte, lic *License) []byte {
	t.Helper()

	data, err := json.Marshal(lic)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	return append(nonce, ciphertext...)
}

func TestGetDefaultLicenseKeyPathUsesEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom-license.key")
	t.Setenv("LICENSE_KEY_PATH", path)

	if got := getDefaultLicenseKeyPath(); got != path {
		t.Fatalf("unexpected default key path: got %s want %s", got, path)
	}
}
