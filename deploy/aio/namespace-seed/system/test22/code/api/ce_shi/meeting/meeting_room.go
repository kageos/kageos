package meeting

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type MeetingRoom struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:会议室ID;type:ID" hide:"create,update"`                                                     // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

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
		if err := db.Model(&MeetingRoom{}).Delete(&MeetingRoom{}, "id in ?", req.GetIds()).Error; err != nil {
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
