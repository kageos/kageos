package auction

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 竞拍出价表单 ================

// BidSubmitReq 竞拍出价请求
type BidSubmitReq struct {
	ItemID int     `json:"item_id" widget:"name:拍品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Amount float64 `json:"amount" widget:"name:出价金额;type:float;min:0;precision:2;unit:元;placeholder:请输入出价金额" validate:"required,gt=0"`
}

// BidSubmitResp 竞拍出价响应
type BidSubmitResp struct {
	Result  string `json:"result" widget:"name:出价结果;type:input"`
	BidTime string `json:"bid_time" widget:"name:出价时间;type:datetime"`
}

// BidSubmit 竞拍出价
func BidSubmit(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req BidSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 1. 查询拍品信息
	var item AuctionItem
	if err := db.Where("id = ?", req.ItemID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("拍品不存在")
		}
		return fmt.Errorf("查询拍品失败: %v", err)
	}

	// 2. 查询所属场次信息
	var session AuctionSession
	if err := db.Where("id = ?", item.SessionID).First(&session).Error; err != nil {
		return fmt.Errorf("查询拍卖场次失败: %v", err)
	}

	// 3. 校验拍品状态（必须为竞价中）
	now := time.Now()
	if now.Before(session.StartTime.Time()) {
		return fmt.Errorf("该拍品所属场次尚未开始，请等待场次开始后再出价")
	}
	if now.After(session.EndTime.Time()) {
		return fmt.Errorf("该拍品所属场次已结束，无法继续出价")
	}
	// 场次进行中，检查拍品状态
	itemStatus := calculateItemStatus(session, item.CurrentPrice, item.TopBidder)
	if itemStatus != "竞价中" && itemStatus != "未开始" {
		return fmt.Errorf("该拍品当前状态为【%s】，无法出价", itemStatus)
	}

	// 4. 校验出价金额必须大于当前最高价
	if req.Amount <= item.CurrentPrice {
		return fmt.Errorf("出价金额必须大于当前最高价 %.2f 元", item.CurrentPrice)
	}

	// 5. 获取当前用户
	currentUser := ctx.GetRequestUser()

	// 6. 在事务中执行竞价逻辑
	err := db.Transaction(func(tx *gorm.DB) error {
		// 6.1 将之前的最高价记录标记为非最高价
		tx.Model(&BidRecord{}).Where("item_id = ? AND is_top_price = ?", item.ID, true).Update("is_top_price", false)

		// 6.2 创建竞价记录
		bidRecord := BidRecord{
			ItemID:     item.ID,
			SessionID:  item.SessionID,
			Bidder:     currentUser,
			Amount:     req.Amount,
			IsTopPrice: true,
		}
		if err := tx.Create(&bidRecord).Error; err != nil {
			return fmt.Errorf("创建竞价记录失败: %v", err)
		}

		// 6.3 更新拍品的当前最高价和最高出价人
		if err := tx.Model(&AuctionItem{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
			"current_price": req.Amount,
			"top_bidder":    currentUser,
		}).Error; err != nil {
			return fmt.Errorf("更新拍品信息失败: %v", err)
		}

		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "BidSubmit transaction err: %v", err)
		return fmt.Errorf("[系统错误]-[BidSubmit] 竞价失败, req: %+v, err: %v", req, err)
	}

	return resp.Form(&BidSubmitResp{
		Result:  "出价成功",
		BidTime: time.Now().Format("2006-01-02 15:04:05"),
	}).Build()
}

// onSelectFuzzyAuctionItemForBid 竞拍出价时选择拍品的回调（只显示竞价中的拍品）
func onSelectFuzzyAuctionItemForBid(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var items []AuctionItem
	now := time.Now()

	queryDB := db.Model(&AuctionItem{}).
		Joins("LEFT JOIN auction_session ON auction_item.session_id = auction_session.id").
		Where("auction_session.start_time <= ? AND auction_session.end_time > ?", now, now)

	if req.IsByValue() {
		queryDB = queryDB.Where("auction_item.id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("auction_item.id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		queryDB = queryDB.Where("auction_item.name LIKE ?", "%"+keyword+"%").Limit(20)
	}

	queryDB.Find(&items)

	resultItems := make([]*callback.SelectFuzzyItem, 0)
	for _, item := range items {
		resultItems = append(resultItems, &callback.SelectFuzzyItem{
			Value: item.ID,
			Label: fmt.Sprintf("%s (当前最高: %.2f元)", item.Name, item.CurrentPrice),
			DisplayInfo: map[string]interface{}{
				"拍品名称":  item.Name,
				"起拍价":   item.StartPrice,
				"当前最高价": item.CurrentPrice,
				"拍品描述":  item.Description,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         resultItems,
	}, nil
}

// BidSubmitTemplate 竞拍出价表单配置
var BidSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "竞拍出价",
		Desc:     `用户选择进行中的拍品后提交出价，系统校验后写入竞价记录并更新拍品当前最高价`,
		Request:  &BidSubmitReq{},
		Response: &BidSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"item_id": onSelectFuzzyAuctionItemForBid,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("bid_submit.form", BidSubmit, BidSubmitTemplate)
}
