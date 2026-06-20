package midnight_pub

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// PubCharacter 酒馆角色
type PubCharacter struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CharacterName string         `json:"character_name" gorm:"column:character_name" widget:"name:角色名;type:input" validate:"required"`
	CharacterCode string         `json:"character_code" gorm:"column:character_code" widget:"name:角色代码;type:input" validate:"required"`
	Profession    string         `json:"profession" gorm:"column:profession" widget:"name:职业;type:input" validate:"required"`
	Personality   string         `json:"personality" gorm:"column:personality" widget:"name:性格;type:text_area" validate:"required"`
	Background    string         `json:"background" gorm:"column:background" widget:"name:背景故事;type:text_area"`
	AppearCount   int            `json:"appear_count" gorm:"column:appear_count;default:0" widget:"name:出场次数;type:integer"`
}

func (p *PubCharacter) TableName() string {
	return "midnight_pub_pub_character"
}

var PubCharacterTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "酒馆角色",
		Request: &PubCharacterListReq{},
		CreateTables: []interface{}{
			&PubCharacter{},
		},
	},
	AutoCrudTable: &PubCharacter{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row PubCharacter
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "PubCharacter Create err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		updates := req.ChangedFields()
		err := db.Model(&PubCharacter{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "PubCharacter Update err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Where("id in (?)", req.GetIds()).Delete(&PubCharacter{}).Error
		if err != nil {
			logger.Errorf(ctx, "PubCharacter Delete err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type PubCharacterListReq struct {
	CharacterName     string `json:"character_name" form:"character_name" widget:"name:角色名;type:input"`
	query.PageSortReq `widget:"-"`
}

func PubCharacterList(ctx *app.Context, resp response.Response) error {
	var req PubCharacterListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "PubCharacterList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&PubCharacter{})
	if req.CharacterName != "" {
		queryDB = queryDB.Where("character_name LIKE ?", "%"+req.CharacterName+"%")
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []*PubCharacter
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
	packageContext.GET("pub_character_list.table", PubCharacterList, PubCharacterTemplate)
}
