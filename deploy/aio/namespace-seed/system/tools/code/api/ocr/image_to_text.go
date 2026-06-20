// image_to_text.go：识别图片中的文字（OCR），路由 POST /tesseract/image_to_text.form
// 上传图片，选择语言，返回识别出的文本。依赖系统已安装 tesseract 及对应语言包。

package ocr

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ImageToTextReq 识别图片中的文字请求
type ImageToTextReq struct {
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*,.png,.jpg,.jpeg,.bmp,.tiff,.tif,.webp;max_size:50MB;max_count:20" validate:"required"`
	Language   string `json:"language" widget:"name:识别语言;type:select;options:chi_sim+eng,chi_sim,eng;render_default:chi_sim+eng"`
}

// ImageToTextResp 识别结果响应
type ImageToTextResp struct {
	OutputText string `json:"output_text" widget:"name:识别文本;type:text_area"`
	Summary    string `json:"summary" widget:"name:说明;type:text_area"`
}

// ImageToText 入口
func ImageToText(ctx *app.Context, resp response.Response) error {
	var req ImageToTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoImageToText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoImageToText 对每张图片执行 tesseract 识别，汇总文本返回
func DoImageToText(ctx *app.Context, req *ImageToTextReq) (*ImageToTextResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	lang := strings.TrimSpace(req.Language)
	if lang == "" {
		lang = "chi_sim+eng"
	}

	var textParts []string
	var summaryParts []string
	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[tesseract/ImageToText] 文件 %s 无本地路径，跳过", filepath.Base(file))
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		text, err := RunTesseractOnFile(ctx, file, lang)
		if err != nil {
			summaryParts = append(summaryParts, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		if text != "" {
			if len(files) > 1 {
				textParts = append(textParts, fmt.Sprintf("【%s】\n%s", filepath.Base(file), text))
			} else {
				textParts = append(textParts, text)
			}
		}
		summaryParts = append(summaryParts, fmt.Sprintf("成功 %s", filepath.Base(file)))
	}

	return &ImageToTextResp{
		OutputText: strings.Join(textParts, "\n\n"),
		Summary:    strings.Join(summaryParts, "\n"),
	}, nil
}

var ImageToTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "识别图片中的文字",
		Desc:     `OCR：上传图片，识别图中的文字并返回文本。支持中文、英文或中英混合，可多图。依赖 tesseract 及语言包（如 chi_sim、eng）。`,
		Tags:     []string{"图片", "OCR", "文字识别", "Tesseract"},
		Request:  &ImageToTextReq{},
		Response: &ImageToTextResp{},
	},
}

func init() {
	packageContext.POST("image.form", ImageToText, ImageToTextTemplate)
}
