package auction

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ================ 成交记录查询 ================

// DealRecord 成交记录表（只读）
type DealRecord struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:成交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	ItemID   int          `json:"item_id" gorm:"column:item_id;comment:拍品ID;index" widget:"-"`
	Item     *AuctionItem `json:"-" gorm:"-" widget:"-" gorm:"foreignKey:ItemID;references:ID"`
	ItemName string       `json:"item_name" gorm:"-" widget:"name:拍品名称;type:text" hide:"create,update"`

	SessionID   int             `json:"session_id" gorm:"column:session_id;comment:所属场次ID;index" widget:"-"`
	Session     *AuctionSession `json:"-" gorm:"-" widget:"-" gorm:"foreignKey:SessionID;references:ID"`
	SessionName string          `json:"session_name" gorm:"-" widget:"name:所属场次;type:text" hide:"create,update"`

	Price     float64 `json:"price" gorm:"column:price;type:decimal(12,2);comment:成交价" widget:"name:成交价;type:float;min:0;precision:2;unit:元" hide:"create,update"`
	Consignor string  `json:"consignor" gorm:"column:consignor;comment:委托方" widget:"name:委托方;type:user" hide:"create,update"`
	Buyer     string  `json:"buyer" gorm:"column:buyer;comment:买受人" widget:"name:买受人;type:user" hide:"create,update"`
}

func (DealRecord) TableName() string {
	return "auction_deal_record"
}

// DealRecordListReq 成交记录列表请求
type DealRecordListReq struct {
	ItemName  string `json:"item_name" form:"item_name" widget:"name:拍品名称;type:input"`
	SessionID string `json:"session_id" form:"session_id" widget:"name:所属场次;type:input"`
	Buyer     string `json:"buyer" form:"buyer" widget:"name:买受人;type:user"`
	Consignor string `json:"consignor" form:"consignor" widget:"name:委托方;type:user"`
	StartDate string `json:"start_date" form:"start_date" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// DealRecordList 成交记录查询
func DealRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req DealRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&DealRecord{})

	if req.ItemName != "" {
		var itemIDs []int
		db.Model(&AuctionItem{}).Where("name LIKE ?", "%"+req.ItemName+"%").Pluck("id", &itemIDs)
		if len(itemIDs) > 0 {
			queryDB = queryDB.Where("item_id IN ?", itemIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}
	if req.SessionID != "" {
		queryDB = queryDB.Where("session_id = ?", req.SessionID)
	}
	if req.Buyer != "" {
		queryDB = queryDB.Where("buyer = ?", req.Buyer)
	}
	if req.Consignor != "" {
		queryDB = queryDB.Where("consignor = ?", req.Consignor)
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

	var records []DealRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	// 预加载关联信息
	for i := range records {
		var item AuctionItem
		if db.Where("id = ?", records[i].ItemID).First(&item).Error == nil {
			records[i].Item = &item
			records[i].ItemName = item.Name
		}
		var session AuctionSession
		if db.Where("id = ?", records[i].SessionID).First(&session).Error == nil {
			records[i].Session = &session
			records[i].SessionName = session.Name
		}
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// DealRecordListTemplate 成交记录查询配置（只读，无新增/编辑/删除）
var DealRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "成交记录查询",
		Desc:         `只读查看拍卖结束后成交的拍品记录`,
		Tags:         []string{"拍卖会系统", "成交记录"},
		Request:      &DealRecordListReq{},
		CreateTables: []interface{}{&DealRecord{}},
	},
	AutoCrudTable: &DealRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("deal_record_list.table", DealRecordList, DealRecordListTemplate)
}
