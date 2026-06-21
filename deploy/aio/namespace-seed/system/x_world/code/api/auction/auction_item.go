package auction

import (
	"errors"
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

// ================ 拍品管理 ================

// AuctionItem 拍品表
type AuctionItem struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	SessionID   int             `json:"session_id" gorm:"column:session_id;comment:所属场次ID;index" widget:"name:所属场次;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Session     *AuctionSession `json:"-" gorm:"-" widget:"-" gorm:"foreignKey:SessionID;references:ID"`
	SessionName string          `json:"session_name" gorm:"-" widget:"name:所属场次;type:text" hide:"create,update"`

	Name         string  `json:"name" gorm:"column:name;comment:拍品名称" widget:"name:拍品名称;type:input" validate:"required,min=2,max=200"`
	Description  string  `json:"description" gorm:"column:description;type:text;comment:拍品描述" widget:"name:拍品描述;type:text_area"`
	Images       string  `json:"images" gorm:"column:images;type:text;comment:拍品图片" widget:"name:拍品图片;type:files;accept:image/*;max_count:5;thumbnail:true;list_preview:true"`
	StartPrice   float64 `json:"start_price" gorm:"column:start_price;type:decimal(12,2);comment:起拍价" widget:"name:起拍价;type:float;min:0;precision:2;unit:元" validate:"required,gte=0"`
	CurrentPrice float64 `json:"current_price" gorm:"column:current_price;type:decimal(12,2);default:0;comment:当前最高价" widget:"name:当前最高价;type:float;min:0;precision:2;unit:元" hide:"create,update"`
	TopBidder    string  `json:"top_bidder" gorm:"column:top_bidder;comment:最高出价人" widget:"name:最高出价人;type:user" hide:"create,update"`
	Status       string  `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,竞价中,已成交,已流拍;options_colors:909399,409EFF,67C23A,F56C6C" hide:"create,update"`
}

func (AuctionItem) TableName() string {
	return "auction_item"
}

// AuctionItemListReq 拍品列表请求
type AuctionItemListReq struct {
	Name      string `json:"name" form:"name" widget:"name:拍品名称;type:input"`
	SessionID string `json:"session_id" form:"session_id" widget:"name:所属场次;type:select" callback:"OnSelectFuzzy"`
	Status    string `json:"status" form:"status" widget:"name:拍品状态;type:select;options:未开始,竞价中,已成交,已流拍;options_colors:909399,409EFF,67C23A,F56C6C"`
	CreatedBy string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	StartDate string `json:"start_date" form:"start_date" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// AuctionItemList 拍品管理
func AuctionItemList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req AuctionItemListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&AuctionItem{})

	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.SessionID != "" {
		queryDB = queryDB.Where("session_id = ?", req.SessionID)
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}
	if req.StartDate != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndDate)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var items []AuctionItem
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}

	// 预加载场次信息并填充状态
	for i := range items {
		var session AuctionSession
		if err := db.Where("id = ?", items[i].SessionID).First(&session).Error; err == nil {
			items[i].Session = &session
			items[i].SessionName = session.Name
		}
		items[i].Status = calculateItemStatus(session, items[i].CurrentPrice, items[i].TopBidder)
	}

	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// AuctionItemListTemplate 拍品管理配置
var AuctionItemListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "拍品管理",
		Desc:         `维护每个拍卖场次下的拍品信息，包括名称、起拍价、当前最高价、图片等`,
		Tags:         []string{"拍卖会系统", "拍品管理"},
		Request:      &AuctionItemListReq{},
		CreateTables: []interface{}{&AuctionItem{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"session_id": onSelectFuzzyAuctionSession,
		},
	},
	AutoCrudTable: &AuctionItem{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row AuctionItem
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		// 校验场次存在
		var session AuctionSession
		if err := db.Where("id = ?", row.SessionID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("拍卖场次不存在")
			}
			return nil, fmt.Errorf("查询拍卖场次失败: %v", err)
		}
		// 初始化当前最高价为起拍价
		row.CurrentPrice = row.StartPrice
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create auction_item err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields AuctionItem
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()

		// 校验场次存在（如果更新了场次）
		if req.IsFieldUpdated("session_id") {
			var session AuctionSession
			if err := db.Where("id = ?", updateFields.SessionID).First(&session).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("拍卖场次不存在")
				}
				return nil, fmt.Errorf("查询拍卖场次失败: %v", err)
			}
		}

		err := db.Model(&AuctionItem{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update auction_item err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&AuctionItem{}).Delete(&AuctionItem{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete auction_item err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// onSelectFuzzyAuctionSession 拍卖场次选择的模糊搜索回调
func onSelectFuzzyAuctionSession(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var sessions []AuctionSession
	db = db.Model(&AuctionSession{})

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ?", "%"+keyword+"%").Limit(20)
	}

	db.Find(&sessions)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, s := range sessions {
		items = append(items, &callback.SelectFuzzyItem{
			Value: s.ID,
			Label: fmt.Sprintf("%s (%s 至 %s)", s.Name, s.StartTime.Time().Format("01-02 15:04"), s.EndTime.Time().Format("01-02 15:04")),
			DisplayInfo: map[string]interface{}{
				"场次名称": s.Name,
				"开始时间": s.StartTime.Time().Format("2006-01-02 15:04"),
				"结束时间": s.EndTime.Time().Format("2006-01-02 15:04"),
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"选中场次": statistics.Value("场次名称"),
			"开始时间": statistics.Value("开始时间"),
			"结束时间": statistics.Value("结束时间"),
		},
	}, nil
}

// calculateItemStatus 计算拍品状态（实时计算，不存储）
func calculateItemStatus(session AuctionSession, currentPrice float64, topBidder string) string {
	now := time.Now()
	// 根据场次状态判断
	if now.Before(session.StartTime.Time()) {
		return "未开始"
	} else if now.After(session.EndTime.Time()) {
		// 场次已结束
		if topBidder != "" && currentPrice > 0 {
			return "已成交"
		}
		return "已流拍"
	}
	// 场次进行中
	if currentPrice > 0 && topBidder != "" {
		return "竞价中"
	}
	return "未开始"
}

// ================ API 注册 ================

func init() {
	packageContext.GET("auction_item_list.table", AuctionItemList, AuctionItemListTemplate)
}
