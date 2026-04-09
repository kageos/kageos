package license

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func decodeLicenseKeyPayload(data []byte) (*LicenseKeyMessage, error) {
	var msg LicenseKeyMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pushed license: %w", err)
	}
	return &msg, nil
}

func decodeLicenseInstructionPayload(data []byte) (*LicenseInstructionMessage, error) {
	var msg LicenseInstructionMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal refresh message: %w", err)
	}
	return &msg, nil
}

func decodeEncryptedLicenseBase64(encrypted string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted license: %w", err)
	}
	return data, nil
}
