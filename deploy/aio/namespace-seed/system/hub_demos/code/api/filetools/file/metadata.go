package file

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ReadMetadataReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传文件;type:files;accept:image/*,video/*,.pdf,*/*;max_size:2000MB;max_count:50" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:文本,JSON;options_colors:409EFF,67C23A;render_default:文本" validate:"required,oneof=文本 JSON"`
}

type ReadMetadataResp struct {
	MetadataText string `json:"metadata_text" widget:"name:元数据;type:text_area"`
}

func ReadMetadata(ctx *app.Context, resp response.Response) error {
	var req ReadMetadataReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoReadMetadata(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoReadMetadata(ctx *app.Context, req *ReadMetadataReq) (*ReadMetadataResp, error) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		return nil, fmt.Errorf("未找到 exiftool，请确认运行环境已安装 ExifTool")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	args := []string{"-G1"}
	if req.OutputFormat == "JSON" {
		args = []string{"-json", "-G1"}
	}
	validCount := 0
	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[File/ReadMetadata] 文件 %s 无本地路径，跳过", filepath.Base(file))
			continue
		}
		args = append(args, file)
		validCount++
	}
	if validCount == 0 {
		return nil, fmt.Errorf("没有可读取的有效文件")
	}

	out, err := exec.Command("exiftool", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[File/ReadMetadata] 执行失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("读取元数据失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		text = "未读取到元数据。"
	}
	return &ReadMetadataResp{MetadataText: text}, nil
}

type StripMetadataReq struct {
	InputFiles string `json:"input_files" widget:"name:上传文件;type:files;accept:image/*,video/*,.pdf,*/*;max_size:2000MB;max_count:50" validate:"required"`
	Mode       string `json:"mode" widget:"name:清理模式;type:select;options:全部元数据,仅GPS位置;options_colors:F56C6C,E6A23C;render_default:全部元数据" validate:"required,oneof=全部元数据 仅GPS位置"`
}

type StripMetadataResp struct {
	OutputFiles string `json:"output_files" widget:"name:清理后的文件;type:files"`
	StripInfo   string `json:"strip_info" widget:"name:清理信息;type:text_area"`
}

func StripMetadata(ctx *app.Context, resp response.Response) error {
	var req StripMetadataReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoStripMetadata(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoStripMetadata(ctx *app.Context, req *StripMetadataReq) (*StripMetadataResp, error) {
	if _, err := exec.LookPath("exiftool"); err != nil {
		return nil, fmt.Errorf("未找到 exiftool，请确认运行环境已安装 ExifTool")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[File/StripMetadata] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		suffix := "_metadata_cleaned"
		if req.Mode == "仅GPS位置" {
			suffix = "_gps_cleaned"
		}
		ext := strings.TrimPrefix(filepath.Ext(file), ".")
		outputName := sanitizeOutputBaseName(filepath.Base(file), "file") + suffix
		if ext != "" {
			outputName += "." + ext
		}
		outputName = uniqueName(outputName, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		if err := copyFile(file, outputPath); err != nil {
			failCount++
			infos = append(infos, fmt.Sprintf("复制失败 %s: %v", filepath.Base(file), err))
			continue
		}

		out, err := exec.Command("exiftool", stripMetadataArgs(req.Mode, outputPath)...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[File/StripMetadata] 清理失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
		if text := strings.TrimSpace(string(out)); text != "" {
			infos = append(infos, text)
		}
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功清理的文件\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("元数据清理完成\n清理模式: %s\n成功: %d\n失败: %d", req.Mode, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &StripMetadataResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		StripInfo:   info,
	}, nil
}

func stripMetadataArgs(mode, outputPath string) []string {
	if mode == "仅GPS位置" {
		return []string{"-gps:all=", "-xmp:geotag=", "-overwrite_original", outputPath}
	}
	return []string{"-all=", "-overwrite_original", outputPath}
}

var ReadMetadataTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "读取文件元数据",
		Desc:     `读取图片、视频、PDF 等文件的 EXIF/IPTC/XMP/GPS/编码等元数据，可输出文本或 JSON。适合查看拍摄信息、定位信息、作者信息和文件来源。`,
		Tags:     []string{"文件", "元数据", "EXIF", "GPS", "ExifTool", "图片", "视频", "PDF"},
		Request:  &ReadMetadataReq{},
		Response: &ReadMetadataResp{},
	},
}

var StripMetadataTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "清理文件元数据",
		Desc:     `清理图片、视频、PDF 等文件中的元数据。默认删除全部元数据，也可仅删除 GPS 位置信息。会先复制输入文件到输出目录，再对副本执行 ExifTool，不会直接修改上传临时文件。`,
		Tags:     []string{"文件", "元数据", "隐私清理", "EXIF", "GPS", "ExifTool"},
		Request:  &StripMetadataReq{},
		Response: &StripMetadataResp{},
	},
}

func init() {
	packageContext.POST("read_metadata.form", ReadMetadata, ReadMetadataTemplate)
	packageContext.POST("strip_metadata.form", StripMetadata, StripMetadataTemplate)
}
