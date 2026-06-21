package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type CreateZipReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传要打包的文件;type:files;accept:*/*;max_size:2000MB;max_count:200" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出 ZIP 文件名;type:input;render_default:workspace_files;placeholder:不用写 .zip"`
}

type CreateZipResp struct {
	OutputFile string `json:"output_file" widget:"name:ZIP 压缩包;type:files"`
	ZipInfo    string `json:"zip_info" widget:"name:打包信息;type:text_area"`
}

func CreateZip(ctx *app.Context, resp response.Response) error {
	var req CreateZipReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCreateZip(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCreateZip(ctx *app.Context, req *CreateZipReq) (*CreateZipResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	baseName := sanitizeArchiveName(strings.TrimSuffix(strings.TrimSpace(req.OutputFileName), ".zip"), "workspace_files")
	outputPath := filepath.Join(outputDir, baseName+".zip")

	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("创建 ZIP 文件失败: %w", err)
	}
	defer outFile.Close()

	writer := zip.NewWriter(outFile)
	seenNames := make(map[string]int)
	var infos []string
	addedCount := 0
	var totalBytes int64
	for _, file := range inputFiles {
		if file == "" {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			logger.Warnf(ctx, "[Archive/CreateZip] 读取文件失败 %s: %v", filepath.Base(file), err)
			infos = append(infos, fmt.Sprintf("跳过 %s: %v", filepath.Base(file), err))
			continue
		}
		if !info.Mode().IsRegular() {
			infos = append(infos, fmt.Sprintf("跳过 %s: 不是普通文件", filepath.Base(file)))
			continue
		}
		entryName := uniqueArchiveName(sanitizeArchiveName(filepath.Base(file), "file"), seenNames)
		if err := addFileToZip(writer, file, entryName); err != nil {
			logger.Warnf(ctx, "[Archive/CreateZip] 添加文件失败 %s: %v", filepath.Base(file), err)
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		addedCount++
		totalBytes += info.Size()
		infos = append(infos, fmt.Sprintf("成功添加 %s -> %s", filepath.Base(file), entryName))
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("写入 ZIP 文件失败: %w", err)
	}
	if addedCount == 0 {
		_ = os.Remove(outputPath)
		return nil, fmt.Errorf("没有成功打包的文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})
	zipInfo := fmt.Sprintf("ZIP 打包完成\n输出文件: %s\n文件数: %d\n原始总大小: %.2f MB", filepath.Base(outputPath), addedCount, float64(totalBytes)/(1024*1024))
	if len(infos) > 0 {
		zipInfo += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &CreateZipResp{OutputFile: outputFiles, ZipInfo: zipInfo}, nil
}

func addFileToZip(writer *zip.Writer, filePath, entryName string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	input, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer input.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = entryName
	header.Method = zip.Deflate
	entryWriter, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(entryWriter, input)
	return err
}

var CreateZipTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "打包文件为 ZIP",
		Desc:     `把多个工作台文件打包成一个 ZIP 压缩包，适合导出多张图片、多份文档、批量处理结果或交付附件。`,
		Tags:     []string{"压缩包", "ZIP", "打包", "文件", "交付"},
		Request:  &CreateZipReq{},
		Response: &CreateZipResp{},
	},
}

func init() {
	packageContext.POST("create_zip.form", CreateZip, CreateZipTemplate)
}
