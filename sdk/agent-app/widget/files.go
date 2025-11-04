package widget

import "github.com/ai-agent-os/ai-agent-os/pkg/convert"

type Files struct {
	// Accept 文件类型限制，支持多种格式（逗号分隔）：
	// 1. 扩展名：.pdf,.doc,.docx,.jpg,.png
	// 2. MIME类型：application/pdf,image/jpeg
	// 3. MIME通配符：image/*,video/*,audio/*
	// 4. 混合使用：.pdf,image/*,video/*,application/zip
	// 示例：accept:.pdf,.doc,.docx,image/*,video/*
	// 为空则不限制类型
	Accept string `json:"accept"`

	// MaxSize 单个文件最大大小，支持单位：B, KB, MB, GB
	// 示例：max_size:10MB, max_size:1024KB, max_size:1GB
	// 为空则使用系统默认值
	MaxSize string `json:"max_size"`

	// MaxCount 最大上传文件数量，默认为 5
	// 示例：max_count:10
	MaxCount int `json:"max_count"`
}

func (i *Files) Config() interface{} {
	return i
}

func (i *Files) Type() string {
	return TypeFiles
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

	return files
}
