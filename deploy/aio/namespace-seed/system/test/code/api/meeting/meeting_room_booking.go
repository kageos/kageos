package meeting

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type MeetingRoomBooking struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:预约ID;type:ID" hide:"create,update"`                                                      // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

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

		if req.IsFieldUpdated("start_time") || req.IsFieldUpdated("end_time") || req.IsFieldUpdated("room_id") || req.IsFieldUpdated("attendee_count") {
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
			if err := validateBookingTimeExclude(db, &tempBooking, req.GetId()); err != nil {
				return nil, err
			}
		}

		if req.IsFieldUpdated("attendees") {
			updates["attendee_count"] = calculateAttendeeCount(updateFields.Attendees)
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

		if err := db.Model(&MeetingRoomBooking{}).Delete(&MeetingRoomBooking{}, "id in ?", req.GetIds()).Error; err != nil {
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
		toUsers := joinUsers(booking.Booker, booking.Attendees)
		if toUsers == "" {
			continue
		}

		roomName := "未知会议室"
		if booking.Room != nil && booking.Room.Name != "" {
			roomName = booking.Room.Name
		}

		startAt := booking.StartTime.Time().Format("2006-01-02 15:04")
		content := fmt.Sprintf("您预约/参与的会议《%s》将在 %s 开始，会议室：%s，请提前准备。", booking.Subject, startAt, roomName)
		// 默认 markdown，ToUsers 来自 booking 的 user/users 字段（逗号分隔格式）
		err := ctx.SendMessage(&app.SendMessageOpts{
			ToUsers: toUsers,
			Title:   "会议即将开始提醒",
			Content: content,
		})
		if err != nil {
			logger.Errorf(ctx, "Send meeting reminder failed, booking_id=%d, err=%v", booking.ID, err)
			continue
		}

		if err := db.Model(&MeetingRoomBooking{}).Where("id = ?", booking.ID).Updates(map[string]interface{}{
			"reminder_sent": true,
			"reminded_at":   time.Now(),
		}).Error; err != nil {
			logger.Errorf(ctx, "Update reminder status failed, booking_id=%d, err=%v", booking.ID, err)
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
