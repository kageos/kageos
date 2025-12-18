package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// FileSnapshot 文件快照表，记录每个文件每个版本的代码内容
// 每个文件一行记录，方便后续对每个文件进行分析、打标签、标记等
type FileSnapshot struct {
	models.Base
	AppID          int64  `json:"app_id" gorm:"index:idx_app_path_file;comment:应用ID"`
	ServiceTreeID  int64  `json:"service_tree_id" gorm:"index;comment:关联的ServiceTree节点ID（目录节点，package类型）"`
	FullCodePath   string `json:"full_code_path" gorm:"type:varchar(500);index:idx_app_path_file;comment:目录完整路径（如 /luobei/app/hr/attendance）"`
	FileName       string `json:"file_name" gorm:"type:varchar(200);index:idx_app_path_file;comment:文件名（不含 .go 后缀）"`
	RelativePath   string `json:"relative_path" gorm:"type:varchar(500);comment:文件相对路径（如 user.go 或 subdir/user.go）"`
	Content        string `json:"content" gorm:"type:text;comment:文件代码内容"`
	DirVersion     string `json:"dir_version" gorm:"type:varchar(50);index:idx_dir_version;comment:目录版本号（如 v1, v2），用于目录回滚"`
	DirVersionNum  int    `json:"dir_version_num" gorm:"index:idx_dir_version;comment:目录版本号（数字部分，如 1, 2）"`
	FileVersion    string `json:"file_version" gorm:"type:varchar(50);index:idx_file_version;comment:文件版本号（如 v1, v2），用于文件回滚"`
	FileVersionNum int    `json:"file_version_num" gorm:"index:idx_file_version;comment:文件版本号（数字部分，如 1, 2）"`
	AppVersion     string `json:"app_version" gorm:"type:varchar(50);index:idx_app_version;comment:对应的应用版本号（如 v101）"`
	AppVersionNum  int    `json:"app_version_num" gorm:"index:idx_app_version;comment:应用版本号（数字部分，如 101）"`
	FileType       string `json:"file_type" gorm:"type:varchar(50);comment:文件类型（如 go, json, yaml等）"`
	IsCurrent      bool   `json:"is_current" gorm:"default:false;index:idx_is_current;comment:是否为当前版本（用于快速查询当前在用的快照）"`
}

func (FileSnapshot) TableName() string {
	return "file_snapshot"
}
