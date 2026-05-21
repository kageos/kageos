package widget

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/convert"
)

func init() {
	RegisterWidgetValidator(TypeFiles, validateFilesWidget)
}

// Files 文件上传组件。
//
// 使用示例：
//
//	Attachment string `json:"attachment" widget:"name:附件;type:files;accept:.pdf,image/*;max_size:10MB;max_count:3"`
//	ProductImages string `json:"product_images" widget:"name:商品图片;type:files;accept:image/*;thumbnail:true;list_preview:true"`
//
// 协议约定：
// - Go 字段必须是 string，保存逗号分隔的文件引用 refs；
// - 不使用 []string，是为了让表单协议和落库协议保持单列字符串；
// - accept/max_size/max_count 只描述上传限制，文件内容和权限仍由文件服务处理。
//
// 校验规则：
// - Go 字段必须是 string 或 *string；
// - max_count 必须是非负整数；
// - accept 每一项必须是扩展名、MIME 类型或 MIME 通配符；
// - max_size 必须是正数加单位 B/KB/MB/GB。
type Files struct {
	// Accept 文件类型限制，支持多种格式（逗号分隔）：
	// 1. 扩展名：.pdf,.doc,.docx,.jpg,.png
	// 2. MIME类型：application/pdf,image/jpeg
	// 3. MIME通配符：image/*,video/*,audio/*
	// 4. 混合使用：.pdf,image/*,video/*,application/zip
	// 示例：accept:.pdf,.doc,.docx,image/*,video/*
	// 为空则不限制类型
	Accept string `json:"accept,omitempty"`

	// MaxSize 单个文件最大大小，支持单位：B, KB, MB, GB
	// 示例：max_size:10MB, max_size:1024KB, max_size:1GB
	// 为空则使用系统默认值
	MaxSize string `json:"max_size,omitempty"`

	// MaxCount 最大上传文件数量，默认为 5
	// 示例：max_count:10
	MaxCount int `json:"max_count,omitempty"`

	// Thumbnail 是否在浏览器上传时由前端生成轻量缩略图/视频封面并关联保存。
	// 后端只保存缩略图引用，不做媒体解码或转码。
	Thumbnail bool `json:"thumbnail,omitempty"`

	// ListPreview 是否在表格列表单元格中使用缩略图/封面内联展示。
	// 通常与 thumbnail:true 配合使用。
	ListPreview bool `json:"list_preview,omitempty"`
}

func (i *Files) Config() interface{} {
	return i
}

func (i *Files) Type() string {
	return TypeFiles
}

func (i *Files) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 3)
	if i.Accept != "" {
		facts = append(facts, SemanticFact{Key: "accept", Value: i.Accept})
	}
	if i.MaxCount > 0 {
		facts = append(facts, SemanticFact{Key: "max_count", Value: fmt.Sprintf("%d", i.MaxCount)})
	}
	if i.MaxSize != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max_size", Value: i.MaxSize})
	}
	if i.Thumbnail && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "thumbnail", Value: "true"})
	}
	if i.ListPreview && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "list_preview", Value: "true"})
	}
	return facts
}

func newFiles(widgetParsed map[string]string) *Files {
	files := &Files{}

	// 从widgetParsed中解析配置
	if accept, exists := widgetParsed["accept"]; exists {
		files.Accept = accept
	}
	if maxSize, exists := widgetParsed["max_size"]; exists {
		files.MaxSize = maxSize
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		files.MaxCount = convert.ToInt(maxCount, 5)
	} else {
		// 默认值
		files.MaxCount = 5
	}
	if thumbnail, exists := widgetParsed["thumbnail"]; exists {
		files.Thumbnail = thumbnail == "true"
	}
	if listPreview, exists := widgetParsed["list_preview"]; exists {
		files.ListPreview = listPreview == "true"
	}

	return files
}

// validateFilesWidget 保证 files 组件使用字符串 refs 协议。
//
// 如果业务希望处理多个文件，仍然使用 string 字段，值形如 "file_ref_1,file_ref_2"；
// 前端 FilesWidget 会负责数组 UI 和字符串协议之间的转换。
func validateFilesWidget(ctx ValidateContext) error {
	var errs []error
	if !isStringLikeType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "files widget uses comma-separated file refs and requires string Go type, got %s", typeName(ctx.GoType)))
	}
	if err := validateNonNegativeIntTag(ctx, "max_count"); err != nil {
		errs = append(errs, err)
	}
	if err := validateFilesAcceptTag(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateFilesMaxSizeTag(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "thumbnail"); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "list_preview"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateFilesAcceptTag(ctx ValidateContext) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed["accept"])
	if raw == "" {
		return nil
	}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.ContainsAny(item, " \t\n\r") {
			return fieldError(ctx, "widget tag %q contains invalid accept item %q", "accept", item)
		}
		if strings.HasPrefix(item, ".") && len(item) > 1 {
			continue
		}
		if strings.Contains(item, "/") {
			parts := strings.Split(item, "/")
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				continue
			}
		}
		return fieldError(ctx, "widget tag %q contains invalid accept item %q", "accept", item)
	}
	return nil
}

func validateFilesMaxSizeTag(ctx ValidateContext) error {
	raw := strings.TrimSpace(ctx.Field.WidgetParsed["max_size"])
	if raw == "" {
		return nil
	}
	upper := strings.ToUpper(raw)
	for _, unit := range []string{"KB", "MB", "GB", "B"} {
		if !strings.HasSuffix(upper, unit) {
			continue
		}
		number := strings.TrimSpace(raw[:len(raw)-len(unit)])
		if number == "" {
			return fieldError(ctx, "widget tag %q must be a positive size like 10MB, got %q", "max_size", raw)
		}
		value, err := strconv.ParseFloat(number, 64)
		if err != nil || value <= 0 {
			return fieldError(ctx, "widget tag %q must be a positive size like 10MB, got %q", "max_size", raw)
		}
		return nil
	}
	return fieldError(ctx, "widget tag %q must use unit B, KB, MB, or GB, got %q", "max_size", raw)
}
