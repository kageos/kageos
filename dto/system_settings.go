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

type UpdateSystemSettingsReq struct {
	RegistrationMode string        `json:"registration_mode" binding:"required,oneof=admin_only email_code debug_code"`
	Email            EmailSettings `json:"email"`
}

type TestEmailReq struct {
	To string `json:"to" binding:"required,email"`
}
