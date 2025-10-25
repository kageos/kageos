package model

import "github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"

const (
	ServiceTreeTypePackage  = "package"
	ServiceTreeTypeFunction = "function"
)

// ServiceTree 表示服务树模型，一个app下可以有无数个package，一个package下面有无数个function，ServiceTree是一个抽象的树干，这个树干上可以挂载各种实体
// 例如我有个tools的app，然后，我有个excel的package（目录对应go的package），然后下面有多个function（go文件）
type ServiceTree struct {
	models.Base
	Title       string `json:"title"`
	Name        string `json:"name"`
	ParentID    int64  `json:"parent_id" gorm:"default:0"`
	Type        string `json:"type"` //package（服务目录） or function （函数，文件）
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags"`
	AppID       int64  `json:"app_id"`
	//下面字段是数据库
	FullIDPath   string         `json:"full_id_path"`   // /0/2/7 这种
	FullNamePath string         `json:"full_name_path"` // /tools/pdf 这种
	Children     []*ServiceTree `json:"children" gorm:"-"`
}

// TableName 指定表名
func (*ServiceTree) TableName() string {
	return "service_tree"
}
