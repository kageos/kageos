# 案例：工单管理（单 Table）

## 一、项目概要

- **类型**：单表，一个 GET Table，一个 .go，纯列表 CRUD。
- **路由**：`ticket_list.table`（工单管理）。
- **适合参考**：单表 CRUD、input/text_area/select/switch/slider/rate/radio/number、Request 筛选、AutoCrudTable、OnTableAddRow/UpdateRow/DeleteRows 回调。

---

## 二、结构化 PRD JSON

`product_manager` 只输出轻量 PRD v2：`project/tables/forms/charts/rules`。
字段只写 `name/widget/required/desc/hide`，`widget` 只保留组件类型；选项、默认值、范围、数据来源和计算规则写进自然语言 `desc`。
完整标准样例见同目录 `prd.json`。

```json
{
  "kind": "agent_app_prd",
  "schema_version": "prd.v2",
  "project": {
    "name": "工单管理",
    "code": "ticket_management",
    "summary": "管理工单提交、处理、状态流转和列表筛选。"
  },
  "tables": [
    {
      "name": "工单",
      "title": "工单列表",
      "desc": "管理工单记录，支持新增、编辑、删除和按条件筛选。",
      "fields": [
        {"name": "工单标题", "widget": "input", "required": true, "desc": "2-200 字，描述工单核心问题。"},
        {"name": "问题描述", "widget": "text_area", "required": true, "desc": "详细描述问题，建议不少于 10 字。"},
        {"name": "优先级", "widget": "select", "required": true, "desc": "有低、中、高三个选项，默认中。"},
        {"name": "工单状态", "widget": "select", "required": true, "desc": "有待处理、处理中、已完成、已关闭四个选项，默认待处理。"},
        {"name": "问题分类", "widget": "select", "required": true, "desc": "有民生、交通、医疗、就业、建议、其他六个选项。"},
        {"name": "联系电话", "widget": "input", "required": true, "desc": "提交人联系电话，通常为 11-20 位。"},
        {"name": "工单来源", "widget": "radio", "required": true, "desc": "有电话、邮件、在线、现场、其他五个来源，默认在线。"},
        {"name": "处理人", "widget": "user", "required": false, "desc": "负责处理该工单的人员。"},
        {"name": "截止时间", "widget": "datetime", "required": false, "desc": "期望完成处理的截止时间。"},
        {"name": "备注", "widget": "text_area", "required": false, "desc": "处理过程中的补充说明。"}
      ],
      "search_fields": [
        {"name": "工单标题", "widget": "input", "required": false, "desc": "按工单标题模糊搜索。"},
        {"name": "工单状态", "widget": "select", "required": false, "desc": "按待处理、处理中、已完成、已关闭筛选。"},
        {"name": "优先级", "widget": "select", "required": false, "desc": "按低、中、高筛选。"},
        {"name": "问题分类", "widget": "select", "required": false, "desc": "按民生、交通、医疗、就业、建议、其他筛选。"},
        {"name": "处理人", "widget": "user", "required": false, "desc": "按工单处理人筛选。"},
        {"name": "创建人", "widget": "user", "required": false, "desc": "按系统记录的创建人筛选。"},
        {"name": "创建开始时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的开始时间。"},
        {"name": "创建结束时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的结束时间。"}
      ],
      "handlers": ["OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRow"],
      "examples": [
        {"工单标题": "打印机无法连接", "问题描述": "办公室打印机无法连接网络。", "优先级": "中", "工单状态": "待处理", "问题分类": "其他", "联系电话": "13800000000", "工单来源": "在线", "处理人": "李工", "截止时间": "2026-05-10 18:00", "备注": "请优先检查网络。"}
      ]
    }
  ],
  "rules": [
    "新建工单默认状态为待处理。",
    "处理中、已完成、已关闭用于描述工单处理进度。",
    "工单表支持新增、编辑、删除和列表筛选。"
  ]
}
```

---

## 三、文件与路由

| 文件       | 说明     | 注册路由            |
|------------|----------|---------------------|
| ticket.go  | 工单管理 | GET ticket_list.table |

---

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/table/ticket`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### ticket.go

```go
package ticket

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type Ticket struct {
	// 框架标签：runner:"name:工单ID" - 设置字段在前端的显示名称
	// 框架标签：hide:"create,update" - 前端仅在列表展示该字段，不进入新增/编辑表单
	// 注意：gorm:"column:id" 明确指定数据库列名，确保映射正确
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	// 框架标签：widget:"type:datetime;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件
	// 注意：gorm:"type:datetime;autoCreateTime" 自动填充创建时间（真实时间列）
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime"  widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	// 框架标签：widget:"type:datetime;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件，（真实时间列）
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	// 框架标签：widget:"-" - 隐藏字段（不在前端显示）
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` //不做展示

	// 删除用户：记录删除操作的用户，方便后续恢复数据
	DeletedBy string `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 框架标签：widget:"type:input" - 文本输入框组件
	// 列表筛选：在 TicketListReq 中声明 Title，并在 Handler 里手写 LIKE 条件
	// 框架标签：validate:"required,min=2,max=200" - 必填字段，长度2-200字符
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"` // 该字段可在 Request 中显式声明为筛选条件；新增时前端和后端都会验证 validate

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	// 框架标签：validate:"required,min=10" - 必填字段，至少10字符
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`

	// 框架标签：widget:"type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" - 下拉选择组件（选项：低/中/高）
	// options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，且数量必须和 options 一致。
	// 框架标签：validate:"required,oneof=低 中 高" - 必填字段，值必须是选项之一
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`

	// 框架标签：widget:"type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" - 下拉选择组件
	// options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，且数量必须和 options 一致。
	// 框架标签：validate:"required,oneof=待处理 处理中 已完成 已关闭" - 值必须是有效状态
	// 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来，例如：oneof='选项 1' '选项 2'
	Status string `json:"status" gorm:"column:status"  widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`

	Classify string `json:"classify" gorm:"column:classify"  widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0" validate:"required,oneof=民生 交通 医疗 就业 建议 其他"`

	// 框架标签：widget:"type:switch" - 开关组件；当前 switch 不支持 render_default，默认值走字段零值/数据库默认值
	// 开关组件支持 bool 类型，true 表示开启，false 表示关闭
	IsUrgent bool `json:"is_urgent" gorm:"column:is_urgent;default:false" widget:"name:是否紧急;type:switch"`

	// 框架标签：widget:"type:slider;min:0;max:100;unit:%" - 滑块/进度条组件
	// 输入模式：显示为滑块，用于编辑/新增表单
	// 输出模式：显示为进度条，自动显示百分比和状态颜色（>80% 67C23A, 50-80% E6A23C, <50% F56C6C）
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：min（最小值，必需）、max（最大值，必需）、unit（单位，可选）
	// 其他功能（提示、百分比、状态颜色等）自动处理，无需配置
	Progress int `json:"progress" gorm:"column:progress;default:0" widget:"name:完成进度;type:slider;min:0;max:100;unit:%" validate:"min=0,max=100"`

	// 框架标签：widget:"type:slider;min:0;max:10;step:0.1;unit:分" - 滑块/进度条组件（浮点数类型）
	// 支持浮点数和整数，例如：8.5、9.0、10 等
	// 输入模式：显示为滑块，支持小数步长（step:0.1）
	// 输出模式：显示为进度条，自动显示百分比和状态颜色
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：min（最小值，必需）、max（最大值，必需）、step（步长，可选，默认1）、unit（单位，可选）
	Score float64 `json:"score" gorm:"column:score;default:0" widget:"name:满意度评分;type:slider;min:0;max:10;step:0.1;unit:分" validate:"min=0,max=10"`

	// 框架标签：widget:"type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好" - 评分组件
	// 输入模式：显示为星级评分（1-5星），支持半星评分
	// 输出模式：显示评分值和星级
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：max（最大星级，默认5）、allow_half（是否允许半星，默认false）、texts（自定义文字数组，可选）
	// 注意：如果配置了 texts，会自动显示文字；如果没有配置 texts，则不显示文字
	Rating float64 `json:"rating" gorm:"column:rating;default:0" widget:"name:评价;type:rate;max:5;allow_half:true;texts:很差,差,一般,好,很好" validate:"min=0,max=5"`

	// 框架标签：validate:"required,min=11,max=20" - 必填字段，长度11-20字符
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`

	// 工单来源：工单的来源渠道（单选，选项较少）
	// 框架标签：widget:"type:radio;options:电话,邮件,在线,现场,其他;render_default:在线" - 单选框组件
	// 框架标签：validate:"required,oneof=电话 邮件 在线 现场 其他" - 必填字段，值必须是选项之一
	Source string `json:"source" gorm:"column:source" widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;render_default:在线" validate:"required,oneof=电话 邮件 在线 现场 其他"`

	// 预期处理时长：预计处理该工单需要的时间（单位：分钟）
	// 框架标签：widget:"type:integer;step:1;unit:分钟;render_default:60" - 数字输入组件，默认60分钟（1小时）
	ExpectedDuration int `json:"expected_duration" gorm:"column:expected_duration;default:60" widget:"name:预期处理时长;type:integer;step:1;unit:分钟;render_default:60" validate:"min=1,max=10080"` // 最大10080分钟（7天）

	// 处理耗时：实际处理该工单花费的时间（单位：分钟，自动计算，只读）
	// 框架标签：widget:"type:float;precision:2;unit:分钟" - 浮点数组件，保留两位小数
	// 当工单状态更新为"已关闭"时，自动根据创建时间计算处理耗时
	HandleDuration float64 `json:"handle_duration" gorm:"column:handle_duration;type:decimal(10,2);default:0" widget:"name:处理耗时;type:float;precision:2;unit:分钟;render_default:0.00"`

	//处理人：type:user 前端会自动在输入时候渲染成用户选择器，输出时候渲染成用户展示
	Handler string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user;render_default:Me()"`

	// 抄送人：type:users 前端会自动在输入时候渲染成多用户选择器，输出时候渲染成多个用户展示
	// 值使用逗号分隔的字符串存储（如 "user1,user2"），便于存储到数据库
	// 如需列表筛选，在 Request 中接收用户值后手写包含查询
	// 框架标签：render_default:Me(),MyLeader() - 前端新增时默认抄送当前用户和上级领导
	CcUsers string `json:"cc_users" gorm:"column:cc_users" widget:"name:抄送人;type:users;render_default:Me(),MyLeader()"`

	// 处理部门：处理这个工单的组织架构，默认是创建用户所在部门
	// 框架标签：widget:"type:department;render_default:MyDepartment()" - 组织架构选择器组件，默认当前用户所在部门
	// 如需列表筛选，在 Request 中声明部门字段并手写 Where
	HandleDepartment string `json:"handle_department" gorm:"column:handle_department" widget:"name:处理部门;type:department;render_default:MyDepartment()"`

	// 关联部门：工单关联的多个部门（用于跨部门协作）
	// 框架标签：widget:"type:departments;max_count:5" - 多组织架构选择器组件，最多选择5个部门
	// 可选参数：
	//   - render_default:MyDepartment() - 前端新增时默认当前用户所在部门，多个值用逗号分隔
	//   - max_count:5 - 最大选择数量，0表示不限制（例如：max_count:5 表示最多选择5个部门）
	// 值使用逗号分隔的字符串存储（如 "/dept1,/dept2"），便于存储到数据库
	// 如需列表筛选，在 Request 中接收部门值后手写包含查询
	RelatedDepartments string `json:"related_departments" gorm:"column:related_departments" widget:"name:关联部门;type:departments;render_default:MyDepartment();max_count:5"`

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	Remark string `json:"remark" gorm:"column:remark"  widget:"name:备注;type:text_area"`

	// 框架标签：widget:"name:详细内容;type:richtext;height:400" - 富文本编辑器组件
	// 输入模式：显示为富文本编辑器，支持格式化文本（粗体、斜体、标题、列表等）
	// 输出模式：显示 HTML 内容
	// 列表筛选：如需按内容筛选，在 Request 中声明筛选字段并手写查询条件
	// 参数说明：height（编辑器高度，单位px，默认300）
	Content string `json:"content" gorm:"column:content;type:text" widget:"name:详细内容;type:richtext;height:400"`

	// 框架标签：widget:"type:multiselect;options:紧急,重要,普通,低优先级;options_colors:FF5722,FF9800,4CAF50,9E9E9E" - 多选标签组件
	// options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，且数量必须和 options 一致。
	// 每个颜色对应一个选项，可以重复使用相同颜色
	// 注意：multiselect 字段使用 string 类型而非 []string，因为 []string 无法直接写入数据库
	// 前端会通过逗号分隔选项来传递多选的值，例如："紧急,重要" 表示选择了"紧急"和"重要"两个选项
	// 如需列表筛选，在 Request 中接收标签值后手写包含查询
	Tags string `json:"tags" gorm:"column:tags" widget:"name:标签;type:multiselect;options:紧急,重要,普通,低优先级;options_colors:FF5722,FF9800,4CAF50,9E9E9E"`

	// 框架标签：widget:"type:color;format:hex;render_default:#409EFF" - 颜色选择器组件
	// 输入模式：显示为颜色选择器（支持 hex、rgb、rgba 格式）
	// 输出模式：显示颜色块和颜色值
	// 列表筛选：如需按颜色筛选，在 Request 中声明筛选字段并手写查询条件
	// 参数说明：format（颜色格式：hex/rgb/rgba，默认hex）、render_default（默认颜色，可选）、show_alpha（是否显示透明度选择器，默认false）
	ThemeColor string `json:"theme_color" gorm:"column:theme_color" widget:"name:主题颜色;type:color;format:hex;render_default:#409EFF"`

	//请求参数里是文件上传组件，如果要存数据库必须是type:json类型
	Attachment string `json:"attachment" gorm:"column:attachment;type:text"  widget:"name:附件;type:files"`

	// 这个字段非必要，纯粹展示怎么获取当前用户的组织架构，正常来讲CreatedBy是非常必要的字段
	// 所属部门：工单提单的部门，默认是创建用户所在部门
	// 由后端赋值的字段可在 OnTableAddRow 中通过 ctx.GetRequestUserDept() 设置；前端是否展示由 hide.scenes 决定。
	// 如需列表筛选，在 Request 中声明部门字段并手写 Where
	Department string `json:"department" gorm:"column:department" widget:"name:提单部门;type:department" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	// 创建用户：用户组件
	CreatedBy string `json:"created_by" gorm:"column:created_by" widget:"name:创建用户;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单；由后端通过 ctx.GetRequestUser() 赋值。

	// 截止时间：工单要求完成的时间
	Deadline types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	// 剩余时间：仅展示用，不落库。在查询到列表后、返回 Table 前根据当前时间与截止时间计算
	RemainingTime string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (t *Ticket) TableName() string {
	return "ticket"
}

var TicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "工单管理",
		Tags:     []string{"工单管理系统"},
		Desc:     `一个简单的工单管理系统 ........`,
		Request:  &TicketListReq{},
		CreateTables: []interface{}{
			&Ticket{},
		},
	},
	AutoCrudTable: &Ticket{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Ticket
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		row.Department = ctx.GetRequestUserDept()
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create crm_ticket err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields Ticket
		if err := req.BindChangedFields(&updateFields); err != nil { //这里不会验证validate，为啥？因为前端只传递了变更的字段，所以无需验证，所以updateFields只有 更新的字段才会有值，没更新的字段是零值
			return nil, err
		}
		//注意：updateFields主要是方便安全的操作变更的字段，如果更新数据我们还是配合用req.ChangedFields()来，这样例如某些字符串想更新成空，或者int想更新成0是可以实现的，
		//用updateFields 的话，gorm是无法更新零值的

		updates := req.ChangedFields()

		// 判断状态是否更新为"已关闭"，如果是则自动计算处理耗时
		if req.IsFieldUpdated("status") && updateFields.Status == "已关闭" {
			// 获取当前工单的创建时间
			var currentTicket Ticket
			if err := db.First(&currentTicket, req.GetId()).Error; err != nil {
				return nil, err
			}

			// 计算处理耗时（分钟）
			durationMinutes := time.Since(currentTicket.CreatedAt.Time()).Minutes()

			// 保留两位小数（四舍五入）
			updates["handle_duration"] = float64(int(durationMinutes*100+0.5)) / 100.0
		} //-> map[string]interface{} 这里值包含此次变更的字段，例如用户把status 变更成：“已完成”，那么这里的map就是只有一个 status:已完成
		err := db.Model(&Ticket{}).Where("id = ?", req.GetId()).Updates(updates).Error //这是标准的更新方式
		if err != nil {
			return nil, err
		}
		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Ticket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
			"deleted_at": time.Now(),
		}).Error
		if err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type TicketListReq struct {
	Title    string `json:"title" form:"title" widget:"name:工单标题;type:input"`
	Priority string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
	Status   string `json:"status" form:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C"`
	Classify string `json:"classify" form:"classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0"`

	query.PageSortReq `widget:"-"`
}

func TicketList(ctx *app.Context, resp response.Response) error {
	var req TicketListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "TicketSearch ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Ticket{})
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Priority != "" {
		queryDB = queryDB.Where("priority = ?", req.Priority)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Classify != "" {
		queryDB = queryDB.Where("classify = ?", req.Classify)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "TicketCount err: %v", err)
		return err
	}
	var lists []*Ticket
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "TicketSearch err: %v", err)
		return err
	}
	// 根据截止时间填充剩余时间（仅展示）
	now := time.Now()
	for _, item := range lists {
		if item.Deadline.IsZero() {
			continue
		}
		deadline := item.Deadline.Time()
		if !now.Before(deadline) {
			item.RemainingTime = "已过期"
			continue
		}
		diff := deadline.Sub(now)
		days := int(diff.Hours()) / 24
		hours := int(diff.Hours()) % 24
		mins := int(diff.Minutes()) % 60
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
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	// 💡packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	packageContext.GET("ticket_list.table", TicketList, TicketTemplate)

}
```
