package config

// EmailConfig 邮箱配置
type EmailConfig struct {
	Mode         string                  `mapstructure:"mode"`
	SMTP         EmailSMTPConfig         `mapstructure:"smtp"`
	Verification EmailVerificationConfig `mapstructure:"verification"`
}

// EmailSMTPConfig SMTP配置
type EmailSMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	FromName string `mapstructure:"from_name"`
}

// EmailVerificationConfig 邮箱验证配置
type EmailVerificationConfig struct {
	CodeLength int `mapstructure:"code_length"`
	CodeExpire int `mapstructure:"code_expire"`
}
