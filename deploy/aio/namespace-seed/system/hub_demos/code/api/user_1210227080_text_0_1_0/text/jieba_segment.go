//<文件名>jieba_segment.go</文件名>

package text

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos-sdk/agent-app/runtime/python"
)

// JiebaSegmentReq 中文分词请求结构体
type JiebaSegmentReq struct {
	// 框架标签：widget:"type:text_area;placeholder:请输入待分词的中文文本..." - 文本输入框
	Text string `json:"text" widget:"name:待分词文本;type:text_area;placeholder:请输入待分词的中文文本..." validate:"required"`

	// 框架标签：widget:"type:select;options:精确模式,全模式,搜索引擎模式" - 分词模式
	Mode string `json:"mode" widget:"name:分词模式;type:select;options:精确模式,全模式,搜索引擎模式;render_default:精确模式"`

	// 框架标签：widget:"type:integer;placeholder:10" - 关键词数量
	TopK int `json:"top_k" widget:"name:关键词数量;type:integer;render_default:10;placeholder:请输入关键词数量"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否移除停用词
	RemoveStopwords bool `json:"remove_stopwords" widget:"name:移除停用词;type:switch;render_default:true"`
}

// KeywordInfo 关键词信息
type KeywordInfo struct {
	Word   string  `json:"word" widget:"name:关键词;type:input"`
	Weight float64 `json:"weight" widget:"name:权重;type:float"`
}

// WordFreqInfo 词频信息
type WordFreqInfo struct {
	Word  string `json:"word" widget:"name:词语;type:input"`
	Count int    `json:"count" widget:"name:频次;type:integer"`
}

// JiebaSegmentResp 中文分词响应结构体
type JiebaSegmentResp struct {
	// 分词结果列表
	Words []string `json:"words" widget:"name:分词结果;type:list;item_type:text"`

	// 关键词列表（带权重）
	Keywords []KeywordInfo `json:"keywords" widget:"name:关键词列表;type:table"`

	// 词频统计（Top 20）
	WordFreq []WordFreqInfo `json:"word_freq" widget:"name:词频统计;type:table"`

	// 统计信息
	Statistics string `json:"statistics" widget:"name:统计信息;type:text_area"`
}

// runJiebaOnText 对给定文本执行 jieba 分词与关键词提取，供 JiebaSegment 与 FileKeywordExtract 复用
func runJiebaOnText(ctx *app.Context, text string, mode string, topK int, removeStopwords bool) (*JiebaSegmentResp, error) {
	if topK <= 0 {
		topK = 10
	}
	if mode == "" {
		mode = "精确模式"
	}
	cutAll := false
	useHMM := true
	if mode == "全模式" {
		cutAll = true
		useHMM = false
	} else if mode == "搜索引擎模式" {
		cutAll = false
		useHMM = true
	}
	type pythonRequest struct {
		Text            string `json:"text"`
		CutAll          bool   `json:"cut_all"`
		UseHMM          bool   `json:"use_hmm"`
		RemoveStopwords bool   `json:"remove_stopwords"`
		TopK            int    `json:"top_k"`
	}
	executor := pythonRuntime.NewExecutor(buildJiebaSegmentCode()).
		WithRequest(pythonRequest{
			Text:            text,
			CutAll:          cutAll,
			UseHMM:          useHMM,
			RemoveStopwords: removeStopwords,
			TopK:            topK,
		}).
		WithTimeout(30 * time.Second)
	defer func() { _ = executor.Close() }()
	var result struct {
		Words      []string       `json:"words"`
		Keywords   []KeywordInfo  `json:"keywords"`
		WordFreq   []WordFreqInfo `json:"word_freq"`
		Statistics struct {
			TotalChars  int `json:"total_chars"`
			TotalWords  int `json:"total_words"`
			UniqueWords int `json:"unique_words"`
		} `json:"statistics"`
	}
	if err := executor.ExecuteJSON(ctx, &result); err != nil {
		return nil, err
	}
	statsText := fmt.Sprintf("总字符数: %d\n总词数: %d\n唯一词数: %d",
		result.Statistics.TotalChars,
		result.Statistics.TotalWords,
		result.Statistics.UniqueWords)
	return &JiebaSegmentResp{
		Words:      result.Words,
		Keywords:   result.Keywords,
		WordFreq:   result.WordFreq,
		Statistics: statsText,
	}, nil
}

// JiebaSegment 中文分词与关键词提取函数
//
// 错误处理说明：
// - 系统错误（Python 执行失败、系统异常等）：直接 return err，框架会记录日志并返回系统错误
// - 业务错误（参数验证失败等）：使用 resp.BizErrorf().Build()，返回给用户的业务错误提示
// 本函数主要涉及系统错误（Python 执行），业务错误由 ShouldBindValidate 处理
func JiebaSegment(ctx *app.Context, resp response.Response) error {
	var req JiebaSegmentReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}
	if req.Mode == "" {
		req.Mode = "精确模式"
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}
	res, err := runJiebaOnText(ctx, req.Text, req.Mode, req.TopK, req.RemoveStopwords)
	if err != nil {
		logger.Errorf(ctx, "[JiebaSegment] Python 执行失败: %v", err)
		return fmt.Errorf("执行中文分词失败: %w", err)
	}
	return resp.Form(res).Build()
}

// buildJiebaSegmentCode 构建中文分词的 Python 代码
func buildJiebaSegmentCode() string {
	code := `import jieba
import jieba.analyse

def kageos_entry(args, output_dir):
    text = args["text"]
    cut_all = args["cut_all"]
    use_hmm = args["use_hmm"]
    remove_stopwords = args["remove_stopwords"]
    top_k = args["top_k"]

    if cut_all:
        words = list(jieba.cut(text, cut_all=True))
    elif not use_hmm:
        words = list(jieba.cut(text, cut_all=False, HMM=False))
    else:
        words = list(jieba.cut_for_search(text))

    if remove_stopwords:
        words = [w for w in words if len(w.strip()) > 1]

    keywords = jieba.analyse.extract_tags(text, topK=top_k, withWeight=True)

    word_freq = {}
    for word in words:
        word = word.strip()
        if word:
            word_freq[word] = word_freq.get(word, 0) + 1

    sorted_word_freq = sorted(word_freq.items(), key=lambda x: x[1], reverse=True)[:20]

    return {
        "data": {
            "words": words,
            "keywords": [{"word": k, "weight": float(w)} for k, w in keywords],
            "word_freq": [{"word": w, "count": c} for w, c in sorted_word_freq],
            "statistics": {
                "total_chars": len(text),
                "total_words": len(words),
                "unique_words": len(word_freq)
            }
        }
    }`

	return code
}

// JiebaSegmentTemplate 中文分词配置
var JiebaSegmentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "中文分词与关键词提取",
		Desc:     `使用 jieba 进行中文分词和关键词提取。支持精确模式、全模式、搜索引擎模式三种分词方式，自动提取关键词（TF-IDF），提供词频统计。应用场景：文本分析、内容提取、搜索优化、关键词挖掘等。`,
		Tags:     []string{"自然语言处理", "中文分词", "关键词提取", "文本分析"},
		Request:  &JiebaSegmentReq{},
		Response: &JiebaSegmentResp{},
	},
}

func init() {
	// 注册Form函数 - 中文分词与关键词提取
	packageContext.POST("segment.form", JiebaSegment, JiebaSegmentTemplate)
}
