package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// BoardPost 版块帖子模型（讨论区/变更记录下的单条帖子，评论回复后续统一做）
type BoardPost struct {
	models.Base
	TreeID       int64  `json:"tree_id" gorm:"type:bigint;not null;index;comment:关联的 ServiceTree 节点 ID（board 类型）"`
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(500);not null;index;comment:完整路径，与版块节点一致，用于多租户隔离"`
	Title        string `json:"title" gorm:"type:varchar(255);not null;comment:标题"`
	Summary      string `json:"summary" gorm:"type:varchar(500);comment:摘要，列表展示；为空时可从正文截取"`
	Cover        string `json:"cover" gorm:"type:text;comment:封面图 URL，多个用逗号分隔，列表与详情展示"`
	Content      string `json:"content" gorm:"type:longtext;comment:正文（富文本 HTML 或 Markdown）"`
	ContentFormat string `json:"content_format" gorm:"type:varchar(20);default:'markdown';comment:正文格式 markdown/html"`
	Author       string `json:"author" gorm:"type:varchar(100);not null;index;comment:发帖人用户名"`
	Status       string `json:"status" gorm:"type:varchar(20);default:'published';comment:状态 draft/published"`
}

// TableName 指定表名
func (*BoardPost) TableName() string {
	return "board_posts"
}
