package auction

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ================ 竞价记录查询 ================

// BidRecord 竞价记录表（只读）
type BidRecord struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:竞价时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	ItemID   int          `json:"item_id" gorm:"column:item_id;comment:拍品ID;index" widget:"-"`
	Item     *AuctionItem `json:"-" gorm:"-" widget:"-" gorm:"foreignKey:ItemID;references:ID"`
	ItemName string       `json:"item_name" gorm:"-" widget:"name:拍品名称;type:text" hide:"create,update"`

	SessionID   int             `json:"session_id" gorm:"column:session_id;comment:所属场次ID;index" widget:"-"`
	Session     *AuctionSession `json:"-" gorm:"-" widget:"-" gorm:"foreignKey:SessionID;references:ID"`
	SessionName string          `json:"session_name" gorm:"-" widget:"name:所属场次;type:text" hide:"create,update"`

	Bidder     string  `json:"bidder" gorm:"column:bidder;comment:竞拍人" widget:"name:竞拍人;type:user" hide:"create,update"`
	Amount     float64 `json:"amount" gorm:"column:amount;type:decimal(12,2);comment:出价金额" widget:"name:出价金额;type:float;min:0;precision:2;unit:元" hide:"create,update"`
	IsTopPrice bool    `json:"is_top_price" gorm:"column:is_top_price;comment:是否最高价" widget:"name:是否最高价;type:switch" hide:"create,update"`
}

func (BidRecord) TableName() string {
	return "auction_bid_record"
}

// BidRecordListReq 竞价记录列表请求
type BidRecordListReq struct {
	ItemName  string `json:"item_name" form:"item_name" widget:"name:拍品名称;type:input"`
	SessionID string `json:"session_id" form:"session_id" widget:"name:所属场次;type:input"`
	Bidder    string `json:"bidder" form:"bidder" widget:"name:竞拍人;type:user"`
	StartDate string `json:"start_date" form:"start_date" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// BidRecordList 竞价记录查询
func BidRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req BidRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&BidRecord{})

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
	if req.Bidder != "" {
		queryDB = queryDB.Where("bidder = ?", req.Bidder)
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

	var records []BidRecord
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

// BidRecordListTemplate 竞价记录查询配置（只读，无新增/编辑/删除）
var BidRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "竞价记录查询",
		Desc:         `只读查看用户提交竞价后产生的竞价记录`,
		Tags:         []string{"拍卖会系统", "竞价记录"},
		Request:      &BidRecordListReq{},
		CreateTables: []interface{}{&BidRecord{}},
	},
	AutoCrudTable: &BidRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("bid_record_list.table", BidRecordList, BidRecordListTemplate)
}
