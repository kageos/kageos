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
