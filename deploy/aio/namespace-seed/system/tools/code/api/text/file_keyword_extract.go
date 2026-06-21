// file_keyword_extract.go：根据上传的文本文件提取关键词，路由 POST /nlp/file_keyword_extract.form
// 读取文件内容（UTF-8）后使用 jieba 分词与 TF-IDF 关键词提取，与「中文分词与关键词提取」同一套逻辑。

package text

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// FileKeywordExtractReq 根据文件提取关键词请求
type FileKeywordExtractReq struct {
	InputFiles      string `json:"input_files" widget:"name:上传文本文件;type:files;accept:.txt,.md,.markdown,.csv,.json,.xml,.html,.htm,text/*;max_size:10MB;max_count:20" validate:"required"`
	Mode            string `json:"mode" widget:"name:分词模式;type:select;options:精确模式,全模式,搜索引擎模式;render_default:精确模式"`
	TopK            int    `json:"top_k" widget:"name:关键词数量;type:integer;render_default:10;placeholder:请输入关键词数量"`
	RemoveStopwords bool   `json:"remove_stopwords" widget:"name:移除停用词;type:switch;render_default:true"`
}

// FileKeywordExtractResp 与 JiebaSegmentResp 一致，Statistics 中会追加来源说明
type FileKeywordExtractResp struct {
	Words      []string       `json:"words" widget:"name:分词结果;type:list;item_type:text"`
	Keywords   []KeywordInfo  `json:"keywords" widget:"name:关键词列表;type:table"`
	WordFreq   []WordFreqInfo `json:"word_freq" widget:"name:词频统计;type:table"`
	Statistics string         `json:"statistics" widget:"name:统计信息;type:text_area"`
}

// FileKeywordExtract 根据上传文件提取关键词
func FileKeywordExtract(ctx *app.Context, resp response.Response) error {
	var req FileKeywordExtractReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoFileKeywordExtract(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoFileKeywordExtract 读取文件内容（UTF-8）后调用 jieba 提取关键词
func DoFileKeywordExtract(ctx *app.Context, req *FileKeywordExtractReq) (*FileKeywordExtractResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var parts []string
	var readNames []string
	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[FileKeywordExtract] 文件 %s 无本地路径，跳过", filepath.Base(file))
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			logger.Warnf(ctx, "[FileKeywordExtract] 读取失败 %s: %v", filepath.Base(file), err)
			continue
		}
		// 按 UTF-8 解析；若含 BOM 则去掉
		text := strings.TrimSpace(string(raw))
		if strings.HasPrefix(text, "\xef\xbb\xbf") {
			text = strings.TrimSpace(text[3:])
		}
		if text == "" {
			continue
		}
		parts = append(parts, text)
		readNames = append(readNames, filepath.Base(file))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("未能从任何文件中读取到有效文本，请上传 UTF-8 编码的文本文件")
	}

	combined := strings.Join(parts, "\n\n")
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "精确模式"
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	jiebaResp, err := runJiebaOnText(ctx, combined, mode, topK, req.RemoveStopwords)
	if err != nil {
		logger.Errorf(ctx, "[FileKeywordExtract] jieba 执行失败: %v", err)
		return nil, fmt.Errorf("执行关键词提取失败: %w", err)
	}

	sourceInfo := fmt.Sprintf("来源：从 %d 个文件提取（%s）", len(readNames), strings.Join(readNames, "、"))
	stats := jiebaResp.Statistics
	if stats != "" {
		stats = stats + "\n" + sourceInfo
	} else {
		stats = sourceInfo
	}

	return &FileKeywordExtractResp{
		Words:      jiebaResp.Words,
		Keywords:   jiebaResp.Keywords,
		WordFreq:   jiebaResp.WordFreq,
		Statistics: stats,
	}, nil
}

var FileKeywordExtractTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "根据文件提取关键词",
		Desc:     `上传文本文件（如 .txt、.md、.csv 等），使用 jieba 进行中文分词并提取关键词（TF-IDF）、词频统计。支持多文件合并分析。文件需为 UTF-8 编码。`,
		Tags:     []string{"自然语言处理", "关键词提取", "文本分析", "文件"},
		Request:  &FileKeywordExtractReq{},
		Response: &FileKeywordExtractResp{},
	},
}

func init() {
	packageContext.POST("keyword_extract.form", FileKeywordExtract, FileKeywordExtractTemplate)
}
