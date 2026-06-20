package dto

type EmailSettings struct {
	Mode        string `json:"mode"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	PasswordSet bool   `json:"password_set"`
	From        string `json:"from"`
	FromName    string `json:"from_name"`
}

type SystemSettingsResp struct {
	RegistrationMode string        `json:"registration_mode"`
	Email            EmailSettings `json:"email"`
}

type TLSCertificateInfo struct {
	Subject      string   `json:"subject"`
	Issuer       string   `json:"issuer"`
	DNSNames     []string `json:"dns_names"`
	IPAddresses  []string `json:"ip_addresses"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	IsSelfSigned bool     `json:"is_self_signed"`
}

type TLSSettingsResp struct {
	Mode            string              `json:"mode"`
	BaseURL         string              `json:"base_url"`
	CertFile        string              `json:"cert_file"`
	KeyFile         string              `json:"key_file"`
	CertExists      bool                `json:"cert_exists"`
	KeyExists       bool                `json:"key_exists"`
	Ready           bool                `json:"ready"`
	Writable        bool                `json:"writable"`
	ReloadSupported bool                `json:"reload_supported"`
	Certificate     *TLSCertificateInfo `json:"certificate,omitempty"`
	Message         string              `json:"message,omitempty"`
}

type UpdateTLSCertificateReq struct {
	CertificatePEM string `json:"certificate_pem" binding:"required"`
	PrivateKeyPEM  string `json:"private_key_pem" binding:"required"`
	Reload         bool   `json:"reload"`
}

type UpdateSystemSettingsReq struct {
	RegistrationMode string        `json:"registration_mode" binding:"required,oneof=admin_only email_code debug_code"`
	Email            EmailSettings `json:"email"`
}

type TestEmailReq struct {
	To string `json:"to" binding:"required,email"`
}
