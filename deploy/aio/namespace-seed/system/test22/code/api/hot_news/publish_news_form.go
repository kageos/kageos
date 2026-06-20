package hot_news

import (
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// PublishNewsReq 发布新闻请求
type PublishNewsReq struct {
	// 分类名称
	Category string `json:"category" widget:"name:分类名称;type:select;options:科技,财经,社会,娱乐,体育;options_colors:409EFF,E6A23C,67C23A,9C27B0,FF9800" validate:"required,oneof=科技 财经 社会 娱乐 体育"`

	// 新闻标题
	Title string `json:"title" widget:"name:新闻标题;type:input" validate:"required,min=2,max=200"`

	// 新闻内容（富文本）
	Content string `json:"content" widget:"name:新闻内容;type:richtext;height:300" validate:"required"`
}

// PublishNewsResp 发布新闻响应
type PublishNewsResp struct {
	// 提交结果
	Result string `json:"result" widget:"name:提交结果;type:input"`
}

var PublishNewsTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "发布新闻",
		Request:  &PublishNewsReq{},
		Response: &PublishNewsResp{},
	},
}

// PublishNews 发布新闻
func PublishNews(ctx *app.Context, resp response.Response) error {
	var req PublishNewsReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 创建新闻记录，状态默认为草稿
	news := &News{
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Status:    "草稿", // 默认草稿状态
		ViewCount: 0,
		CreatedBy: ctx.GetRequestUser(),
	}

	if err := db.Create(news).Error; err != nil {
		logger.Errorf(ctx, "[PublishNews] 发布新闻失败, req: %+v, err: %v", req, err)
		return err
	}

	return resp.Form(&PublishNewsResp{
		Result: "新闻发布成功",
	}).Build()
}

func init() {
	packageContext.POST("publish_news.form", PublishNews, PublishNewsTemplate)
}
