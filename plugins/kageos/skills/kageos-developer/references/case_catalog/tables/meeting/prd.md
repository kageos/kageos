# 案例：智能会议室预约（Table + Form + 定时任务）

## 一、项目概要

- **类型**：两张核心 Table（会议室、预约记录）+ 两个 POST Form（空闲查询、会前提醒），以多表资源占用为主，Form 作为业务动作补充。
- **GET Table**：`meeting_room_list.table`（会议室管理）、`meeting_room_booking_list.table`（会议室预约管理）。
- **POST Form**：`meeting_room_query_available.form`（查询空闲会议室 + 一键预约）、`meeting_room_notify_soon.form`（会前提醒，内置默认调度，发布后开箱即用）。
- **关系**：预约表关联会议室；预约时选会议室用 **OnSelectFuzzy**，只返回状态为「可用」的会议室；预约列表带「会议室详情」**link** 跳转到会议室列表。
- **状态**：预约状态不落库，按开始/结束时间实时计算（待开始/进行中/已结束）；列表筛选「预约状态」时在 Handler 中拼时间条件。
- **提醒**：会前提醒通过 `ctx.SendNotification` 发给预约人和参会人；发送前先条件更新 `reminder_sent` 做幂等 claim，失败时释放标记；`FormTemplate.Schedules` 默认每 2 分钟扫描未来 5 分钟内即将开始的会议。
- **适合参考**：中小企业资源预约、台账管理、跨表筛选、空闲资源查询、一键跳转、定时巡检、消息提醒等轻量业务闭环。

---

## 二、结构化 PRD JSON

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 保留实现参考、SDK 写法、注意事项与完整 Go 源码。

`product_manager` 阶段只需要产出轻量 PRD，不要把代码、数据库列名或完整 SDK 标签塞进 PRD；选项、默认值、数据来源和计算规则写在自然语言 `desc` 和 `rules` 中。`app_developer` 阶段再按本 Markdown 的完整源码落地。

## 三、业务校验与逻辑要点

- **会议室管理**：会议室可新增、编辑、删除；删除前需检查是否存在未结束预约，存在则禁止删除，建议改为「维护中」或「停用」。
- **预约新增**：预约人由后端 `ctx.GetRequestUser()` 写入；参会人为空时默认预约人；参会人数根据参会人字符串自动计算。
- **预约校验**：结束时间必须晚于开始时间；开始时间不能是过去时间（保留 15 分钟操作宽限）；会议室必须存在且状态为「可用」；参会人数不能超过会议室容量；同一会议室时间段不能重叠。
- **预约更新**：更新会议室、开始时间、结束时间或参会人时，必须用合成后的记录重新校验容量和冲突，并在冲突检测中排除当前记录 ID。
- **提醒重置**：更新会议室、时间、预约人、参会人或会议主题后，需要重置 `reminder_sent/reminded_at`，允许定时任务按新信息重新提醒。
- **预约删除**：进行中的会议禁止删除，可提示「不能删除进行中的会议，请等待会议结束后再删除」。
- **预约列表筛「会议室名称」**：先用 `MeetingRoom.name LIKE ?` 查出 room_id，再用 `room_id IN ?` 过滤预约表；查不到时用 `1 = 0` 返回空结果。
- **预约列表筛「预约状态」**：用当前时间与 start_time、end_time 比较：待开始 `start_time > now`，进行中 `start_time <= now AND end_time > now`，已结束 `end_time <= now`。
- **查询空闲会议室**：按时间段查出已占用 room_id，再返回状态为「可用」且容量满足条件的会议室；每行生成跳转到预约新增页的一键预约链接并预填会议室和时间。
- **会前提醒**：`meeting_room_notify_soon.form` 在 `FormTemplate.Schedules` 中声明默认调度；发布/安装应用后由平台幂等同步到 timer-scheduler。默认每 2 分钟扫描未来 5 分钟内将开始且未提醒的会议；发送前条件更新标记为已提醒，发送失败或没有接收人时释放标记，避免重复提醒和漏提醒。

---

## 四、文件与路由

| 文件 | 说明 | 注册路由 |
| --- | --- | --- |
| init_.go | 包上下文 | RouterGroup `/meeting` |
| meeting_room.go | 会议室管理、会议室选择 OnSelectFuzzy、删除保护 | GET meeting_room_list.table |
| meeting_room_booking.go | 预约管理、时间冲突校验、状态计算、会前提醒和默认调度 | GET meeting_room_booking_list.table；POST meeting_room_notify_soon.form |
| meeting_room_query_available.go | 查询空闲会议室、一键预约链接 | POST meeting_room_query_available.form |

**OnSelectFuzzyMap**：预约表 Template 中注册 `"room_id": onSelectFuzzyMeetingRoom`，会议室下拉只显示状态为「可用」的会议室；用户按名称、类型、位置搜索，前端实际提交会议室 ID。

---

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/tables/meeting`）即获得 PRD 与完整代码，无需再调用 read_file。

---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### init_.go

```go
package meeting

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/meeting",
	Name:        "智能会议室管理",
	Desc:        "",
}
```

### meeting_room.go

```go
package meeting

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

type MeetingRoom struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:会议室ID;type:ID" hide:"create,update"`                                                     // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Name      string `json:"name" gorm:"column:name;comment:会议室名称" widget:"name:会议室名称;type:input" validate:"required,min=2,max=50"`
	Type      string `json:"type" gorm:"column:type;comment:会议室类型" widget:"name:会议室类型;type:select;options:小型,中型,大型,会议室,培训室,多功能厅;options_colors:909399,409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	Capacity  int    `json:"capacity" gorm:"column:capacity;comment:容纳人数" widget:"name:容纳人数;type:integer;min:1;max:1000" validate:"required,min=1,max=1000"`
	Equipment string `json:"equipment" gorm:"column:equipment;type:text;comment:设备配置" widget:"name:设备配置;type:text_area"`
	Location  string `json:"location" gorm:"column:location;comment:位置信息" widget:"name:位置信息;type:input" validate:"required,min=2,max=100"`
	Status    string `json:"status" gorm:"column:status;comment:状态;default:可用" widget:"name:状态;type:select;options:可用,维护中,停用;options_colors:67C23A,E6A23C,F56C6C;render_default:可用" validate:"required,oneof=可用 维护中 停用"`
}

func (MeetingRoom) TableName() string {
	return "crm_meeting_room"
}

type MeetingRoomListReq struct {
	ID     int    `json:"id" form:"id" widget:"name:会议室ID;type:integer"`
	Name   string `json:"name" form:"name" widget:"name:会议室名称;type:input"`
	Type   string `json:"type" form:"type" widget:"name:会议室类型;type:select;options:小型,中型,大型,会议室,培训室,多功能厅;options_colors:909399,409EFF,67C23A,E6A23C,F56C6C,9C27B0"`
	Status string `json:"status" form:"status" widget:"name:状态;type:select;options:可用,维护中,停用;options_colors:67C23A,E6A23C,F56C6C"`

	query.PageSortReq `widget:"-"`
}

func MeetingRoomList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req MeetingRoomListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&MeetingRoom{})
	if req.ID > 0 {
		queryDB = queryDB.Where("id = ?", req.ID)
	}
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Type != "" {
		queryDB = queryDB.Where("type = ?", req.Type)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	var rooms []MeetingRoom
	if order := (&req.PageSortReq).GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Model(&MeetingRoom{}).Count(&total).Error; err != nil {
		return err
	}
	if err := queryDB.Offset((&req.PageSortReq).GetOffset()).Limit((&req.PageSortReq).GetLimit()).Find(&rooms).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      rooms,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var MeetingRoomListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name: "会议室管理",
		Desc: `## 功能说明
会议室管理 用于会议室信息的增删改查管理，包括会议室名称、类型、容纳人数、设备配置、位置信息、状态等。

## 适用场景
- 需要集中维护、查询和跟踪会议室管理相关记录。
- 适合作为后台管理、数据台账或业务运营入口使用。
- 可配合平台的权限、操作记录和定时任务等通用能力形成完整工作流。

## 使用说明
- 先使用筛选条件定位目标数据，再查看列表、详情或统计信息。
- 根据页面开放的能力进行新增、编辑、删除或批量处理。
- 执行前确认必填字段、枚举值、文件字段和关联数据是否填写完整。`,
		Tags:         []string{"会议室系统", "会议室管理"},
		Request:      &MeetingRoomListReq{},
		Response:     query.PaginatedTable[[]MeetingRoom]{},
		CreateTables: []interface{}{&MeetingRoom{}},
	},
	AutoCrudTable: &MeetingRoom{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row MeetingRoom
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create meeting_room err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields MeetingRoom
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()
		if err := db.Model(&MeetingRoom{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "Update meeting_room err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		var activeBookingCount int64
		if err := db.Model(&MeetingRoomBooking{}).
			Where("room_id in ? AND end_time > ?", req.GetIds(), time.Now()).
			Count(&activeBookingCount).Error; err != nil {
			return nil, fmt.Errorf("检查会议室预约失败: %w", err)
		}
		if activeBookingCount > 0 {
			return nil, fmt.Errorf("会议室存在未结束的预约，请先处理预约或将会议室状态改为停用")
		}
		if err := db.Model(&MeetingRoom{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": ctx.GetRequestUser(),
		}).Error; err != nil {
			logger.Errorf(ctx, "Delete meeting_room err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func onSelectFuzzyMeetingRoom(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var rooms []MeetingRoom
	db = db.Model(&MeetingRoom{}).Where("status = ?", "可用")
	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ? OR type LIKE ? OR location LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").Limit(20)
	}

	db.Find(&rooms)

	items := make([]*callback.SelectFuzzyItem, 0, len(rooms))
	for _, r := range rooms {
		items = append(items, &callback.SelectFuzzyItem{
			Value: r.ID,
			Label: fmt.Sprintf("%s - %s (容纳%d人, %s)", r.Name, r.Type, r.Capacity, r.Location),
			DisplayInfo: map[string]interface{}{
				"会议室名称": r.Name,
				"类型":    r.Type,
				"容纳人数":  r.Capacity,
				"位置":    r.Location,
				"设备配置":  r.Equipment,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"选中会议室": statistics.Value("会议室名称"),
			"会议室类型": statistics.Value("类型"),
			"容纳人数":  statistics.Value("容纳人数"),
			"位置":    statistics.Value("位置"),
			"设备配置":  statistics.Value("设备配置"),
		},
	}, nil
}

func init() {
	packageContext.GET("meeting_room_list.table", MeetingRoomList, MeetingRoomListTemplate)
}
```

### meeting_room_booking.go

```go
package meeting

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

type MeetingRoomBooking struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:预约ID;type:ID" hide:"create,update"`                                                      // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	RoomID   int          `json:"room_id" gorm:"column:room_id;comment:会议室ID;index" widget:"name:会议室;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Room     *MeetingRoom `json:"-" widget:"-" gorm:"foreignKey:RoomID;references:ID"`
	RoomName string       `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" hide:"create,update"`               // 前端仅在列表展示，不进入新增/编辑表单。
	RoomLink string       `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	Booker        string     `json:"booker" gorm:"column:booker;comment:预约人" widget:"name:预约人;type:user;desc:请使用实际用户名，禁止随意填写" hide:"create,update"`
	Attendees     string     `json:"attendees" gorm:"column:attendees;type:text;comment:参会人（逗号分隔）" widget:"name:参会人;type:users;desc:请使用实际用户名，多个用户用逗号分隔，禁止随意填写;render_default:Me()"`
	Subject       string     `json:"subject" gorm:"column:subject;comment:会议主题" widget:"name:会议主题;type:input" validate:"required,min=2,max=200"`
	Description   string     `json:"description" gorm:"column:description;type:text;comment:会议描述" widget:"name:会议描述;type:richtext;height:360"`
	Attachment    string     `json:"attachment" gorm:"column:attachment;type:text;comment:会议附件" widget:"name:会议附件;type:files;accept:.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.png,.jpg,.jpeg,.gif;max_size:100MB;max_count:10"`
	StartTime     types.Time `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间;index" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP" validate:"required"`
	EndTime       types.Time `json:"end_time" gorm:"column:end_time;type:datetime;comment:结束时间;index" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	AttendeeCount int        `json:"attendee_count" gorm:"column:attendee_count;comment:参会人数" widget:"name:参会人数;type:integer" hide:"create,update"`
	Status        string     `json:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	ReminderSent  bool       `json:"reminder_sent" gorm:"column:reminder_sent;default:false;comment:是否已发送会前提醒" widget:"name:是否已提醒;type:switch" hide:"create,update"`
	RemindedAt    types.Time `json:"reminded_at" gorm:"column:reminded_at;type:datetime;comment:提醒发送时间" widget:"name:提醒时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	Remark        string     `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
}

func (MeetingRoomBooking) TableName() string {
	return "crm_meeting_room_booking"
}

type MeetingRoomBookingListReq struct {
	RoomName string `json:"room_name" form:"room_name" gorm:"-" widget:"name:会议室名称;type:input"`
	Status   string `json:"status" form:"status" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`

	query.PageSortReq `widget:"-"`
}

func MeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req MeetingRoomBookingListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&MeetingRoomBooking{})
	if req.RoomName != "" {
		var roomIDs []int
		if err := db.Model(&MeetingRoom{}).Where("name LIKE ?", "%"+req.RoomName+"%").Pluck("id", &roomIDs).Error; err == nil && len(roomIDs) > 0 {
			queryDB = queryDB.Where("room_id IN ?", roomIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}

	if req.Status != "" {
		now := time.Now()
		switch req.Status {
		case "待开始":
			queryDB = queryDB.Where("crm_meeting_room_booking.start_time > ?", now)
		case "进行中":
			queryDB = queryDB.Where("crm_meeting_room_booking.start_time <= ? AND crm_meeting_room_booking.end_time > ?", now, now)
		case "已结束":
			queryDB = queryDB.Where("crm_meeting_room_booking.end_time <= ?", now)
		}
	}

	queryDB = queryDB.Preload("Room")

	var bookings []MeetingRoomBooking
	if order := (&req.PageSortReq).GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Model(&MeetingRoomBooking{}).Count(&total).Error; err != nil {
		return err
	}
	if err := queryDB.Offset((&req.PageSortReq).GetOffset()).Limit((&req.PageSortReq).GetLimit()).Find(&bookings).Error; err != nil {
		return err
	}

	for i := range bookings {
		if bookings[i].Room != nil {
			bookings[i].RoomName = bookings[i].Room.Name
		}
		bookings[i].Status = calculateBookingStatus(bookings[i].StartTime, bookings[i].EndTime)
		params := MeetingRoom{ID: bookings[i].RoomID}
		bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list.table", params, "查看会议室详情")
	}

	return resp.Table(response.TableResult{
		Items:      bookings,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var MeetingRoomBookingListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name: "会议室预约管理",
		Desc: `## 功能说明
会议室预约管理 用于会议室预约的增删改查管理，包括会议室选择、预约人、会议主题、时间安排、参会人数、预约状态等。

## 适用场景
- 需要集中维护、查询和跟踪会议室预约管理相关记录。
- 适合作为后台管理、数据台账或业务运营入口使用。
- 可配合平台的权限、操作记录和定时任务等通用能力形成完整工作流。

## 使用说明
- 先使用筛选条件定位目标数据，再查看列表、详情或统计信息。
- 根据页面开放的能力进行新增、编辑、删除或批量处理。
- 执行前确认必填字段、枚举值、文件字段和关联数据是否填写完整。`,
		Tags:         []string{"会议室系统", "预约管理"},
		Request:      &MeetingRoomBookingListReq{},
		Response:     query.PaginatedTable[[]MeetingRoomBooking]{},
		CreateTables: []interface{}{&MeetingRoomBooking{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"room_id": onSelectFuzzyMeetingRoom,
		},
	},
	AutoCrudTable: &MeetingRoomBooking{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row MeetingRoomBooking
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}

		// 自动设置预约人为当前登录用户（后端获取，不允许前端手动填写）
		currentUser := ctx.GetRequestUser()
		if currentUser != "" {
			row.Booker = currentUser
		}

		// 自动填充：参会人为空时，默认填充为预约人
		if strings.TrimSpace(row.Attendees) == "" {
			row.Attendees = row.Booker
		}

		// 自动计算参会人数
		row.AttendeeCount = calculateAttendeeCount(row.Attendees)

		if err := validateBookingTime(db, &row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create meeting_room_booking err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var currentBooking MeetingRoomBooking
		if err := db.Where("id = ?", req.GetId()).First(&currentBooking).Error; err != nil {
			return nil, fmt.Errorf("预约记录不存在")
		}

		var updateFields MeetingRoomBooking
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}
		updates := req.ChangedFields()

		if req.IsFieldUpdated("attendees") {
			updates["attendee_count"] = calculateAttendeeCount(updateFields.Attendees)
		}

		if req.IsFieldUpdated("start_time") || req.IsFieldUpdated("end_time") || req.IsFieldUpdated("room_id") || req.IsFieldUpdated("attendee_count") || req.IsFieldUpdated("attendees") {
			tempBooking := MeetingRoomBooking{
				RoomID:        currentBooking.RoomID,
				StartTime:     currentBooking.StartTime,
				EndTime:       currentBooking.EndTime,
				AttendeeCount: currentBooking.AttendeeCount,
			}
			if req.IsFieldUpdated("start_time") {
				tempBooking.StartTime = updateFields.StartTime
			}
			if req.IsFieldUpdated("end_time") {
				tempBooking.EndTime = updateFields.EndTime
			}
			if req.IsFieldUpdated("room_id") {
				tempBooking.RoomID = updateFields.RoomID
			}
			if req.IsFieldUpdated("attendee_count") {
				tempBooking.AttendeeCount = updateFields.AttendeeCount
			}
			if req.IsFieldUpdated("attendees") {
				tempBooking.AttendeeCount = calculateAttendeeCount(updateFields.Attendees)
			}
			if err := validateBookingTimeExclude(db, &tempBooking, req.GetId()); err != nil {
				return nil, err
			}
		}

		if req.IsFieldUpdated("start_time") || req.IsFieldUpdated("end_time") || req.IsFieldUpdated("room_id") ||
			req.IsFieldUpdated("booker") || req.IsFieldUpdated("attendees") || req.IsFieldUpdated("subject") {
			updates["reminder_sent"] = false
			updates["reminded_at"] = types.Time{}
		}

		if err := db.Model(&MeetingRoomBooking{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "Update meeting_room_booking err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		var bookings []MeetingRoomBooking
		if err := db.Where("id in ?", req.GetIds()).Find(&bookings).Error; err != nil {
			return nil, err
		}
		for _, booking := range bookings {
			if calculateBookingStatus(booking.StartTime, booking.EndTime) == "进行中" {
				return nil, fmt.Errorf("不能删除进行中的会议，请等待会议结束后再删除")
			}
		}

		if err := db.Model(&MeetingRoomBooking{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": ctx.GetRequestUser(),
		}).Error; err != nil {
			logger.Errorf(ctx, "Delete meeting_room_booking err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type MeetingRoomNotifySoonReq struct {
	LeadMinutes int `json:"lead_minutes" widget:"name:提前提醒分钟数;type:integer;min:1;render_default:5"`
}

type MeetingRoomNotifySoonResp struct {
	CheckedCount  int `json:"checked_count" widget:"name:扫描会议数;type:integer"`
	NotifiedCount int `json:"notified_count" widget:"name:已通知会议数;type:integer"`
}

func MeetingRoomNotifySoon(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	req := MeetingRoomNotifySoonReq{LeadMinutes: 5}
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	if req.LeadMinutes <= 0 {
		req.LeadMinutes = 5
	}

	now := time.Now()
	windowEnd := now.Add(time.Duration(req.LeadMinutes) * time.Minute)

	var bookings []MeetingRoomBooking
	if err := db.Model(&MeetingRoomBooking{}).
		Preload("Room").
		Where("start_time > ? AND start_time <= ? AND reminder_sent = ? AND deleted_at IS NULL", now, windowEnd, false).
		Find(&bookings).Error; err != nil {
		return err
	}

	notifiedCount := 0
	for _, booking := range bookings {
		claimTime := time.Now()
		claim := db.Model(&MeetingRoomBooking{}).
			Where("id = ? AND reminder_sent = ? AND deleted_at IS NULL", booking.ID, false).
			Updates(map[string]interface{}{
				"reminder_sent": true,
				"reminded_at":   claimTime,
			})
		if claim.Error != nil {
			logger.Errorf(ctx, "Claim meeting reminder failed, booking_id=%d, err=%v", booking.ID, claim.Error)
			continue
		}
		if claim.RowsAffected == 0 {
			continue
		}

		toUsers := joinUsers(booking.Booker, booking.Attendees)
		if toUsers == "" {
			if err := db.Model(&MeetingRoomBooking{}).Where("id = ?", booking.ID).Updates(map[string]interface{}{
				"reminder_sent": false,
				"reminded_at":   types.Time{},
			}).Error; err != nil {
				logger.Errorf(ctx, "Release meeting reminder claim failed, booking_id=%d, err=%v", booking.ID, err)
			}
			continue
		}

		roomName := "未知会议室"
		if booking.Room != nil && booking.Room.Name != "" {
			roomName = booking.Room.Name
		}

		startAt := booking.StartTime.Time().Format("2006-01-02 15:04")
		content := fmt.Sprintf("您预约/参与的会议《%s》将在 %s 开始，会议室：%s，请提前准备。", booking.Subject, startAt, roomName)
		// 默认 markdown，ToUsers 来自 booking 的 user/users 字段（逗号分隔格式）
		err := ctx.SendNotification(&app.SendNotificationOpts{
			ToUsers: toUsers,
			Title:   "会议即将开始提醒",
			Message: content,
		})
		if err != nil {
			logger.Errorf(ctx, "Send meeting reminder failed, booking_id=%d, err=%v", booking.ID, err)
			if err := db.Model(&MeetingRoomBooking{}).Where("id = ?", booking.ID).Updates(map[string]interface{}{
				"reminder_sent": false,
				"reminded_at":   types.Time{},
			}).Error; err != nil {
				logger.Errorf(ctx, "Release meeting reminder claim failed, booking_id=%d, err=%v", booking.ID, err)
			}
			continue
		}

		notifiedCount++
	}

	return resp.Form(&MeetingRoomNotifySoonResp{
		CheckedCount:  len(bookings),
		NotifiedCount: notifiedCount,
	}).Build()
}

func joinUsers(users ...string) string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, userSet := range users {
		parts := strings.Split(userSet, ",")
		for _, part := range parts {
			user := strings.TrimSpace(part)
			if user == "" {
				continue
			}
			if _, ok := seen[user]; ok {
				continue
			}
			seen[user] = struct{}{}
			result = append(result, user)
		}
	}
	return strings.Join(result, ",")
}

func validateBookingTime(db *gorm.DB, booking *MeetingRoomBooking) error {
	if !booking.EndTime.Time().After(booking.StartTime.Time()) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}

	now := time.Now()
	// 宽限期15分钟：容忍用户从查询到提交的操作延迟
	if booking.StartTime.Time().Before(now.Add(-15 * time.Minute)) {
		return fmt.Errorf("开始时间不能是过去时间")
	}

	var room MeetingRoom
	if err := db.Where("id = ?", booking.RoomID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("会议室不存在")
		}
		return fmt.Errorf("查询会议室失败: %v", err)
	}
	if room.Status != "可用" {
		return fmt.Errorf("会议室 %s 当前状态为 %s，无法预约", room.Name, room.Status)
	}
	if booking.AttendeeCount > room.Capacity {
		return fmt.Errorf("参会人数 %d 超过会议室容量 %d", booking.AttendeeCount, room.Capacity)
	}

	var conflictBooking MeetingRoomBooking
	err := db.Where("room_id = ? AND deleted_at IS NULL", booking.RoomID).
		Where("((start_time <= ? AND end_time > ?) OR (start_time < ? AND end_time >= ?) OR (start_time >= ? AND end_time <= ?))",
			booking.StartTime, booking.StartTime,
			booking.EndTime, booking.EndTime,
			booking.StartTime, booking.EndTime).
		First(&conflictBooking).Error
	if err == nil {
		return fmt.Errorf("该时间段已被预约，请选择其他时间或会议室")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("检查时间冲突失败: %v", err)
	}
	return nil
}

func validateBookingTimeExclude(db *gorm.DB, booking *MeetingRoomBooking, excludeID int) error {
	if !booking.EndTime.Time().After(booking.StartTime.Time()) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}

	now := time.Now()
	// 宽限期15分钟：容忍用户从查询到提交的操作延迟
	if booking.StartTime.Time().Before(now.Add(-15 * time.Minute)) {
		return fmt.Errorf("开始时间不能是过去时间")
	}

	var room MeetingRoom
	if err := db.Where("id = ?", booking.RoomID).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("会议室不存在")
		}
		return fmt.Errorf("查询会议室失败: %v", err)
	}
	if room.Status != "可用" {
		return fmt.Errorf("会议室 %s 当前状态为 %s，无法预约", room.Name, room.Status)
	}
	if booking.AttendeeCount > room.Capacity {
		return fmt.Errorf("参会人数 %d 超过会议室容量 %d", booking.AttendeeCount, room.Capacity)
	}

	var conflictBooking MeetingRoomBooking
	err := db.Where("room_id = ? AND id != ? AND deleted_at IS NULL", booking.RoomID, excludeID).
		Where("((start_time <= ? AND end_time > ?) OR (start_time < ? AND end_time >= ?) OR (start_time >= ? AND end_time <= ?))",
			booking.StartTime, booking.StartTime,
			booking.EndTime, booking.EndTime,
			booking.StartTime, booking.EndTime).
		First(&conflictBooking).Error
	if err == nil {
		return fmt.Errorf("该时间段已被预约，请选择其他时间或会议室")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("检查时间冲突失败: %v", err)
	}
	return nil
}

func calculateBookingStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "待开始"
	}
	if now.Before(endTime.Time()) {
		return "进行中"
	}
	return "已结束"
}

// calculateAttendeeCount 根据参会人数字符串计算参会人数
func calculateAttendeeCount(attendees string) int {
	if strings.TrimSpace(attendees) == "" {
		return 0
	}
	parts := strings.Split(attendees, ",")
	count := 0
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func init() {
	packageContext.GET("meeting_room_booking_list.table", MeetingRoomBookingList, MeetingRoomBookingListTemplate)
	packageContext.POST("meeting_room_notify_soon.form", MeetingRoomNotifySoon, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name: "会议即将开始提醒（定时任务）",
			Desc: `## 功能说明
会议即将开始提醒（定时任务） 用于巡检未来N分钟内将开始的会议，并给预约人和参会人发送提醒消息。应用发布后会内置默认调度，开箱即用。

## 适用场景
- 适合会议室预约场景的自动会前提醒。
- 适合平台发布后自动创建默认定时任务，减少人工配置。
- 也可作为工作台智能体或管理员手动触发的巡检工具。

## 使用说明
- 默认调度每 2 分钟执行一次，扫描未来 5 分钟内未提醒的会议。
- 如需手动触发，可填写提前提醒分钟数并提交表单。
- 发送成功后会标记已提醒，发送失败会释放标记，避免重复提醒或漏提醒。`,
			Tags:     []string{"会议室系统", "消息提醒", "定时任务"},
			Request:  &MeetingRoomNotifySoonReq{},
			Response: &MeetingRoomNotifySoonResp{},
		},
		Schedules: []app.FormSchedule{
			{
				Code:        "meeting_reminder_soon",
				Title:       "会议即将开始提醒",
				Description: "每 2 分钟扫描未来 5 分钟内即将开始且未提醒的会议，并通知预约人和参会人。",
				CronExpr:    "*/2 * * * *",
				Body:        MeetingRoomNotifySoonReq{LeadMinutes: 5},
			},
		},
	})
}
```

### meeting_room_query_available.go

```go
package meeting

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
)

// QueryAvailableReq 查询空闲会议室请求
type QueryAvailableReq struct {
	StartTime   string `json:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP" validate:"required"`
	EndTime     string `json:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	MinCapacity int    `json:"min_capacity" widget:"name:最少容纳人数;type:integer;min:1;placeholder:不填则返回全部"`
}

// AvailableRoom 空闲会议室
type AvailableRoom struct {
	RoomID    int    `json:"room_id" widget:"name:会议室ID;type:ID"`
	Name      string `json:"name" widget:"name:会议室名称;type:text"`
	Type      string `json:"type" widget:"name:类型;type:text"`
	Capacity  int    `json:"capacity" widget:"name:容纳人数;type:integer"`
	Equipment string `json:"equipment" widget:"name:设备配置;type:text"`
	Location  string `json:"location" widget:"name:位置;type:text"`
	BookLink  string `json:"book_link" widget:"name:一键预约;type:link;link_type:primary"`
}

// QueryAvailableResp 查询空闲会议室响应
type QueryAvailableResp struct {
	TotalCount int             `json:"total_count" widget:"name:空闲会议室数量;type:integer"`
	Rooms      []AvailableRoom `json:"rooms" widget:"name:空闲会议室;type:table"`
}

// QueryAvailableForm 查询空闲会议室表单
func QueryAvailableForm(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req QueryAvailableReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 解析时间
	startTime, err := types.ParseTime(req.StartTime)
	if err != nil {
		return fmt.Errorf("开始时间格式错误")
	}
	endTime, err := types.ParseTime(req.EndTime)
	if err != nil {
		return fmt.Errorf("结束时间格式错误")
	}

	// 校验时间逻辑
	if !endTime.Time().After(startTime.Time()) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}

	// 查询已被预约的会议室ID（时间段有重叠的）
	var bookedRoomIDs []int
	err = db.Model(&MeetingRoomBooking{}).
		Where("deleted_at IS NULL").
		Where("((start_time <= ? AND end_time > ?) OR (start_time < ? AND end_time >= ?) OR (start_time >= ? AND end_time <= ?))",
			startTime.Time(), startTime.Time(),
			endTime.Time(), endTime.Time(),
			startTime.Time(), endTime.Time()).
		Pluck("room_id", &bookedRoomIDs).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[QueryAvailableForm] 查询已预约会议室失败, req: %+v, err: %w", req, err)
	}

	// 查询可用状态的会议室
	queryDB := db.Model(&MeetingRoom{}).Where("status = ?", "可用")

	// 排除已被预约的会议室
	if len(bookedRoomIDs) > 0 {
		queryDB = queryDB.Where("id NOT IN ?", bookedRoomIDs)
	} else {
		// 没有已预约会议室时，排除一个不存在的ID避免语法问题
		queryDB = queryDB.Where("1 = 1")
	}

	// 按容纳人数筛选（如果填写了）
	if req.MinCapacity > 0 {
		queryDB = queryDB.Where("capacity >= ?", req.MinCapacity)
	}

	var rooms []MeetingRoom
	if err := queryDB.Order("name ASC").Find(&rooms).Error; err != nil {
		return fmt.Errorf("[系统错误]-[QueryAvailableForm] 查询会议室失败, req: %+v, err: %w", req, err)
	}

	// 构建响应：每个会议室生成一键预约链接
	resultRooms := make([]AvailableRoom, 0, len(rooms))
	for _, room := range rooms {
		// 构建预约链接参数：跳转到预约表新增 Tab，预填 room_id、start_time、end_time
		params := MeetingRoomBooking{
			RoomID:    room.ID,
			StartTime: startTime,
			EndTime:   endTime,
		}
		bookLink, err := ctx.BuildFunctionUrlWithText("meeting_room_booking_list.table?_tab=OnTableAddRow", params, "一键预约")
		if err != nil {
			bookLink = ""
		}

		resultRooms = append(resultRooms, AvailableRoom{
			RoomID:    room.ID,
			Name:      room.Name,
			Type:      room.Type,
			Capacity:  room.Capacity,
			Equipment: room.Equipment,
			Location:  room.Location,
			BookLink:  bookLink,
		})
	}

	return resp.Form(&QueryAvailableResp{
		TotalCount: len(resultRooms),
		Rooms:      resultRooms,
	}).Build()
}

func init() {
	packageContext.POST("meeting_room_query_available.form", QueryAvailableForm, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name: "查询空闲会议室",
			Desc: `## 功能说明
查询空闲会议室 用于在指定时间段内查找可用的空闲会议室，并支持一键预约。

## 适用场景
- 需要快速查找某个时间段内哪些会议室可用。
- 支持按容纳人数筛选，返回满足条件的会议室。
- 点击「一键预约」可直接跳转到预约表单并预填会议室和时间。

## 使用说明
- 填写开始时间和结束时间（必填）。
- 可选填写最少容纳人数，不填则返回全部空闲会议室。
- 点击空闲会议室行的「一键预约」按钮，跳转到预约表单预填会议室和时间。`,
			Tags:     []string{"会议室系统", "预约查询"},
			Request:  &QueryAvailableReq{},
			Response: &QueryAvailableResp{},
		},
	})
}
```
