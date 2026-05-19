# 案例：会议室预约（多 Table）

## 一、项目概要

- **类型**：主从两表，两个 GET Table，无独立 POST Form。
- **路由**：`meeting_room_list.table`（会议室管理）、`meeting_room_booking_list.table`（预约管理）。
- **关系**：预约表关联会议室；预约时选会议室用 **OnSelectFuzzy**（只筛「可用」会议室）；预约列表可带「会议室详情」**link** 跳转到会议室列表。
- **状态**：预约状态不落库，按开始/结束时间**实时计算**（待开始/进行中/已结束）；列表请求可筛「预约状态」时在 Handler 里用时间条件过滤。
- **列表筛**：预约列表请求带「会议室名称」「预约状态」等**外表/计算字段**，需在 Handler 里手动拼条件（如按会议室 name like 查 room_id、按状态用 start_time/end_time 条件）。
- **适合参考**：主从两表、两 .go 两 GET、OnSelectFuzzy、link、时间状态计算、列表筛外表字段、预约时间冲突校验。

---

## 二、结构化 PRD

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 只保留实现参考、SDK 写法和注意事项，不再承载旧 PRD 表格。

## 三、业务校验要点

- **预约新增/编辑**：结束时间 > 开始时间；开始时间不能为过去；会议室存在且状态为「可用」；参会人数 ≤ 会议室容纳人数；同一会议室、时间段不重叠（需排除自身 ID 做冲突检测）。
- **预约删除**：进行中的会议禁止删除，可提示「不能删除进行中的会议，请等待会议结束后再删除」。
- **预约列表筛「会议室名称」**：用 `MeetingRoom.name LIKE ?` 查出 room_id 列表，再 `room_id IN ?` 过滤预约。
- **预约列表筛「预约状态」**：用当前时间与 start_time、end_time 比较，例如：待开始 `start_time > now`，进行中 `start_time <= now AND end_time > now`，已结束 `end_time <= now`。

---

## 四、文件与路由

| 文件                   | 说明           | 注册路由                    |
|------------------------|----------------|-----------------------------|
| meeting_room.go        | 会议室管理     | GET meeting_room_list.table     |
| meeting_room_booking.go | 预约管理     | GET meeting_room_booking_list.table |

**OnSelectFuzzyMap**：预约表 Template 中 `"room_id": onSelectFuzzyMeetingRoom`，会议室下拉只显示状态为「可用」的会议室。

---

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/tables/meeting`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### meeting_room.go

```go
package meeting

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 会议室信息管理 ================

// MeetingRoom 会议室信息表
type MeetingRoom struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:会议室ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time          `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Name      string `json:"name" gorm:"column:name;comment:会议室名称" widget:"name:会议室名称;type:input" validate:"required,min=2,max=50"`
	Type      string `json:"type" gorm:"column:type;comment:会议室类型" widget:"name:会议室类型;type:select;options:小型,中型,大型,会议室,培训室,多功能厅;options_colors:909399,409EFF,67C23A,E6A23C,F56C6C,9C27B0" validate:"required"`
	Capacity  int    `json:"capacity" gorm:"column:capacity;comment:容纳人数" widget:"name:容纳人数;type:number" validate:"required,min=1,max=1000"`
	Equipment string `json:"equipment" gorm:"column:equipment;type:text;comment:设备配置" widget:"name:设备配置;type:text_area"`
	Location  string `json:"location" gorm:"column:location;comment:位置信息" widget:"name:位置信息;type:input" validate:"required,min=2,max=100"`
	Status    string `json:"status" gorm:"column:status;comment:状态;default:可用" widget:"name:状态;type:select;options:可用,维护中,停用;options_colors:67C23A,E6A23C,F56C6C;render_default:可用" validate:"required,oneof=可用 维护中 停用"`
}

func (MeetingRoom) TableName() string {
	return "crm_meeting_room"
}

// MeetingRoomListReq 会议室列表请求
type MeetingRoomListReq struct {
	Name   string `json:"name" form:"name" widget:"name:会议室名称;type:input"`
	Type   string `json:"type" form:"type" widget:"name:会议室类型;type:select;options:小型会议室,中型会议室,大型会议室,培训室,视频会议室"`
	Status string `json:"status" form:"status" widget:"name:状态;type:select;options:可用,维护中,停用;options_colors:67C23A,E6A23C,F56C6C"`

	query.PageSortReq `widget:"-"`
}

// MeetingRoomList 会议室管理
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
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Type != "" {
		queryDB = queryDB.Where("type = ?", req.Type)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var rooms []MeetingRoom
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rooms).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      rooms,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// MeetingRoomListTemplate 会议室管理配置
var MeetingRoomListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "会议室管理",
		Desc:         `会议室信息的增删改查管理，包括会议室名称、类型、容纳人数、设备配置、位置信息、状态等`,
		Tags:         []string{"会议室系统", "会议室管理"},
		Request:      &MeetingRoomListReq{},
		CreateTables: []interface{}{&MeetingRoom{}},
	},
	AutoCrudTable: &MeetingRoom{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row MeetingRoom
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		err := db.Create(&row).Error
		if err != nil {
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
		err := db.Model(&MeetingRoom{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update meeting_room err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&MeetingRoom{}).Delete(&MeetingRoom{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete meeting_room err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ 模糊搜索回调 ================

// onSelectFuzzyMeetingRoom 会议室选择的模糊搜索回调（预约等场景选择会议室时使用）
func onSelectFuzzyMeetingRoom(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var rooms []MeetingRoom

	db = db.Model(&MeetingRoom{}).
		Where("status = ?", "可用") // 只显示可用的会议室

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ? OR type LIKE ? OR location LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
			Limit(20)
	}

	db.Find(&rooms)

	items := make([]*callback.SelectFuzzyItem, 0)
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
		MaxSelections: 1, // 只能单选
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

// ================ API 注册 ================

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
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 会议室预约管理 ================

// MeetingRoomBooking 会议室预约表
type MeetingRoomBooking struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:预约ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time          `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	RoomID   int          `json:"room_id" gorm:"column:room_id;comment:会议室ID;index" widget:"name:会议室;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Room     *MeetingRoom `json:"-" widget:"-" gorm:"foreignKey:RoomID;references:ID"`
	RoomName string       `json:"room_name" gorm:"-" widget:"name:会议室名称;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	RoomLink string       `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	Booker        string `json:"booker" gorm:"column:booker;comment:预约人" widget:"name:预约人;type:user;render_default:Me()" validate:"required"`
	Attendees     string `json:"attendees" gorm:"column:attendees;type:text;comment:参会人（逗号分隔）" widget:"name:参会人;type:users"`
	Subject       string `json:"subject" gorm:"column:subject;comment:会议主题" widget:"name:会议主题;type:input" validate:"required,min=2,max=200"`
	Description   string `json:"description" gorm:"column:description;type:text;comment:会议描述" widget:"name:会议描述;type:text_area"`
	StartTime     types.Time  `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间;index" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP" validate:"required"`
	EndTime       types.Time  `json:"end_time" gorm:"column:end_time;type:datetime;comment:结束时间;index" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	AttendeeCount int    `json:"attendee_count" gorm:"column:attendee_count;comment:参会人数" widget:"name:参会人数;type:number" validate:"required,min=1"`
	Status        string `json:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	Remark        string `json:"remark" gorm:"column:remark;type:text;comment:备注" widget:"name:备注;type:text_area"`
}

func (MeetingRoomBooking) TableName() string {
	return "crm_meeting_room_booking"
}

// MeetingRoomBookingListReq 预约列表请求
//
// 注意：RoomName 为外表字段（来自 crm_meeting_room），Status 为计算字段筛选，需在 Handler 中手动处理。
type MeetingRoomBookingListReq struct {
	RoomName string `json:"room_name" form:"room_name" gorm:"-" widget:"name:会议室名称;type:input"`
	Status   string `json:"status" form:"status" gorm:"-" widget:"name:预约状态;type:select;options:待开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`

	query.PageSortReq `widget:"-"`
}

// MeetingRoomBookingList 会议室预约管理
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
		if err := db.Model(&MeetingRoom{}).
			Where("name LIKE ?", "%"+req.RoomName+"%").
			Pluck("id", &roomIDs).Error; err == nil && len(roomIDs) > 0 {
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

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var bookings []MeetingRoomBooking
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&bookings).Error; err != nil {
		return err
	}

	for i := range bookings {
		if bookings[i].Room != nil {
			bookings[i].RoomName = bookings[i].Room.Name
		}
		bookings[i].Status = calculateBookingStatus(bookings[i].StartTime, bookings[i].EndTime)
		params := MeetingRoom{
			ID: bookings[i].RoomID,
		}
		bookings[i].RoomLink, _ = ctx.BuildFunctionUrlWithText("meeting_room_list.table", params, "查看会议室详情")
	}

	return resp.Table(response.TableResult{
		Items:      bookings,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// MeetingRoomBookingListTemplate 预约管理配置
var MeetingRoomBookingListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "会议室预约管理",
		Desc:         `会议室预约的增删改查管理，包括会议室选择、预约人、会议主题、时间安排、参会人数、预约状态等`,
		Tags:         []string{"会议室系统", "预约管理"},
		Request:      &MeetingRoomBookingListReq{},
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

		if err := validateBookingTime(db, &row); err != nil {
			return nil, err
		}

		err := db.Create(&row).Error
		if err != nil {
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

		if req.IsFieldUpdated("start_time") || req.IsFieldUpdated("end_time") || req.IsFieldUpdated("room_id") {
			tempBooking := MeetingRoomBooking{
				RoomID:    currentBooking.RoomID,
				StartTime: currentBooking.StartTime,
				EndTime:   currentBooking.EndTime,
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
			if err := validateBookingTimeExclude(db, &tempBooking, req.GetId()); err != nil {
				return nil, err
			}
		}
		err := db.Model(&MeetingRoomBooking{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
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
			status := calculateBookingStatus(booking.StartTime, booking.EndTime)
			if status == "进行中" {
				return nil, fmt.Errorf("不能删除进行中的会议，请等待会议结束后再删除")
			}
		}

		err := db.Model(&MeetingRoomBooking{}).Delete(&MeetingRoomBooking{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete meeting_room_booking err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ 辅助函数 ================

// validateBookingTime 验证预约时间（新增时使用）
func validateBookingTime(db *gorm.DB, booking *MeetingRoomBooking) error {
	if !booking.EndTime.Time().After(booking.StartTime.Time()) {
		return fmt.Errorf("结束时间必须晚于开始时间")
	}

	now := time.Now()
	if booking.StartTime.Time().Before(now) {
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

// validateBookingTimeExclude 验证预约时间（更新时使用，排除指定ID）
func validateBookingTimeExclude(db *gorm.DB, booking *MeetingRoomBooking, excludeID int) error {
	if !booking.EndTime.Time().After(booking.StartTime.Time()) {
		return fmt.Errorf("结束时间必须晚于开始时间")
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

// calculateBookingStatus 计算预约状态（实时计算，不存储到数据库）
func calculateBookingStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "待开始"
	} else if now.Before(endTime.Time()) {
		return "进行中"
	}
	return "已结束"
}

// ================ API 注册 ================

func init() {
	packageContext.GET("meeting_room_booking_list.table", MeetingRoomBookingList, MeetingRoomBookingListTemplate)
}
```
