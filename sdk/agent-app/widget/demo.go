package widget

import "gorm.io/gorm"

type Demo struct {
	// 框架标签：runner:"name:工单ID" - 设置字段在前端的显示名称
	// 框架标签：permission:"read" - 字段只读权限（不能编辑）
	// 注意：gorm:"column:id" 明确指定数据库列名，确保映射正确
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read"` //在table 表格里只读，不能编辑

	// 框架标签：widget:"type:datetime;kind:datetime" - 日期时间选择器组件
	// 注意：gorm:"autoCreateTime:milli" 自动填充创建时间（毫秒级时间戳，必须是毫秒级别）
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime:milli;column:created_at"  widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`

	// 框架标签：widget:"type:datetime;kind:datetime" - 日期时间选择器组件，（毫秒级时间戳，必须是毫秒级别）
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`

	// 框架标签：runner:"-" - 隐藏字段（不在前端显示）
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` //不做展示

	// 框架标签：widget:"type:input" - 文本输入框组件
	// 框架标签：search:"like" - 启用模糊搜索功能
	// 框架标签：validate:"required,min=2,max=200" - 必填字段，长度2-200字符
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"` //该字段支持模糊搜索，同时新增时候前端会验证validate，后端sdk内部也会验证

	// 框架标签：widget:"type:input;mode:text_area" - 多行文本区域组件
	// 框架标签：validate:"required,min=10" - 必填字段，至少10字符
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`

	// 框架标签：widget:"name:优先级;type:select;options:低,中,高;default:中" - 下拉选择组件（默认值为"中"）
	// 框架标签：validate:"required,oneof=低 中 高" - 必填字段，值必须是选项之一
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;default:中" validate:"required,oneof=低 中 高"`

	// 框架标签：widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理" - 下拉选择组件（默认状态为"待处理"）
	// 框架标签：validate:"required,oneof=待处理 处理中 已完成 已关闭" - 值必须是有效状态
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Status string `json:"status" gorm:"column:status"  widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`

	// 框架标签：validate:"required,min=11,max=20" - 必填字段，长度11-20字符
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`

	// 框架标签：widget:"type:input;mode:text_area" - 多行文本区域组件
	Remark string `json:"remark" gorm:"column:remark"  widget:"name:备注;type:text_area"`

	// 创建用户：用户组件
	CreateBy string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"` //read 表示只读，后端赋值的，非read的字段前端界面会自动渲染成用户选择器进行选择
}
