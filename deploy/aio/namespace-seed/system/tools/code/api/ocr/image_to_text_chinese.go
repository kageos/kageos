// image_to_text_chinese.go：识别图片中的中文，路由 POST /tesseract/image_to_text_chinese.form
// 上传图片，仅做中文识别（chi_sim），返回识别文本。适合纯中文场景。

package ocr

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ImageToTextChineseReq 识别图片中的中文请求（无语言参数，固定 chi_sim）
type ImageToTextChineseReq struct {
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*,.png,.jpg,.jpeg,.bmp,.tiff,.tif,.webp;max_size:50MB;max_count:20" validate:"required"`
}

// ImageToTextChineseResp 与 ImageToTextResp 一致
type ImageToTextChineseResp struct {
	OutputText string `json:"output_text" widget:"name:识别文本;type:text_area"`
	Summary    string `json:"summary" widget:"name:说明;type:text_area"`
}

// ImageToTextChinese 入口：固定使用中文语言包
func ImageToTextChinese(ctx *app.Context, resp response.Response) error {
	var req ImageToTextChineseReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	// 复用 DoImageToText，语言固定为 chi_sim
	wrapped := &ImageToTextReq{
		InputFiles: req.InputFiles,
		Language:   "chi_sim",
	}
	res, err := DoImageToText(ctx, wrapped)
	if err != nil {
		return err
	}
	return resp.Form(&ImageToTextChineseResp{
		OutputText: res.OutputText,
		Summary:    res.Summary,
	}).Build()
}

var ImageToTextChineseTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "识别图片中的中文",
		Desc:     `OCR：上传图片，仅识别图中的中文并返回文本。适合纯中文截图、文档。依赖 tesseract 及 chi_sim 语言包。`,
		Tags:     []string{"图片", "OCR", "中文", "文字识别", "Tesseract"},
		Request:  &ImageToTextChineseReq{},
		Response: &ImageToTextChineseResp{},
	},
}

func init() {
	packageContext.POST("chinese_image.form", ImageToTextChinese, ImageToTextChineseTemplate)
}
