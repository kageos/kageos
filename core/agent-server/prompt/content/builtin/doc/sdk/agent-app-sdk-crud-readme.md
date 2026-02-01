# Agent-App SDK 快速入门

## 📚 什么是 Agent-App SDK？

Agent-App SDK 是一个用于**快速构建 CRUD 管理系统**的 Go 语言框架。通过定义结构体和配置标签，可以**自动生成**完整的前端表格、搜索、分页、新增、编辑、删除等功能。

**核心特点**：

- ✅ **代码极简**：200行代码实现一个完整的管理系统
- ✅ **组件丰富**：支持 20+ 种前端组件类型
- ✅ **功能强大**：支持复杂业务逻辑、权限控制、数据验证
- ✅ **开发高效**：从 Excel 到系统，19秒生成

---

## 🚀 快速开始（5分钟）

### 第1步：定义结构体

```go
type CrmTicket struct {
    // ⚠️ 系统字段（必须包含）
    ID        int             `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"`                                      // 主键ID
    CreatedAt int64           `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"` // 创建时间（毫秒级时间戳）
    CreateBy  string          `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"`                                                         // 创建用户（只读）
    DeletedAt gorm.DeletedAt  `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`                                                                                      // 软删除标记（隐藏）
    DeletedBy string          `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`                                                                                            // 删除用户（隐藏）
  
    // 业务字段
    Title    string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"`                                          // input - 文本输入框
    Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger;default:中" search:"in" validate:"required,oneof=低 中 高"` // select - 下拉选择
    Handler  string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user;default:Me()" search:"in"`                                                                // user - 用户选择器（默认当前用户）
}

// TableName 指定数据库表名
func (t *CrmTicket) TableName() string {
    return "crm_ticket"
}
```

### 第2步：配置模板和注册路由

```go
// CrmTicketTemplate 表格模板配置
var CrmTicketTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name:         "工单管理",
        CreateTables: []interface{}{&CrmTicket{}}, // 自动创建表
    },
    AutoCrudTable: &CrmTicket{}, // 自动生成 CRUD
    // 可选：添加回调函数处理业务逻辑（OnTableAddRow、OnTableUpdateRow、OnTableDeleteRows）
}

// 列表查询函数
func CrmTicketList(ctx *app.Context, resp response.Response) error {
    var req struct{ *query.SearchFilterPageReq }
    ctx.ShouldBind(&req)
    var lists []*CrmTicket
    return resp.Table(&lists).AutoSearchFilterPaged(ctx.GetGormDB(), &CrmTicket{}, req.SearchFilterPageReq).Build()
}

// init 函数：注册路由
func init() {
    // packageContext 由脚手架生成，直接使用即可
    packageContext.GET("crm_ticket", CrmTicketList, CrmTicketTemplate)
}
```

完成！🎉 系统自动生成完整的 CRUD 功能。

---

## 📖 完整示例：工单管理系统

以下是一个完整的工单管理系统，展示了所有常用组件和功能：

```go

package ticket

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"gorm.io/gorm"
)

type CrmTicket struct {
	// 框架标签：runner:"name:工单ID" - 设置字段在前端的显示名称
	// 框架标签：permission:"read" - 字段只读权限（不能编辑）
	// 注意：gorm:"column:id" 明确指定数据库列名，确保映射正确
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"` //在table 表格里只读，不能编辑

	// 框架标签：widget:"type:timestamp;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件
	// 注意：gorm:"autoCreateTime:milli" 自动填充创建时间（毫秒级时间戳，必须是毫秒级别）
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime:milli;column:created_at"  widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`

	// 框架标签：widget:"type:timestamp;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件，（毫秒级时间戳，必须是毫秒级别）
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`

	// 框架标签：widget:"-" - 隐藏字段（不在前端显示）
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` //不做展示

	// 删除用户：记录删除操作的用户，方便后续恢复数据
	DeletedBy string `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 框架标签：widget:"type:input" - 文本输入框组件
	// 框架标签：search:"like" - 启用模糊搜索功能
	// 框架标签：validate:"required,min=2,max=200" - 必填字段，长度2-200字符
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"` //该字段支持模糊搜索，同时新增时候前端会验证validate，后端sdk内部也会验证

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	// 框架标签：validate:"required,min=10" - 必填字段，至少10字符
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" search:"like" validate:"required,min=10"`

	// 框架标签：widget:"type:select;options:低,中,高;options_colors:success,warning,danger;default:中" - 下拉选择组件（选项：低/中/高）
	// options_colors 支持预设颜色（warning,info,success,danger,primary）和自定义颜色（如：#FF9800 橙色，#9C27B0 紫色）
	// 框架标签：validate:"required,oneof=低 中 高" - 必填字段，值必须是选项之一
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger;default:中" search:"in" validate:"required,oneof=低 中 高"`

	// 框架标签：widget:"type:select;options:待处理,处理中,已完成,已关闭;options_colors:info,warning,success,danger;default:待处理" - 下拉选择组件
	// options_colors 支持预设颜色（warning,info,success,danger,primary）和自定义颜色（如：#FF9800 橙色，#9C27B0 紫色）
	// 框架标签：validate:"required,oneof=待处理 处理中 已完成 已关闭" - 值必须是有效状态
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Status string `json:"status" gorm:"column:status"  widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:info,warning,success,danger;default:待处理" search:"in" validate:"required,oneof=待处理 处理中 已完成 已关闭"`

	Classify string `json:"classify" gorm:"column:classify"  widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:info,warning,success,danger,#FF9800,#9C27B0" search:"in" validate:"required,oneof=民生 交通 医疗 就业 建议 其他"`

	// 框架标签：widget:"type:switch" - 开关组件，用于布尔值字段
	// 开关组件支持 bool 类型，true 表示开启，false 表示关闭
	IsUrgent bool `json:"is_urgent" gorm:"column:is_urgent;default:false" widget:"name:是否紧急;type:switch" search:"eq"`

	// 框架标签：widget:"type:slider;min:0;max:100;unit:%" - 滑块/进度条组件
	// 输入模式：显示为滑块，用于编辑/新增表单
	// 输出模式：显示为进度条，自动显示百分比和状态颜色（>80% success, 50-80% warning, <50% danger）
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：min（最小值，必需）、max（最大值，必需）、unit（单位，可选）
	// 其他功能（提示、百分比、状态颜色等）自动处理，无需配置
	Progress int `json:"progress" gorm:"column:progress;default:0" widget:"name:完成进度;type:slider;min:0;max:100;unit:%" search:"gte,lte" validate:"min=0,max=100"`

	// 框架标签：widget:"type:slider;min:0;max:10;step:0.1;unit:分" - 滑块/进度条组件（浮点数类型）
	// 支持浮点数和整数，例如：8.5、9.0、10 等
	// 输入模式：显示为滑块，支持小数步长（step:0.1）
	// 输出模式：显示为进度条，自动显示百分比和状态颜色
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：min（最小值，必需）、max（最大值，必需）、step（步长，可选，默认1）、unit（单位，可选）
	Score float64 `json:"score" gorm:"column:score;default:0" widget:"name:满意度评分;type:slider;min:0;max:10;step:0.1;unit:分" search:"gte,lte" validate:"min=0,max=10"`

	// 框架标签：widget:"type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好" - 评分组件
	// 输入模式：显示为星级评分（1-5星），支持半星评分
	// 输出模式：显示评分值和星级
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：max（最大星级，默认5）、allow_half（是否允许半星，默认false）、texts（自定义文字数组，可选）
	// 注意：如果配置了 texts，会自动显示文字；如果没有配置 texts，则不显示文字
	Rating float64 `json:"rating" gorm:"column:rating;default:0" widget:"name:评价;type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好" search:"gte,lte" validate:"min=0,max=5"`

	// 框架标签：validate:"required,min=11,max=20" - 必填字段，长度11-20字符
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" search:"like" validate:"required,min=11,max=20"`

	// 工单来源：工单的来源渠道（单选，选项较少）
	// 框架标签：widget:"type:radio;options:电话,邮件,在线,现场,其他;default:在线" - 单选框组件
	// 框架标签：validate:"required,oneof=电话 邮件 在线 现场 其他" - 必填字段，值必须是选项之一
	Source string `json:"source" gorm:"column:source" widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;default:在线" search:"in" validate:"required,oneof=电话 邮件 在线 现场 其他"`

	// 预期处理时长：预计处理该工单需要的时间（单位：分钟）
	// 框架标签：widget:"type:number;step:1;unit:分钟;default:60" - 数字输入组件，默认60分钟（1小时）
	ExpectedDuration int `json:"expected_duration" gorm:"column:expected_duration;default:60" widget:"name:预期处理时长;type:number;step:1;unit:分钟;default:60" search:"gte,lte" validate:"min=1,max=10080"` // 最大10080分钟（7天）

	// 处理耗时：实际处理该工单花费的时间（单位：分钟，自动计算，只读）
	// 框架标签：widget:"type:float;precision:2;unit:分钟" - 浮点数组件，保留两位小数
	// 当工单状态更新为"已关闭"时，自动根据创建时间计算处理耗时
	HandleDuration float64 `json:"handle_duration" gorm:"column:handle_duration;type:decimal(10,2);default:0" widget:"name:处理耗时;type:float;precision:2;unit:分钟;default:0.00" search:"gte,lte" permission:"read"`

	//处理人：type:user 前端会自动在输入时候渲染成用户选择器，输出时候渲染成用户展示
	Handler string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user;default:Me()" search:"in"`

	// 抄送人：type:users 前端会自动在输入时候渲染成多用户选择器，输出时候渲染成多个用户展示
	// 值使用逗号分隔的字符串存储（如 "user1,user2"），便于存储到数据库
	// 框架标签：search:"contains" - 使用 FIND_IN_SET 进行包含查询（用于多选场景）
	// 框架标签：default:Me(),MyLeader() - 默认抄送当前用户和上级领导
	CcUsers string `json:"cc_users" gorm:"column:cc_users" widget:"name:抄送人;type:users;default:Me(),MyLeader()" search:"contains"`

	// 处理部门：处理这个工单的组织架构，默认是创建用户所在部门
	// 框架标签：widget:"type:department;default:MyDepartment()" - 组织架构选择器组件，默认当前用户所在部门
	// 框架标签：search:"in" - 支持多选搜索（使用 IN 查询）
	HandleDepartment string `json:"handle_department" gorm:"column:handle_department" widget:"name:处理部门;type:department;default:MyDepartment()" search:"in"`

	// 关联部门：工单关联的多个部门（用于跨部门协作）
	// 框架标签：widget:"type:departments;max_count:5" - 多组织架构选择器组件，最多选择5个部门
	// 可选参数：
	//   - default:MyDepartment() - 默认值，支持函数调用 MyDepartment()（当前用户所在部门），多个值用逗号分隔
	//   - max_count:5 - 最大选择数量，0表示不限制（例如：max_count:5 表示最多选择5个部门）
	// 值使用逗号分隔的字符串存储（如 "/dept1,/dept2"），便于存储到数据库
	// 框架标签：search:"contains" - 使用 FIND_IN_SET 进行包含查询（用于多选场景）
	RelatedDepartments string `json:"related_departments" gorm:"column:related_departments" widget:"name:关联部门;type:departments;max_count:5" search:"contains"`

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	Remark string `json:"remark" gorm:"column:remark"  widget:"name:备注;type:text_area" search:"like"`

	// 框架标签：widget:"name:详细内容;type:richtext;height:400" - 富文本编辑器组件
	// 输入模式：显示为富文本编辑器，支持格式化文本（粗体、斜体、标题、列表等）
	// 输出模式：显示 HTML 内容
	// 搜索模式：支持文本搜索
	// 参数说明：height（编辑器高度，单位px，默认300）
	Content string `json:"content" gorm:"column:content;type:text" widget:"name:详细内容;type:richtext;height:400" search:"like"`

	// 框架标签：widget:"type:multiselect;options:紧急,重要,普通,低优先级;options_colors:#FF5722,#FF9800,#4CAF50,#9E9E9E" - 多选标签组件
	// options_colors 支持预设颜色（warning,info,success,danger,primary）和自定义颜色（如：#FF5722 深橙，#FF9800 橙色，#4CAF50 绿色，#9C27B0 紫色）
	// 每个颜色对应一个选项，可以重复使用相同颜色
	// 注意：multiselect 字段使用 string 类型而非 []string，因为 []string 无法直接写入数据库
	// 前端会通过逗号分隔选项来传递多选的值，例如："紧急,重要" 表示选择了"紧急"和"重要"两个选项
	// 框架标签：search:"contains" - 使用 FIND_IN_SET 进行包含查询（用于多选场景）
	Tags string `json:"tags" gorm:"column:tags" widget:"name:标签;type:multiselect;options:紧急,重要,普通,低优先级;options_colors:#FF5722,#FF9800,#4CAF50,#9E9E9E" search:"contains"`

	// 框架标签：widget:"type:color;format:hex;default:#409EFF" - 颜色选择器组件
	// 输入模式：显示为颜色选择器（支持 hex、rgb、rgba 格式）
	// 输出模式：显示颜色块和颜色值
	// 搜索模式：支持文本搜索
	// 参数说明：format（颜色格式：hex/rgb/rgba，默认hex）、default（默认颜色，可选）、show_alpha（是否显示透明度选择器，默认false）
	ThemeColor string `json:"theme_color" gorm:"column:theme_color" widget:"name:主题颜色;type:color;format:hex;default:#409EFF" search:"like"`

	//请求参数里是文件上传组件，如果要存数据库必须是type:json类型
	Attachment *types.Files `json:"attachment" gorm:"column:attachment;type:json"  widget:"name:附件;type:files" search:"like"`

	// 这个字段非必要，纯粹展示怎么获取当前用户的组织架构，正常来讲CreateBy是非常必要的字段
	// 所属部门：工单提单的部门，默认是创建用户所在部门
	// read 表示只读，表示要后端赋值的，可以通过ctx.GetRequestUserDept()来获取当前用户的部门，非read的字段前端界面会自动渲染成组织架构选择器选择，
	// 框架标签：search:"in" - 支持多选搜索（使用 IN 查询）
	Department string `json:"department" gorm:"column:department" widget:"name:提单部门;type:department" search:"in" permission:"read"`

	// 创建用户：用户组件
	CreateBy string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" search:"in" permission:"read"` //read 表示只读，表示要后端赋值的，可以通过ctx.GetRequestUser()获取当前用户，非read的字段前端界面会自动渲染成用户选择器进行选择

	// 截止时间：工单要求完成的时间（毫秒时间戳）
	Deadline int64 `json:"deadline" gorm:"column:deadline;default:0" widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"`

	// 剩余时间：仅展示用，不落库。在列表 Build 之后根据当前时间与截止时间计算
	RemainingTime string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:text" permission:"read"`
}

func (t *CrmTicket) TableName() string {
	return "crm_ticket"
}

var CrmTicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "工单管理",
		Tags:     []string{"工单管理系统"},
		Desc:     "一个简单的工单管理系统 ........",
		Request:  &CrmTicketListReq{},
		Response: []*CrmTicket{},
		CreateTables: []interface{}{
			&CrmTicket{},
		},
	},
	AutoCrudTable: &CrmTicket{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CrmTicket
		if err := ctx.ShouldBindValidate(&row); err != nil { //这里内部会用validate的库验证validate的标签
			return nil, err
		}
		row.CreateBy = ctx.GetRequestUser() //获取请求用户
		row.Department = ctx.GetRequestUserDept()
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create crm_ticket err: %v", err)
			return nil, err
		}
		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields CrmTicket
		if err := req.BindUpdates(&updateFields); err != nil { //这里不会验证validate，为啥？因为前端只传递了变更的字段，所以无需验证，所以updateFields只有 更新的字段才会有值，没更新的字段是零值
			return nil, err
		}
		//注意：updateFields主要是方便安全的操作变更的字段，如果更新数据我们还是配合用req.GetUpdates()来，这样例如某些字符串想更新成空，或者int想更新成0是可以实现的，
		//用updateFields 的话，gorm是无法更新零值的

		updates := req.GetUpdates()

		// 判断状态是否更新为"已关闭"，如果是则自动计算处理耗时
		if req.IsFieldUpdated("status") && updateFields.Status == "已关闭" {
			// 获取当前工单的创建时间
			var currentTicket CrmTicket
			if err := db.First(&currentTicket, req.GetId()).Error; err != nil {
				return nil, err
			}

			// 计算处理耗时（分钟）：(当前时间 - 创建时间) / 1000 / 60
			// CreatedAt 是毫秒级时间戳，需要转换为分钟
			now := time.Now().UnixMilli()
			durationMinutes := float64(now-currentTicket.CreatedAt) / 1000.0 / 60.0

			// 保留两位小数（四舍五入）
			updates["handle_duration"] = float64(int(durationMinutes*100+0.5)) / 100.0
		} //-> map[string]interface{} 这里值包含此次变更的字段，例如用户把status 变更成：“已完成”，那么这里的map就是只有一个 status:已完成
		err := db.Model(&CrmTicket{}).Where("id = ?", req.GetId()).Updates(updates).Error //这是标准的更新方式
		if err != nil {
			return nil, err
		}
		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		// 软删除：手动更新 deleted_by 和 deleted_at，方便后续恢复数据
		err := db.Model(&CrmTicket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(), // 记录删除用户
			"deleted_at": time.Now(),           // 记录删除时间
		}).Error
		if err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type CrmTicketListReq struct {
	*query.SearchFilterPageReq //前端会传递符合框架规范的查询字符串，里面包含AutoCrudTable这里这张表的字段相关的 查询，排序，分页等等参数，后端无需关心这些
}

func CrmTicketList(ctx *app.Context, resp response.Response) error {
	var req CrmTicketListReq

	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "CrmTicketSearch ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	var lists []*CrmTicket
	//直接把SearchFilterPageReq透传到框架里，框架可以直接处理内部的逻辑，最终返回的数据是lists，同时会包含分页的信息
	err := resp.Table(&lists).AutoSearchFilterPaged(db, &CrmTicket{}, req.SearchFilterPageReq).Build() // 执行Build的时候会从数据库获取数据
	if err != nil {
		logger.Errorf(ctx, "CrmTicketSearch err: %v", err)
		return err
	}

	//根据当前时间和截止时间计算剩余时间（仅展示）（此时lists已经有数据了）我们可以在查询到数据后做任意的操作
	now := time.Now().UnixMilli()
	for _, item := range lists {
		if item.Deadline <= 0 {
			continue
		}
		if now >= item.Deadline {
			item.RemainingTime = "已过期"
			continue
		}
		diffMs := item.Deadline - now
		diffSec := diffMs / 1000
		days := diffSec / 86400
		hours := (diffSec % 86400) / 3600
		mins := (diffSec % 3600) / 60
		if days > 0 {
			item.RemainingTime = fmt.Sprintf("%d天%d小时", days, hours)
		} else if hours > 0 {
			item.RemainingTime = fmt.Sprintf("%d小时%d分钟", hours, mins)
		} else if mins > 0 {
			item.RemainingTime = fmt.Sprintf("%d分钟", mins)
		} else {
			item.RemainingTime = "不足1分钟"
		}
	}
	return nil
}

func init() {
	// 💡packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	packageContext.GET("crm_ticket", CrmTicketList, CrmTicketTemplate)
}
```

---

## 🎨 组件类型详解

### 1. 基础输入组件

#### input - 文本输入框

```go
Title string `widget:"name:标题;type:input;placeholder:请输入标题" search:"like" validate:"required,min=2,max=200"`
```

#### text_area - 多行文本

```go
Description string `widget:"name:描述;type:text_area;placeholder:请输入描述" search:"like" validate:"required,min=10"`
```

#### richtext - 富文本编辑器

```go
Content string `gorm:"type:text" widget:"name:详细内容;type:richtext;height:400" search:"like"`
```

### 2. 选择组件

#### select - 下拉选择（单选）

```go
// 基础用法
Status string `widget:"name:状态;type:select;options:待处理,处理中,已完成;default:待处理" validate:"required,oneof=待处理 处理中 已完成"`

// 带颜色（预设颜色：info/success/warning/danger/primary）
Priority string `widget:"name:优先级;type:select;options:低,中,高;options_colors:success,warning,danger;default:中" validate:"required,oneof=低 中 高"`

// 自定义颜色（支持 hex 格式）
Classify string `widget:"name:问题分类;type:select;options:民生,交通,医疗;options_colors:#FF9800,#9C27B0,#4CAF50" validate:"required,oneof=民生 交通 医疗"`
```

#### radio - 单选框（选项较少时使用，2-5个）

```go
Source string `widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;default:在线" validate:"required,oneof=电话 邮件 在线 现场 其他"`
```

#### multiselect - 多选下拉

```go
// 注意：字段类型必须是 string（不是 []string），使用逗号分隔存储
Tags string `widget:"name:标签;type:multiselect;options:紧急,重要,普通;options_colors:#FF5722,#FF9800,#4CAF50" search:"contains"`
```

**组件选择建议**：

- 选项 > 5 个：用 `select`
- 选项 2-5 个：用 `radio`（更直观）
- 多选：用 `multiselect`

### 3. 数字输入组件

#### number - 整数输入

```go
EstimatedHours int `widget:"name:预计工时;type:number;step:1;unit:小时;default:0" validate:"min=0,max=1000"`
```

#### float - 浮点数输入

```go
Price float64 `gorm:"type:decimal(10,2)" widget:"name:价格;type:float;precision:2;step:0.01;unit:元;default:0.00" validate:"min=0.01"`
```

#### slider - 滑块（进度、评分）

```go
// 进度条（整数）
Progress int `widget:"name:完成进度;type:slider;min:0;max:100;unit:%" validate:"min=0,max=100"`

// 评分（浮点数）
Score float64 `widget:"name:满意度评分;type:slider;min:0;max:10;step:0.1;unit:分" validate:"min=0,max=10"`
```

#### rate - 星级评分

```go
Rating float64 `widget:"name:评价;type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好" validate:"min=0,max=5"`
```

### 4. 日期时间组件

#### timestamp - 日期时间选择器

- **约定**：后端**无需在代码里做日期格式化**，字段类型用 **int64**，直接存、传**毫秒时间戳**即可；前端会根据 widget 的 `format` 自动格式化展示。
- **错误写法**：使用 `string` 类型并在后端格式化为 "YYYY-MM-DD HH:mm:ss" 等字符串（如 `BidTime string` + 代码里 `time.Format(...)`）。timestamp 的 `format` 仅用于前端展示，后端只返回时间戳。

```go
// 自动填充（创建时间、更新时间）
CreatedAt int64 `gorm:"autoCreateTime:milli" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`
UpdatedAt int64 `gorm:"autoUpdateTime:milli" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`

// 手动输入（截止时间）
Deadline int64 `widget:"name:截止时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss;default:Tomorrow()"`
```

**时间函数支持**：

- `Now()` - 当前时间
- `Now(+1h)` - 一小时后
- `Now(-2d)` - 两天前
- `Today()` - 今天 00:00:00
- `Tomorrow()` - 明天 00:00:00
- `Yesterday()` - 昨天 00:00:00

### 5. 用户和部门组件

#### user - 用户选择器

```go
// 默认当前用户（可编辑字段，前端自动填充默认值，用户可以修改）
Handler  string `widget:"name:处理人;type:user;default:Me()" search:"in"`

// 默认上级领导（可编辑字段，前端自动填充默认值，用户可以修改）
Approver string `widget:"name:审批人;type:user;default:MyLeader()" search:"in"`

// 只读字段（permission:"read"），必须在回调中手动填充
// ⚠️ 注意：只读字段不会显示，Me() 不会触发，必须在 OnTableAddRow 中手动填充
CreateBy string `widget:"name:创建用户;type:user" search:"in" permission:"read"`
// 在回调中：row.CreateBy = ctx.GetRequestUser()
```

#### users - 多用户选择器

```go
// 默认当前用户和上级领导
CcUsers string `widget:"name:抄送人;type:users;default:Me(),MyLeader()" search:"contains"`

// 最多选择5个用户
Reviewers string `widget:"name:审核人;type:users;max_count:5" search:"contains"`
```

#### department - 部门选择器

```go
// 默认当前用户所在部门（可编辑字段，前端自动填充默认值，用户可以修改）
HandleDepartment string `widget:"name:处理部门;type:department;default:MyDepartment()" search:"in"`

// 只读字段（permission:"read"），必须在回调中手动填充
// ⚠️ 注意：只读字段不会显示，MyDepartment() 不会触发，必须在 OnTableAddRow 中手动填充
Department string `widget:"name:提单部门;type:department" search:"in" permission:"read"`
// 在回调中：row.Department = ctx.GetRequestUserDept()
```

#### departments - 多部门选择器

```go
RelatedDepartments string `widget:"name:关联部门;type:departments;max_count:5" search:"contains"`
```

**动态默认值函数**（⚠️ 前端逻辑，只适用于可编辑字段）：

- `Me()` - 当前登录用户（前端自动填充，用户可以修改）
- `MyLeader()` - 当前用户的上级领导（前端自动填充，用户可以修改）
- `MyDepartment()` - 当前用户所在部门（前端自动填充，用户可以修改）

**重要说明**：

- 这些函数是前端逻辑，只在字段显示时才会触发
- 如果字段设置了 `permission:"read"`，字段不会显示，函数也不会触发
- 只读字段必须在后端回调中手动填充（如：`row.CreateBy = ctx.GetRequestUser()`）

### 6. 其他组件

#### switch - 开关

```go
IsEnabled bool `widget:"name:是否启用;type:switch;default:true" search:"eq"`
```

#### color - 颜色选择器

```go
ThemeColor string `widget:"name:主题颜色;type:color;format:hex;default:#409EFF" search:"like"`
```

#### files - 文件上传

```go
// 注意：字段类型必须是 *types.Files，数据库类型必须是 json
Attachment *types.Files `gorm:"type:json" widget:"name:附件;type:files;max_size:10MB;max_count:5"`
```

---

## 🏷️ 标签详解

### widget 标签

`widget` 标签用于配置前端组件，格式：`widget:"name:显示名称;type:组件类型;配置项:值"`

**常用配置项**：

- `name` - 字段显示名称（必需）
- `type` - 组件类型（必需）
- `default` - 默认值
- `placeholder` - 占位符
- `options` - 选项列表（select/radio/multiselect）
- `options_colors` - 选项颜色
- `max_size` - 最大文件大小（files）
- `max_count` - 最大数量（files/users/departments）
- `format` - 格式（timestamp/color）
- `precision` - 精度（float）
- `step` - 步长（number/float/slider）
- `unit` - 单位
- `min/max` - 最小值/最大值（slider）

### validate 标签

`validate` 标签用于数据验证（严格按照 `validator/v10` 库标准）：

```go
Title    string  `validate:"required,min=2,max=200"`        // 必填，2-200字符
Priority string  `validate:"required,oneof=低 中 高"`       // 必填，值必须是选项之一
Email    string  `validate:"required,email"`                // 必填，邮箱格式
Age      int     `validate:"required,min=1,max=120"`        // 必填，1-120之间
Price    float64 `validate:"required,min=0.01,max=9999.99"` // 必填，0.01-9999.99
```

**常用验证规则**：

- `required` - 必填
- `min=n` - 最小长度/最小值
- `max=n` - 最大长度/最大值
- `oneof=值1 值2` - 值必须是选项之一（空格分隔）
- `email` - 邮箱格式
- `url` - URL格式

**注意**：如果选项值包含空格，用单引号：`oneof='选项 1' '选项 2'`

### search 标签

`search` 标签用于配置搜索功能：

| 搜索类型   | 说明                    | 适用组件                      | 示例                |
| ---------- | ----------------------- | ----------------------------- | ------------------- |
| `like`     | 模糊搜索（LIKE）        | input/text_area               | `search:"like"`     |
| `in`       | 精确匹配（IN）          | select/radio/user/department  | `search:"in"`       |
| `contains` | 包含查询（FIND_IN_SET） | multiselect/users/departments | `search:"contains"` |
| `eq`       | 精确匹配（=）           | ID/switch                     | `search:"eq"`       |
| `gte,lte`  | 范围搜索（>=, <=）      | timestamp/number/float/slider | `search:"gte,lte"`  |

### permission 标签

`permission` 标签用于控制字段权限：

| 权限值          | 新增表单  | 更新表单  | 列表展示  | 适用场景               |
| --------------- | --------- | --------- | --------- | ---------------------- |
| `read`          | ❌ 不显示 | ❌ 不显示 | ✅ 显示   | 系统字段、自动生成字段 |
| `create`        | ✅ 可编辑 | ❌ 不显示 | ❌ 不显示 | 仅创建时填写的字段     |
| `update`        | ❌ 不显示 | ✅ 可编辑 | ❌ 不显示 | 仅更新时修改的字段     |
| `create,update` | ✅ 可编辑 | ✅ 可编辑 | ❌ 不显示 | 敏感信息字段           |
| 不设置          | ✅ 可编辑 | ✅ 可编辑 | ✅ 显示   | 普通业务字段           |

---

## 🔧 回调函数详解

### OnTableAddRow - 新增行

```go
OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    db := ctx.GetGormDB()
    var row CrmTicket
  
    // 绑定并验证数据
    if err := ctx.ShouldBindValidate(&row); err != nil {
        return nil, err
    }
  
    // 自动填充字段
    row.CreateBy = ctx.GetRequestUser()          // 当前用户
    row.Department = ctx.GetRequestUserDept()    // 当前用户所在部门
  
    // 自动生成订单号（示例）
    // row.OrderNo = generateOrderNo()
  
    // 保存到数据库
    err := db.Create(&row).Error
    if err != nil {
        logger.Errorf(ctx, "Create crm_ticket err: %v", err)
        return nil, err
    }
  
    // 💡 扩展功能：调用第三方接口（企业微信、钉钉等）
    // createWeChatGroup(row.Handler, row.CcUsers)
  
    return &callback.OnTableAddRowResp{Data: &row}, nil
}
```

**关键方法**：

- `ctx.ShouldBindValidate(&row)` - 绑定并验证数据
- `ctx.GetRequestUser()` - 获取当前用户
- `ctx.GetRequestUserDept()` - 获取当前用户部门
- `ctx.GetGormDB()` - 获取数据库连接

### OnTableUpdateRow - 更新行

```go
OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    db := ctx.GetGormDB()
    var updateFields CrmTicket
    if err := req.BindUpdates(&updateFields); err != nil {
        return nil, err
    }
  
    updates := req.GetUpdates() // 只包含变更的字段
  
    // 💡 判断字段是否更新
    if req.IsFieldUpdated("status") {
        // 状态字段被更新了
        if updateFields.Status == "已关闭" {
            // 自动计算处理耗时
            var currentTicket CrmTicket
            if err := db.First(&currentTicket, req.GetId()).Error; err != nil {
                return nil, err
            }
          
            now := time.Now().UnixMilli()
            durationMinutes := float64(now-currentTicket.CreatedAt) / 1000.0 / 60.0
            updates["handle_duration"] = float64(int(durationMinutes*100+0.5)) / 100.0
        }
    }
  
    // 更新数据库
    err := db.Model(&CrmTicket{}).Where("id = ?", req.GetId()).Updates(updates).Error
    if err != nil {
        return nil, err
    }
  
    // 💡 扩展功能：状态流转通知
    // notifyStatusChange(updateFields.Status)
  
    return &callback.OnTableUpdateRowResp{}, nil
}
```

**关键方法**：

- `req.GetId()` - 获取记录 ID
- `req.GetUpdates()` - 获取变更的字段（map[string]interface{}）
- `req.IsFieldUpdated("fieldName")` - 判断字段是否被更新
- `req.BindUpdates(&target)` - 绑定变更字段到结构体

**⚠️ 重要说明**：

1. `req.GetUpdates()` 只包含此次变更的字段，未更新的字段不会出现
2. 必须使用 `db.Model(&CrmTicket{})` 指定模型，否则 GORM 无法确定表名
3. `OnTableUpdateRow` 不会验证 `validate` 标签（因为只更新部分字段）

### OnTableDeleteRows - 删除行

```go
OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
    db := ctx.GetGormDB()
  
    // ⚠️ 推荐做法：软删除（手动更新 deleted_by 和 deleted_at）
    // 好处：记录删除用户和删除时间，方便后续恢复数据
    err := db.Model(&CrmTicket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
        "deleted_by": ctx.GetRequestUser(), // 记录删除用户
        "deleted_at": time.Now(),           // 记录删除时间
    }).Error
  
    // 不推荐：直接使用 db.Delete()（虽然也是软删除，但不会记录删除用户）
    // err := db.Delete(&CrmTicket{}, "id in ?", req.GetIds()).Error
  
    if err != nil {
        return nil, err
    }
    return &callback.OnTableDeleteRowsResp{}, nil
}
```

**关键方法**：

- `req.GetIds()` - 获取要删除的记录 ID 列表（支持批量删除）

**软删除说明**：

- 结构体必须包含 `DeletedAt gorm.DeletedAt` 字段
- 推荐同时添加 `DeletedBy string` 字段，记录删除用户
- 使用 `Updates` 手动更新删除信息，方便后续恢复数据

---

## 📋 列表查询函数

```go
type CrmTicketListReq struct {
    *query.SearchFilterPageReq // 自动处理搜索、过滤、分页
}

func CrmTicketList(ctx *app.Context, resp response.Response) error {
    var req CrmTicketListReq
    if err := ctx.ShouldBind(&req); err != nil {
        logger.Errorf(ctx, "CrmTicketList ShouldBind err: %v", err)
        return err
    }
  
    db := ctx.GetGormDB()
    var lists []*CrmTicket
  
    // 自动处理搜索、过滤、分页
    err := resp.Table(&lists).AutoSearchFilterPaged(db, &CrmTicket{}, req.SearchFilterPageReq).Build()
    if err != nil {
        logger.Errorf(ctx, "CrmTicketList err: %v", err)
        return err
    }
	// 此时lists已经有数据了，可以对字段进行处理了，例如对某些read字段进行计算，等等，或者对某些字段脱敏等等
	
    return nil
}
```

**关键点**：

- `*query.SearchFilterPageReq` - 自动处理搜索、过滤、分页
- `resp.Table(&lists).AutoSearchFilterPaged(...)` - 框架自动处理所有查询逻辑

---

## 🚀 路由注册

### packageContext 说明

`packageContext` 由脚手架生成，用于注册路由和管理包上下文。

**使用方式**：
在 `init()` 函数中直接使用 `packageContext` 注册路由：

```go
func init() {
    // packageContext 由脚手架生成，直接使用即可
    // GET(路由名称, 查询函数, 表格模板)
    packageContext.GET("crm_ticket", CrmTicketList, CrmTicketTemplate)
}
```

**完整示例**：

```go
package ticket

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"

// 在 init() 中注册路由
func init() {
    // 路由名称一般使用下划线命名法（与函数名对应）
    packageContext.GET("crm_ticket", CrmTicketList, CrmTicketTemplate)
    packageContext.GET("crm_order", CrmOrderList, CrmOrderTemplate)
}
```

**注意事项**：

- **不要声明 `packageContext`**：由脚手架生成，直接使用即可

---

## 💡 最佳实践

### 1. 系统字段（每个表格都必须包含）

```go
ID        int             `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"`                                      // 主键ID，自动递增
CreatedAt int64           `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"` // 创建时间，自动填充（毫秒级时间戳）
CreateBy  string          `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"`                                                         // 创建用户，只读（在回调中自动填充）
DeletedAt gorm.DeletedAt  `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`                                                                                      // 删除时间，软删除标记（隐藏）
DeletedBy string          `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`                                                                                            // 删除用户（隐藏，方便后续恢复数据）
```

**说明**：

- `DeletedBy` 字段用于记录删除用户，方便后续恢复数据
- 删除时需要在 `OnTableDeleteRows` 中手动赋值 `deleted_by` 和 `deleted_at`

### 2. 自动生成字段（设置为只读）

```go
// 订单号（自动生成）
OrderNo string `widget:"name:订单号;type:input" permission:"read"`

// 在 OnTableAddRow 中生成
row.OrderNo = generateOrderNo()
```

### 3. 自动计算字段（设置为只读）

```go
// 总金额（自动计算）
TotalAmount float64 `gorm:"type:decimal(10,2)" widget:"name:总金额;type:float;precision:2;unit:元" permission:"read"`

// 在 OnTableUpdateRow 中计算
if req.IsFieldUpdated("quantity") || req.IsFieldUpdated("unit_price") {
    updates["total_amount"] = row.Quantity * row.UnitPrice
}
```

### 4. 多选字段存储

```go
// multiselect - 使用 string 类型（逗号分隔）
Tags string `widget:"name:标签;type:multiselect;options:紧急,重要" search:"contains"`

// users - 使用 string 类型（逗号分隔）
CcUsers string `widget:"name:抄送人;type:users" search:"contains"`

// departments - 使用 string 类型（逗号分隔）
RelatedDepartments string `widget:"name:关联部门;type:departments" search:"contains"`
```

**⚠️ 重要**：多选字段使用 `string` 类型，不是 `[]string`，使用逗号分隔存储。

---

## ⚡ 快速参考：组件选择

| 需求              | 组件类型    | 示例               |
| ----------------- | ----------- | ------------------ |
| 短文本输入        | input       | 标题、电话、邮箱   |
| 长文本输入        | text_area   | 描述、备注         |
| 富文本编辑        | richtext    | 详细内容、公告     |
| 单选（>5个选项）  | select      | 状态、优先级、分类 |
| 单选（2-5个选项） | radio       | 来源、性别         |
| 多选              | multiselect | 标签、分类         |
| 整数输入          | number      | 数量、工时         |
| 小数输入          | float       | 价格、金额         |
| 进度、评分        | slider      | 完成进度、满意度   |
| 星级评分          | rate        | 评价、评分         |
| 布尔值            | switch      | 是否启用、是否紧急 |
| 日期时间          | timestamp   | 创建时间、截止时间 |
| 颜色选择          | color       | 主题颜色、标签颜色 |
| 文件上传          | files       | 附件、图片         |
| 用户选择          | user        | 处理人、负责人     |
| 多用户选择        | users       | 抄送人、审核人     |
| 部门选择          | department  | 处理部门、所属部门 |
| 多部门选择        | departments | 关联部门           |

---

## 🔍 常见问题

### Q1: 如何设置默认值？

在 `widget` 标签中使用 `default:值`：

```go
Status string `widget:"name:状态;type:select;options:待处理,处理中;default:待处理"`
```

### Q2: 如何让字段只读？

使用 `permission:"read"`：

```go
ID int `widget:"name:ID;type:ID" permission:"read"`
```

### Q3: 如何隐藏字段？

使用 `widget:"-"`：

```go
DeletedAt gorm.DeletedAt `widget:"-"`
```

### Q4: 文件上传字段如何定义？

字段类型 `*types.Files`，数据库类型 `json`：

```go
Attachment *types.Files `gorm:"type:json" widget:"name:附件;type:files;max_size:10MB;max_count:5"`
```

### Q5: 如何自动填充当前用户？

**重要说明**：`Me()` 是前端逻辑，只适用于可编辑字段。如果字段设置了 `permission:"read"`，字段不会显示，`Me()` 也不会触发。

**场景1：只读字段（如 CreateBy）**

```go
// ⚠️ 注意：permission:"read" 字段不会显示，Me() 不会触发
CreateBy string `widget:"name:创建用户;type:user" permission:"read"`

// ✅ 必须在回调中手动填充
OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    // ...
    row.CreateBy = ctx.GetRequestUser() // 后端手动填充
    // ...
}
```

**场景2：可编辑字段（如 Handler）**

```go
// ✅ 可编辑字段可以使用 default:Me()，前端会自动填充默认值
Handler string `widget:"name:处理人;type:user;default:Me()"`

// 说明：
// - 前端会自动将当前用户设置为默认值
// - 用户可以修改或删除这个默认值
// - 如果用户不修改，提交时会使用当前用户
```

### Q6: 如何实现状态流转逻辑？

在 `OnTableUpdateRow` 中判断字段更新：

```go
if req.IsFieldUpdated("status") && updateFields.Status == "已关闭" {
    // 执行状态流转逻辑
}
```

### Q7: multiselect 为什么用 string 不是 []string？

`[]string` 无法直接写入数据库，使用 `string` 类型存储逗号分隔的值：

```go
Tags string `widget:"name:标签;type:multiselect;options:紧急,重要"`
// 存储格式：\"紧急,重要\"
```

### Q8: 如何集成第三方接口？

在回调函数中调用：

```go
OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    // ... 创建记录 ...
  
    // 调用企业微信创建群
    createWeChatGroup(row.Handler, row.CcUsers)
  
    return &callback.OnTableAddRowResp{Data: &row}, nil
}
```

---

## 🎯 总结

通过 Agent-App SDK，你可以：

1. **200行代码**实现完整的 CRUD 系统
2. **支持 20+ 种组件**，覆盖所有常见场景
3. **自动处理**搜索、过滤、分页、验证
4. **灵活扩展**，支持复杂业务逻辑

**从 Excel 到系统，只需 19秒！** 🚀
