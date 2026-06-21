package text

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos-sdk/agent-app/runtime/python"
)

type WordCloudReq struct {
	Text            string `json:"text" widget:"name:文本内容;type:text_area;placeholder:可直接粘贴评论、会议纪要、调研反馈等文本"`
	InputFiles      string `json:"input_files" widget:"name:上传文本文件;type:files;accept:.txt,.md,.markdown,.csv,.json,.html,.htm,text/*,*/*;max_size:50MB;max_count:20"`
	OutputFileName  string `json:"output_file_name" widget:"name:输出文件名;type:input;render_default:wordcloud;placeholder:不用写 .png"`
	UseJieba        bool   `json:"use_jieba" widget:"name:中文分词;type:switch;render_default:true"`
	RemoveStopwords bool   `json:"remove_stopwords" widget:"name:移除常见停用词;type:switch;render_default:true"`
	Width           int    `json:"width" widget:"name:图片宽度;type:integer;min:400;max:4000;step:100;unit:px;render_default:1600" validate:"min=0,max=4000"`
	Height          int    `json:"height" widget:"name:图片高度;type:integer;min:300;max:3000;step:100;unit:px;render_default:1000" validate:"min=0,max=3000"`
	MaxWords        int    `json:"max_words" widget:"name:最多词数;type:integer;min:20;max:1000;render_default:200" validate:"min=0,max=1000"`
	Background      string `json:"background" widget:"name:背景色;type:select;options:白色,透明,深色;options_colors:909399,909399,409EFF;render_default:白色" validate:"required,oneof=白色 透明 深色"`
	ColorMap        string `json:"color_map" widget:"name:配色;type:select;options:viridis,plasma,inferno,magma,Set2,tab10;options_colors:409EFF,E6A23C,F56C6C,909399,67C23A,909399;render_default:viridis" validate:"required,oneof=viridis plasma inferno magma Set2 tab10"`
}

type WordCloudResp struct {
	OutputFile string `json:"output_file" widget:"name:词云图片;type:files"`
	Summary    string `json:"summary" widget:"name:生成信息;type:text_area"`
}

func WordCloud(ctx *app.Context, resp response.Response) error {
	var req WordCloudReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoWordCloud(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoWordCloud(ctx *app.Context, req *WordCloudReq) (*WordCloudResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	var parts []string
	var sources []string
	if strings.TrimSpace(req.Text) != "" {
		parts = append(parts, req.Text)
		sources = append(sources, "直接输入文本")
	}
	for _, file := range inputFiles {
		if file == "" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			logger.Warnf(ctx, "[Text/WordCloud] 读取失败 %s: %v", filepath.Base(file), err)
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(string(data), "\ufeff"))
		if text == "" {
			continue
		}
		parts = append(parts, text)
		sources = append(sources, filepath.Base(file))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("请填写文本内容，或上传 UTF-8 文本文件")
	}

	width := req.Width
	if width <= 0 {
		width = 1600
	}
	height := req.Height
	if height <= 0 {
		height = 1000
	}
	maxWords := req.MaxWords
	if maxWords <= 0 {
		maxWords = 200
	}
	outputDir := fs.GetTraceOutputDir()
	baseName := sanitizeTextOutputName(strings.TrimSuffix(strings.TrimSpace(req.OutputFileName), filepath.Ext(req.OutputFileName)), "wordcloud")
	outputPath := filepath.Join(outputDir, baseName+".png")

	executor := pythonRuntime.NewExecutor(wordCloudPythonCode()).
		WithRequest(map[string]interface{}{
			"text":             strings.Join(parts, "\n\n"),
			"output_path":      outputPath,
			"use_jieba":        req.UseJieba,
			"remove_stopwords": req.RemoveStopwords,
			"width":            width,
			"height":           height,
			"max_words":        maxWords,
			"background":       req.Background,
			"color_map":        req.ColorMap,
		}).
		WithOutputDir(outputDir).
		WithTimeout(45 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		WordCount  int    `json:"word_count"`
		FontPath   string `json:"font_path"`
		OutputName string `json:"output_name"`
	}
	if err := executor.ExecuteJSON(ctx, &result); err != nil {
		return nil, fmt.Errorf("生成词云失败: %w", err)
	}

	return &WordCloudResp{
		OutputFile: fs.ResponseFiles([]string{outputPath}),
		Summary: fmt.Sprintf("词云生成完成\n输出文件: %s\n图片尺寸: %dx%d\n最多词数: %d\n中文分词: %t\n移除停用词: %t\n词条数: %d\n字体: %s\n来源: %s",
			filepath.Base(outputPath), width, height, maxWords, req.UseJieba, req.RemoveStopwords, result.WordCount, result.FontPath, strings.Join(sources, "、")),
	}, nil
}

func sanitizeTextOutputName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}

func wordCloudPythonCode() string {
	return `import os
import re
import jieba
from wordcloud import WordCloud

STOPWORDS = {
    "的", "了", "和", "是", "在", "就", "都", "而", "及", "与", "着", "或", "一个", "没有", "我们", "你们", "他们",
    "这个", "那个", "这些", "那些", "进行", "以及", "可以", "如果", "因为", "所以", "但是", "然后", "已经", "需要",
    "the", "and", "is", "to", "of", "in", "for", "a", "an", "on", "with", "as", "by", "or", "from", "this", "that"
}
FONT_CANDIDATES = [
    "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
    "/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
    "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.otf",
    "/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
    "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
]

def find_font():
    for path in FONT_CANDIDATES:
        if os.path.exists(path):
            return path
    return None

def normalize_text(text):
    text = re.sub(r"https?://\S+", " ", text)
    text = re.sub(r"[\r\n\t]+", " ", text)
    return text.strip()

def kageos_entry(args, output_dir):
    text = normalize_text(args["text"])
    if not text:
        raise ValueError("文本内容为空")
    use_jieba = bool(args.get("use_jieba", True))
    remove_stopwords = bool(args.get("remove_stopwords", True))
    width = int(args.get("width") or 1600)
    height = int(args.get("height") or 1000)
    max_words = int(args.get("max_words") or 200)
    background_label = args.get("background") or "白色"
    color_map = args.get("color_map") or "viridis"
    output_path = args["output_path"]

    if use_jieba:
        words = [w.strip() for w in jieba.cut(text) if w.strip()]
    else:
        words = [w.strip() for w in re.split(r"\s+", text) if w.strip()]
    if remove_stopwords:
        words = [w for w in words if w not in STOPWORDS and len(w) > 1]
    if not words:
        raise ValueError("分词后没有可用于词云的词")

    background_color = "white"
    mode = "RGB"
    if background_label == "透明":
        background_color = None
        mode = "RGBA"
    elif background_label == "深色":
        background_color = "#111827"

    font_path = find_font()
    wc = WordCloud(
        font_path=font_path,
        width=width,
        height=height,
        max_words=max_words,
        background_color=background_color,
        mode=mode,
        colormap=color_map,
        collocations=False,
        random_state=42,
    )
    wc.generate(" ".join(words))
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    wc.to_file(output_path)
    return {
        "data": {
            "word_count": len(words),
            "font_path": font_path or "默认字体",
            "output_name": os.path.basename(output_path),
        }
    }`
}

var WordCloudTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "生成词云图",
		Desc:     `根据粘贴文本或上传文本文件生成 PNG 词云图，支持中文 jieba 分词、停用词过滤、尺寸、背景和配色设置。适合评论反馈、会议纪要、调研结果和报告配图。`,
		Tags:     []string{"词云", "文本分析", "可视化", "jieba", "wordcloud", "报告配图"},
		Request:  &WordCloudReq{},
		Response: &WordCloudResp{},
	},
}

func init() {
	packageContext.POST("wordcloud.form", WordCloud, WordCloudTemplate)
}
