// rating_submit.go
// 提交评价表单：选择评价对象、评分、评论、上传附件

package rating

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// RatingSubmitReq 提交评价请求
type RatingSubmitReq struct {
	ObjectID    int     `json:"object_id" widget:"name:选择评价对象;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Rating      float64 `json:"rating" widget:"name:评分;type:rate;max:5;allow_half:true;texts:很差,较差,一般,满意,惊喜" validate:"required,min=0.5,max=5"`
	Comment     string  `json:"comment" widget:"name:评论;type:text_area;placeholder:请输入您的评价（可选）" validate:"max=1000"`
	Attachments string  `json:"attachments" widget:"name:图片附件;type:files;accept:image/*;max_count:9;thumbnail:true"`
}

// RatingSubmitResp 提交评价响应
type RatingSubmitResp struct {
	Success      bool   `json:"success" widget:"name:是否成功;type:switch"`
	Message      string `json:"message" widget:"name:处理结果;type:text_area"`
	ObjectName   string `json:"object_name" widget:"name:评价对象;type:text"`
	Rating       string `json:"rating" widget:"name:评分;type:text"`
	SubmitTime   string `json:"submit_time" widget:"name:提交时间;type:datetime"`
	FunctionLink string `json:"function_link" widget:"name:查看全部评价;type:link;target:_blank"`
}

// ================ 模糊搜索回调 ================

// submitOnSelectFuzzyObject 评价对象下拉回调（提交表单用）
func submitOnSelectFuzzyObject(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[submitOnSelectFuzzyObject] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[submitOnSelectFuzzyObject]： 数据库连接失败, req: %+v", req)
	}

	var objects []RatingObject
	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("name LIKE ? OR object_type LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").
			Limit(20)
	}
	db.Find(&objects)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, obj := range objects {
		items = append(items, &callback.SelectFuzzyItem{
			Value: obj.ID,
			Label: fmt.Sprintf("%s (%s)", obj.Name, obj.ObjectType),
			DisplayInfo: map[string]interface{}{
				"事物名称": obj.Name,
				"类型":   obj.ObjectType,
				"当前评分": fmt.Sprintf("%.1f", obj.AverageRating),
				"评价人数": obj.RatingCount,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
		Statistics: map[string]interface{}{
			"事物名称": statistics.Value("事物名称"),
			"类型":   statistics.Value("类型"),
			"当前评分": statistics.Value("当前评分"),
			"评价人数": statistics.Value("评价人数"),
		},
	}, nil
}

// ================ 提交评价 ================

// RatingSubmit 提交评价入口
func RatingSubmit(ctx *app.Context, resp response.Response) error {
	var req RatingSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoRatingSubmit(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoRatingSubmit 提交评价业务逻辑
func DoRatingSubmit(ctx *app.Context, req *RatingSubmitReq) (*RatingSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoRatingSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoRatingSubmit]： 数据库连接失败, req: %+v", req)
	}

	userInfo := ctx.GetRequestUser()

	var obj RatingObject
	if err := db.Where("id = ?", req.ObjectID).First(&obj).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("评价对象不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoRatingSubmit] 查询评价对象失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoRatingSubmit]： 查询评价对象失败, req: %+v, err: %w", req, err)
	}

	var count int64
	if err := db.Model(&RatingRecord{}).Where("object_id = ? AND submitter = ?", req.ObjectID, userInfo).Count(&count).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoRatingSubmit] 查询已有评价失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoRatingSubmit]： 查询已有评价失败, req: %+v, err: %w", req, err)
	}
	if count > 0 {
		return nil, fmt.Errorf("您已经评价过该对象了，每人每个对象只能评价一次")
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		record := &RatingRecord{
			ObjectID:    req.ObjectID,
			Rating:      req.Rating,
			Comment:     req.Comment,
			Attachments: req.Attachments,
			Submitter:   userInfo,
		}
		if err := tx.Create(record).Error; err != nil {
			return fmt.Errorf("创建评价记录失败: %v", err)
		}

		var stat struct {
			Total float64
			Count int64
		}
		if err := tx.Model(&RatingRecord{}).Where("object_id = ?", req.ObjectID).
			Select("COALESCE(SUM(rating), 0) as total, COUNT(*) as count").
			Scan(&stat).Error; err != nil {
			return fmt.Errorf("统计评分失败: %v", err)
		}
		totalRating := stat.Total
		ratingCount := stat.Count

		newAvg := totalRating / float64(ratingCount)
		newAvg = float64(int(newAvg*100+0.5)) / 100

		if err := tx.Model(&RatingObject{}).Where("id = ?", req.ObjectID).Updates(map[string]interface{}{
			"average_rating": newAvg,
			"rating_count":   ratingCount,
		}).Error; err != nil {
			return fmt.Errorf("更新平均评分失败: %v", err)
		}

		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoRatingSubmit] 事务失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoRatingSubmit]： 事务失败, req: %+v, err: %w", req, err)
	}

	params := RatingRecord{ObjectID: req.ObjectID}
	functionLink, _ := ctx.BuildFunctionUrlWithText("rating_record_list.table", params, "查看全部评价")

	return &RatingSubmitResp{
		Success:      true,
		Message:      "评价提交成功！",
		ObjectName:   obj.Name,
		Rating:       fmt.Sprintf("%.1f", req.Rating),
		SubmitTime:   time.Now().Format("2006-01-02 15:04:05"),
		FunctionLink: functionLink,
	}, nil
}

// RatingSubmitTemplate 提交评价配置
var RatingSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交评价",
		Desc:     `选择评价对象进行评分和评论，支持上传图片附件`,
		Tags:     []string{"评分系统", "评价提交"},
		Request:  &RatingSubmitReq{},
		Response: &RatingSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"object_id": submitOnSelectFuzzyObject,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("rating_submit.form", RatingSubmit, RatingSubmitTemplate)
}
