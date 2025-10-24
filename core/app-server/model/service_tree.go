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
	Type        string `json:"type"` //package or function
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags"`
	AppID       int64  `json:"app_id"`
	//下面字段是数据库
	Level         int            `json:"level" gorm:"default:1"`
	Sort          int            `json:"sort" gorm:"default:0"`
	FullIDPath    string         `json:"full_id_path"`
	FullNamePath  string         `json:"full_name_path"`
	User          string         `json:"user"`
	ChildrenCount int            `json:"children_count" gorm:"default:0"`
	Children      []*ServiceTree `json:"children" gorm:"-"`
}

// TableName 指定表名
func (*ServiceTree) TableName() string {
	return "service_tree"
}
