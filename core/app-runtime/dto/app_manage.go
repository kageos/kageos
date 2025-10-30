package dto

type UpdateAppResp struct {
	User       string      `json:"user"`
	App        string      `json:"app"`
	OldVersion string      `json:"old_version"`
	NewVersion string      `json:"new_version"`
	Diff       interface{} `json:"diff,omitempty"`  // API diff 信息
	Error      error       `json:"error,omitempty"` // 回调过程中的错误
}
