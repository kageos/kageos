package midnight_pub

import (
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// PubStatus 酒馆状态
type PubStatus struct {
	ID               int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt        types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt        types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	AtmosphereTag    string         `json:"atmosphere_tag" gorm:"column:atmosphere_tag" widget:"name:氛围标签;type:input"`
	LateNightLevel   string         `json:"late_night_level" gorm:"column:late_night_level" widget:"name:夜深程度;type:input"`
	PopularityIndex  int            `json:"popularity_index" gorm:"column:popularity_index;default:50" widget:"name:人气指数;type:integer;min:0;max:100"`
	ActiveCharacters string         `json:"active_characters" gorm:"column:active_characters" widget:"name:活跃角色;type:input"`
	HotTopic         string         `json:"hot_topic" gorm:"column:hot_topic" widget:"name:热门话题;type:input"`
}

func (p *PubStatus) TableName() string {
	return "midnight_pub_pub_status"
}

var PubStatusTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "酒馆状态",
		Request: &PubStatusListReq{},
		CreateTables: []interface{}{
			&PubStatus{},
		},
	},
	AutoCrudTable: &PubStatus{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row PubStatus
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "PubStatus Create err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		updates := req.ChangedFields()
		err := db.Model(&PubStatus{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "PubStatus Update err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Where("id in (?)", req.GetIds()).Delete(&PubStatus{}).Error
		if err != nil {
			logger.Errorf(ctx, "PubStatus Delete err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type PubStatusListReq struct {
	query.PageSortReq `widget:"-"`
}

func PubStatusList(ctx *app.Context, resp response.Response) error {
	var req PubStatusListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "PubStatusList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&PubStatus{})
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []*PubStatus
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("pub_status_list.table", PubStatusList, PubStatusTemplate)
}
