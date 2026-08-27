package dto

import "time"

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

type LoginAnnouncement struct {
	Enabled  bool   `json:"enabled"`
	Markdown string `json:"markdown"`
}

type UpdateLoginAnnouncementReq struct {
	Enabled  bool   `json:"enabled"`
	Markdown string `json:"markdown"`
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

type SystemResourceComponent struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	PoolKey   string `json:"pool_key"`
	UsedBytes uint64 `json:"used_bytes"`
	Available bool   `json:"available"`
}

type SystemEnvironmentInfo struct {
	Mode              string `json:"mode"`
	Deployment        string `json:"deployment"`
	Containerized     bool   `json:"containerized"`
	ContainerEngine   string `json:"container_engine"`
	ContainerRemote   bool   `json:"container_remote"`
	StorageRootSource string `json:"storage_root_source"`
}

type SystemStoragePool struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Path        string  `json:"path,omitempty"`
	TotalBytes  uint64  `json:"total_bytes"`
	UsedBytes   uint64  `json:"used_bytes"`
	FreeBytes   uint64  `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
	Primary     bool    `json:"primary"`
	Available   bool    `json:"available"`
}

type SystemResourceSnapshot struct {
	CollectedAt       time.Time                 `json:"collected_at"`
	Hostname          string                    `json:"hostname"`
	OperatingSystem   string                    `json:"operating_system"`
	Architecture      string                    `json:"architecture"`
	CPUCores          int                       `json:"cpu_cores"`
	CPUUsedPercent    float64                   `json:"cpu_used_percent"`
	CPUAvailable      bool                      `json:"cpu_available"`
	Load1             float64                   `json:"load_1"`
	Load5             float64                   `json:"load_5"`
	Load15            float64                   `json:"load_15"`
	LoadAvailable     bool                      `json:"load_available"`
	UptimeSeconds     uint64                    `json:"uptime_seconds"`
	MemoryTotalBytes  uint64                    `json:"memory_total_bytes"`
	MemoryUsedBytes   uint64                    `json:"memory_used_bytes"`
	MemoryUsedPercent float64                   `json:"memory_used_percent"`
	MemoryAvailable   bool                      `json:"memory_available"`
	DiskMount         string                    `json:"disk_mount"`
	DiskTotalBytes    uint64                    `json:"disk_total_bytes"`
	DiskUsedBytes     uint64                    `json:"disk_used_bytes"`
	DiskFreeBytes     uint64                    `json:"disk_free_bytes"`
	DiskUsedPercent   float64                   `json:"disk_used_percent"`
	Environment       SystemEnvironmentInfo     `json:"environment"`
	StoragePools      []SystemStoragePool       `json:"storage_pools"`
	Components        []SystemResourceComponent `json:"components"`
}

type SystemResourceHistoryPoint struct {
	CollectedAt       time.Time `json:"collected_at"`
	DiskUsedBytes     uint64    `json:"disk_used_bytes"`
	DiskUsedPercent   float64   `json:"disk_used_percent"`
	MemoryUsedPercent float64   `json:"memory_used_percent"`
	Load1             float64   `json:"load_1"`
}

type StorageExpansionForecast struct {
	Status             string  `json:"status"`
	PoolKey            string  `json:"pool_key"`
	CurrentUsedPercent float64 `json:"current_used_percent"`
	DailyGrowthByte    float64 `json:"daily_growth_bytes"`
	TargetPercent      float64 `json:"target_percent"`
	DaysToTarget       *int    `json:"days_to_target,omitempty"`
	Message            string  `json:"message"`
}

type SystemResourceOverviewResp struct {
	Current               SystemResourceSnapshot       `json:"current"`
	History               []SystemResourceHistoryPoint `json:"history"`
	HistoryHours          int                          `json:"history_hours"`
	SampleIntervalMinutes int                          `json:"sample_interval_minutes"`
	Forecast              StorageExpansionForecast     `json:"forecast"`
}
