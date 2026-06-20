package hot_news

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// News 新闻管理
type News struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 新闻标题
	Title string `json:"title" gorm:"column:title" widget:"name:新闻标题;type:input" validate:"required,min=2,max=200"`

	// 新闻内容（富文本）
	Content string `json:"content" gorm:"column:content;type:text" widget:"name:新闻内容;type:richtext;height:300" validate:"required"`

	// 分类名称
	Category string `json:"category" gorm:"column:category" widget:"name:分类名称;type:select;options:科技,财经,社会,娱乐,体育;options_colors:409EFF,E6A23C,67C23A,9C27B0,FF9800" validate:"required,oneof=科技 财经 社会 娱乐 体育"`

	// 发布状态：草稿、已发布、已下架
	Status string `json:"status" gorm:"column:status" widget:"name:发布状态;type:select;options:草稿,已发布,已下架;options_colors:909399,67C23A,F56C6C;render_default:草稿" validate:"required,oneof=草稿 已发布 已下架"`

	// 浏览量（仅展示，不在表单中填写）
	ViewCount int `json:"view_count" gorm:"column:view_count;default:0" widget:"name:浏览量;type:integer" hide:"create,update"`

	// 创建人
	CreatedBy string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (n *News) TableName() string {
	return "news"
}

// NewsListReq 列表查询请求
type NewsListReq struct {
	Title     string `json:"title" form:"title" widget:"name:新闻标题;type:input"`
	Category  string `json:"category" form:"category" widget:"name:分类名称;type:select;options:科技,财经,社会,娱乐,体育;options_colors:409EFF,E6A23C,67C23A,9C27B0,FF9800"`
	Status    string `json:"status" form:"status" widget:"name:发布状态;type:select;options:草稿,已发布,已下架;options_colors:909399,67C23A,F56C6C"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

var NewsTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "新闻管理",
		Request:      &NewsListReq{},
		CreateTables: []interface{}{&News{}},
	},
	AutoCrudTable: &News{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row News
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		// 默认状态为草稿（已在 widget render_default 中设置）
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "[News-Add] 创建新闻失败, err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields News
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		err := db.Model(&News{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "[News-Update] 更新新闻失败, err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&News{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "[News-Delete] 删除新闻失败, err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// NewsList 新闻列表
func NewsList(ctx *app.Context, resp response.Response) error {
	var req NewsListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[News-List] 参数绑定失败, err: %v", err)
		return err
	}

	db := ctx.GetGormDB()
	queryDB := db.Model(&News{})

	// 按新闻标题模糊搜索
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	// 按分类筛选
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}
	// 按发布状态筛选
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	// 按创建时间范围筛选
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	// 排序
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}

	// 统计总数
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[News-List] 统计失败, err: %v", err)
		return err
	}

	// 分页查询
	var lists []*News
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[News-List] 查询失败, err: %v", err)
		return err
	}

	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("news_list.table", NewsList, NewsTemplate)
}
