# 案例：NLP 工具（单 Form）

## 一、项目概要

- **类型**：单 Form，POST，无 Table。
- **路由**：jieba_segment.form（分词/词频）；路由组 `/form/nlp`。
- **适合参考**：无 files 或可选、text_area/select/number/switch、响应里 table、**pythonRuntime**（jieba）；须 **`defer executor.Close()`**；Go 与 Python **同机子进程**，非隔离远程环境。

---

## 二、PRD 要点（表格格式）

### 分词（jieba_segment.form，POST）

**请求**（表单字段五列：字段 | 类型 | 必填 | 默认值 | 说明）

| 字段       | 类型     | 必填 | 默认值 | 说明 |
|------------|----------|------|--------|------|
| 待分词文本 | 多行文本 | ✓   | —      | 中文文本 |
| 分词模式   | 下拉选择 | ✗   | 精确模式 | 精确模式/全模式/搜索引擎模式 |
| 关键词数量 | 数字输入 | ✗   | 10     | 个 |
| 移除停用词 | 开关     | ✗   | true   | — |

**响应**

| 字段       | 类型     | 说明 |
|------------|----------|------|
| 分词结果   | 多行文本 | 分词后的词语列表 |
| 关键词列表 | 表格     | 关键词、权重 |
| 词频统计   | 表格     | 词语、频次（Top 20） |

---

## 三、文件与路由

| 文件               | 说明     | 注册路由            |
|--------------------|----------|---------------------|
| jieba_segment.go   | 分词/词频 | POST jieba_segment.form |

---

## 四、说明

代码随本案例一起提供；read_doc 本案例路径（如 `/builtin/doc/case_catalog/form/nlp`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### jieba_segment.go

```go
//<文件名>jieba_segment.go</文件名>

package nlp

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	pythonRuntime "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/runtime/python"
)

// JiebaSegmentReq 中文分词请求结构体
type JiebaSegmentReq struct {
	// 框架标签：widget:"type:text_area;placeholder:请输入待分词的中文文本..." - 文本输入框
	Text string `json:"text" widget:"name:待分词文本;type:text_area;placeholder:请输入待分词的中文文本..." validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	Mode string `json:"mode" widget:"name:分词模式;type:select;options:精确模式,全模式,搜索引擎模式;options_colors:success,primary,info;default:精确模式"`

	// 框架标签：widget:"type:number;placeholder:10" - 关键词数量
	TopK int `json:"top_k" widget:"name:关键词数量;type:number;placeholder:10（默认10个）"`

	// 框架标签：widget:"type:switch;default:true" - 是否移除停用词
	RemoveStopwords bool `json:"remove_stopwords" widget:"name:移除停用词;type:switch;default:true"`
}

// KeywordInfo 关键词信息
type KeywordInfo struct {
	Word   string  `json:"word" widget:"name:关键词;type:input" permission:"read"`
	Weight float64 `json:"weight" widget:"name:权重;type:number" permission:"read"`
}

// WordFreqInfo 词频信息
type WordFreqInfo struct {
	Word  string `json:"word" widget:"name:词语;type:input" permission:"read"`
	Count int    `json:"count" widget:"name:频次;type:number" permission:"read"`
}

// JiebaSegmentResp 中文分词响应结构体
type JiebaSegmentResp struct {
	// 分词结果列表
	Words []string `json:"words" widget:"name:分词结果;type:text_area" permission:"read"`

	// 关键词列表（带权重）
	Keywords []KeywordInfo `json:"keywords" widget:"name:关键词列表;type:table" permission:"read"`

	// 词频统计（Top 20）
	WordFreq []WordFreqInfo `json:"word_freq" widget:"name:词频统计;type:table" permission:"read"`

	// 统计信息
	Statistics string `json:"statistics" widget:"name:统计信息;type:text_area" permission:"read"`
}

// JiebaSegment 中文分词与关键词提取入口（SDK 注册用）：解析请求 → 调 DoJiebaSegment → 写响应
func JiebaSegment(ctx *app.Context, resp response.Response) error {
	var req JiebaSegmentReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoJiebaSegment(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoJiebaSegment 中文分词与关键词提取业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoJiebaSegment(ctx *app.Context, req *JiebaSegmentReq) (*JiebaSegmentResp, error) {
	if req.Mode == "" {
		req.Mode = "精确模式"
	}
	if req.TopK <= 0 {
		req.TopK = 10
	}

	pythonCode := buildJiebaSegmentCode()

	cutAll := false
	useHMM := true
	if req.Mode == "全模式" {
		cutAll = true
		useHMM = false
	} else if req.Mode == "搜索引擎模式" {
		cutAll = false
		useHMM = true
	}

	type PythonRequest struct {
		Text            string `json:"text"`
		CutAll          bool   `json:"cut_all"`
		UseHMM          bool   `json:"use_hmm"`
		RemoveStopwords bool   `json:"remove_stopwords"`
		TopK            int    `json:"top_k"`
	}

	pythonReq := PythonRequest{
		Text:            req.Text,
		CutAll:          cutAll,
		UseHMM:          useHMM,
		RemoveStopwords: req.RemoveStopwords,
		TopK:            req.TopK,
	}

	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pythonReq).
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
		logger.Errorf(ctx, "[系统错误]-[DoJiebaSegment] Python 执行失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoJiebaSegment]： 执行中文分词失败, req: %+v, err: %w", req, err)
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

// buildJiebaSegmentCode 构建中文分词的 Python 代码
func buildJiebaSegmentCode() string {
	code := `import jieba
import jieba.analyse
import json

# 从请求中获取参数（自动注入到全局命名空间）
# text, cut_all, use_hmm, remove_stopwords, top_k 已自动注入

# 分词
if cut_all:
    words = list(jieba.cut(text, cut_all=True))
elif not use_hmm:
    words = list(jieba.cut(text, cut_all=False, HMM=False))
else:
    # 搜索引擎模式
    words = list(jieba.cut_for_search(text))

# 移除停用词（简单处理，移除单字符和空白）
if remove_stopwords:
    words = [w for w in words if len(w.strip()) > 1]

# 关键词提取（TF-IDF）
keywords = jieba.analyse.extract_tags(text, topK=top_k, withWeight=True)

# 词频统计
word_freq = {}
for word in words:
    word = word.strip()
    if word:
        word_freq[word] = word_freq.get(word, 0) + 1

# 按频次排序，取 Top 20
sorted_word_freq = sorted(word_freq.items(), key=lambda x: x[1], reverse=True)[:20]

# 构建结果
result = {
    "words": words,
    "keywords": [{"word": k, "weight": float(w)} for k, w in keywords],
    "word_freq": [{"word": w, "count": c} for w, c in sorted_word_freq],
    "statistics": {
        "total_chars": len(text),
        "total_words": len(words),
        "unique_words": len(word_freq)
    }
}

output_json(result)`

	return code
}

// JiebaSegmentTemplate 中文分词配置
var JiebaSegmentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "中文分词与关键词提取",
		Desc:     "使用 jieba 进行中文分词和关键词提取。支持精确模式、全模式、搜索引擎模式三种分词方式，自动提取关键词（TF-IDF），提供词频统计。应用场景：文本分析、内容提取、搜索优化、关键词挖掘等。",
		Tags:     []string{"自然语言处理", "中文分词", "关键词提取", "文本分析"},
		Request:  &JiebaSegmentReq{},
		Response: &JiebaSegmentResp{},
	},
}

func init() {
	// 注册Form函数 - 中文分词与关键词提取
	packageContext.POST("jieba_segment.form", JiebaSegment, JiebaSegmentTemplate)
}
```

