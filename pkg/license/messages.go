package license

// LicenseKeyMessage License 密钥消息。
type LicenseKeyMessage struct {
	EncryptedLicense string `json:"encrypted_license"` // 加密的 License（Base64 编码）
	Algorithm        string `json:"algorithm"`         // 加密算法（如 "aes-256-gcm"）
	Timestamp        int64  `json:"timestamp"`         // 时间戳
}

// LicenseInstructionMessage License 指令消息（用于刷新和注销）。
type LicenseInstructionMessage struct {
	Action           string `json:"action"`                      // 指令类型：refresh（刷新）、deactivate（注销）
	Timestamp        int64  `json:"timestamp"`                   // 时间戳
	EncryptedLicense string `json:"encrypted_license,omitempty"` // 加密的 License（Base64 编码，可选，refresh 时携带）
	Algorithm        string `json:"algorithm,omitempty"`         // 加密算法（如 "aes-256-gcm"，可选）
}

// LicenseKeyRequestMessage License 密钥请求消息。
type LicenseKeyRequestMessage struct {
	Request string `json:"request"` // 请求类型：license_key
}
