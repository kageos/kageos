package cert_manager

const (
	defaultRenewBeforeDays   = 30
	defaultDNSWaitSeconds    = 300
	defaultDNSPollSeconds    = 15
	defaultCertKeyAlgorithm  = "ECDSA P-256"
	defaultLetsEncryptCAName = "Let's Encrypt"

	statusUnchecked = "未检查"
	statusOK        = "正常"
	statusWarning   = "即将过期"
	statusExpired   = "已过期"
	statusFailed    = "失败"
	statusPending   = "待部署"

	requestTypeIssue      = "首次签发"
	requestTypeManual     = "手动续期"
	requestTypeAutoRenew  = "自动续期"
	requestStatusPending  = "待执行"
	requestStatusRunning  = "执行中"
	requestStatusWaitDNS  = "等待DNS生效"
	requestStatusVerify   = "验证中"
	requestStatusIssued   = "签发成功"
	requestStatusFailed   = "失败"
	requestStatusCanceled = "已取消"
)
