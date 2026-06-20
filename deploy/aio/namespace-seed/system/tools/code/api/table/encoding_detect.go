package table

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	stdunicode "unicode"
	"unicode/utf8"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type EncodingDetectReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传 CSV/文本文件;type:files;accept:.csv,.tsv,.txt,.log,.md,text/*,*/*;max_size:500MB;max_count:50" validate:"required"`
	SampleBytes  int    `json:"sample_bytes" widget:"name:采样字节数;type:integer;min:1024;max:1048576;render_default:65536;placeholder:请输入采样字节数，例如 65536" validate:"min=0,max=1048576"`
	OutputReport bool   `json:"output_report" widget:"name:输出检测报告文件;type:switch;render_default:true"`
}

type EncodingDetectResp struct {
	ReportText string `json:"report_text" widget:"name:编码检测报告;type:text_area"`
	OutputFile string `json:"output_file" widget:"name:检测报告;type:files"`
	Summary    string `json:"summary" widget:"name:处理说明;type:text_area"`
}

type encodingCandidate struct {
	Name    string
	Decoder *encoding.Decoder
}

type encodingResult struct {
	Name       string
	Confidence string
	Reason     string
	Preview    string
	Score      float64
}

func EncodingDetect(ctx *app.Context, resp response.Response) error {
	var req EncodingDetectReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoEncodingDetect(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoEncodingDetect(ctx *app.Context, req *EncodingDetectReq) (*EncodingDetectResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	sampleBytes := req.SampleBytes
	if sampleBytes <= 0 {
		sampleBytes = 65536
	}
	if sampleBytes < 1024 {
		sampleBytes = 1024
	}
	if sampleBytes > 1048576 {
		sampleBytes = 1048576
	}

	var blocks []string
	var summaries []string
	for _, file := range inputFiles {
		if file == "" {
			continue
		}
		block, summary, err := detectOneEncoding(file, sampleBytes)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		blocks = append(blocks, block)
		summaries = append(summaries, summary)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功检测的文件\n%s", strings.Join(summaries, "\n"))
	}

	report := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "encoding_detect_report.md")
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return nil, fmt.Errorf("写入编码检测报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &EncodingDetectResp{
		ReportText: report,
		OutputFile: outputFile,
		Summary:    "编码检测完成\n" + strings.Join(summaries, "\n"),
	}, nil
}

func detectOneEncoding(path string, sampleBytes int) (string, string, error) {
	sample := readSample(path, sampleBytes)
	if len(sample) == 0 {
		return "", "", fmt.Errorf("文件为空或读取失败")
	}
	stat, _ := os.Stat(path)
	result := detectEncoding(sample)
	delimiter := delimiterLabel(detectDelimiter(sample))

	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(filepath.Base(path))
	b.WriteString("\n")
	if stat != nil {
		b.WriteString(fmt.Sprintf("- 文件大小: %d bytes\n", stat.Size()))
	}
	b.WriteString(fmt.Sprintf("- 采样字节数: %d\n", len(sample)))
	b.WriteString(fmt.Sprintf("- 推测编码: `%s`\n", result.Name))
	b.WriteString(fmt.Sprintf("- 置信度: %s\n", result.Confidence))
	b.WriteString(fmt.Sprintf("- 判断依据: %s\n", result.Reason))
	b.WriteString(fmt.Sprintf("- 可能分隔符: %s\n", delimiter))
	if !strings.EqualFold(result.Name, "UTF-8") && !strings.EqualFold(result.Name, "UTF-8-BOM") {
		b.WriteString(fmt.Sprintf("- 转 UTF-8 建议: `iconv -f %s -t UTF-8 input.csv > output.csv`\n", iconvEncodingName(result.Name)))
	}
	b.WriteString("\n### 文本预览\n")
	b.WriteString("```text\n")
	b.WriteString(result.Preview)
	b.WriteString("\n```\n")
	summary := fmt.Sprintf("成功 %s: %s（%s）", filepath.Base(path), result.Name, result.Confidence)
	return b.String(), summary, nil
}

func detectEncoding(sample []byte) encodingResult {
	if bytes.HasPrefix(sample, []byte{0xEF, 0xBB, 0xBF}) {
		text := strings.TrimPrefix(string(sample), "\ufeff")
		return encodingResult{Name: "UTF-8-BOM", Confidence: "高", Reason: "检测到 UTF-8 BOM", Preview: textPreview(text), Score: 100}
	}
	if bytes.HasPrefix(sample, []byte{0xFF, 0xFE}) {
		text := decodeWith(textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM).NewDecoder(), sample[2:])
		return encodingResult{Name: "UTF-16LE", Confidence: "高", Reason: "检测到 UTF-16LE BOM", Preview: textPreview(text), Score: 100}
	}
	if bytes.HasPrefix(sample, []byte{0xFE, 0xFF}) {
		text := decodeWith(textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM).NewDecoder(), sample[2:])
		return encodingResult{Name: "UTF-16BE", Confidence: "高", Reason: "检测到 UTF-16BE BOM", Preview: textPreview(text), Score: 100}
	}
	if utf8.Valid(sample) {
		text := string(sample)
		return encodingResult{Name: "UTF-8", Confidence: "高", Reason: "样本是合法 UTF-8 字节序列", Preview: textPreview(text), Score: 95}
	}

	candidates := []encodingCandidate{
		{Name: "GB18030", Decoder: simplifiedchinese.GB18030.NewDecoder()},
		{Name: "GBK", Decoder: simplifiedchinese.GBK.NewDecoder()},
		{Name: "Big5", Decoder: traditionalchinese.Big5.NewDecoder()},
		{Name: "UTF-16LE", Decoder: textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM).NewDecoder()},
		{Name: "UTF-16BE", Decoder: textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM).NewDecoder()},
		{Name: "Windows-1252", Decoder: charmap.Windows1252.NewDecoder()},
	}
	best := encodingResult{Name: "未知", Confidence: "低", Reason: "未找到可靠编码", Preview: textPreview(string(sample))}
	for _, candidate := range candidates {
		text := decodeWith(candidate.Decoder, sample)
		score, reason := scoreDecodedText(text)
		if score > best.Score {
			best = encodingResult{
				Name:    candidate.Name,
				Score:   score,
				Reason:  reason,
				Preview: textPreview(text),
			}
		}
	}
	switch {
	case best.Score >= 80:
		best.Confidence = "中高"
	case best.Score >= 60:
		best.Confidence = "中"
	default:
		best.Confidence = "低"
	}
	return best
}

func decodeWith(decoder *encoding.Decoder, sample []byte) string {
	out, _, err := transform.Bytes(decoder, sample)
	if err != nil {
		return string(out)
	}
	return string(out)
}

func scoreDecodedText(text string) (float64, string) {
	if text == "" {
		return -100, "解码结果为空"
	}
	total := 0
	replacement := 0
	control := 0
	printable := 0
	chinese := 0
	for _, r := range text {
		total++
		switch {
		case r == utf8.RuneError:
			replacement++
		case r == '\n' || r == '\r' || r == '\t':
			printable++
		case stdunicode.IsControl(r):
			control++
		default:
			printable++
		}
		if r >= 0x4E00 && r <= 0x9FFF {
			chinese++
		}
	}
	if total == 0 {
		return -100, "解码结果为空"
	}
	printableRatio := float64(printable) / float64(total)
	replacementRatio := float64(replacement) / float64(total)
	controlRatio := float64(control) / float64(total)
	chineseRatio := float64(chinese) / float64(total)
	score := printableRatio*80 - replacementRatio*140 - controlRatio*120 + chineseRatio*20
	reason := fmt.Sprintf("可打印字符 %.1f%%，替换字符 %.1f%%，控制字符 %.1f%%，中文字符 %.1f%%",
		printableRatio*100, replacementRatio*100, controlRatio*100, chineseRatio*100)
	return score, reason
}

func textPreview(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) > 3000 {
		return string(runes[:3000]) + "\n...（预览已截断）"
	}
	return text
}

func iconvEncodingName(name string) string {
	switch name {
	case "UTF-8-BOM":
		return "UTF-8"
	case "GBK":
		return "GBK"
	case "GB18030":
		return "GB18030"
	case "Big5":
		return "BIG5"
	case "UTF-16LE":
		return "UTF-16LE"
	case "UTF-16BE":
		return "UTF-16BE"
	case "Windows-1252":
		return "WINDOWS-1252"
	default:
		return name
	}
}

var EncodingDetectTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "检测文本/CSV编码",
		Desc:     `检测上传 CSV、TSV、TXT、日志等文本文件的可能编码，覆盖 UTF-8、UTF-8 BOM、GB18030/GBK、Big5、UTF-16 和 Windows-1252，并给出文本预览与转 UTF-8 建议。适合导入表格前排查中文乱码。`,
		Tags:     []string{"编码检测", "CSV", "乱码", "GBK", "UTF-8", "文本"},
		Request:  &EncodingDetectReq{},
		Response: &EncodingDetectResp{},
	},
}

func init() {
	packageContext.POST("encoding_detect.form", EncodingDetect, EncodingDetectTemplate)
}
