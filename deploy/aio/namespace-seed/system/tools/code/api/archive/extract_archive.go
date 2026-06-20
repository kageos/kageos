package archive

import (
	"archive/zip"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ExtractArchiveReq struct {
	InputFiles string `json:"input_files" widget:"name:上传压缩包;type:files;accept:.zip,.tar,.tgz,.tar.gz,.tar.xz,.txz,.tar.bz2,.tbz2,application/zip,application/x-tar,*/*;max_size:2000MB;max_count:20" validate:"required"`
	MaxFiles   int    `json:"max_files" widget:"name:最多返回文件数;type:integer;min:1;max:500;render_default:100" validate:"min=0,max=500"`
}

type ExtractArchiveResp struct {
	OutputFiles string `json:"output_files" widget:"name:解出的文件;type:files"`
	ExtractInfo string `json:"extract_info" widget:"name:解包信息;type:text_area"`
}

func ExtractArchive(ctx *app.Context, resp response.Response) error {
	var req ExtractArchiveReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExtractArchive(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoExtractArchive(ctx *app.Context, req *ExtractArchiveReq) (*ExtractArchiveResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 100
	}
	if maxFiles > 500 {
		maxFiles = 500
	}

	outputDir := fs.GetTraceOutputDir()
	seen := make(map[string]int)
	var outputPaths []string
	var infos []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		if len(outputPaths) >= maxFiles {
			infos = append(infos, fmt.Sprintf("跳过 %s: 已达到返回上限 %d", filepath.Base(path), maxFiles))
			continue
		}
		kind := archiveKind(path)
		var added []string
		var err error
		switch kind {
		case "zip":
			added, err = extractZipArchive(path, outputDir, seen, maxFiles-len(outputPaths))
		case "tar":
			added, err = extractTarArchive(path, outputDir, seen, maxFiles-len(outputPaths))
		default:
			err = fmt.Errorf("暂不支持该压缩包格式")
		}
		if err != nil {
			logger.Warnf(ctx, "[Archive/ExtractArchive] 解包失败 %s: %v", filepath.Base(path), err)
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		outputPaths = append(outputPaths, added...)
		infos = append(infos, fmt.Sprintf("成功 %s: 返回 %d 个文件", filepath.Base(path), len(added)))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功解出的文件\n%s", strings.Join(infos, "\n"))
	}
	if len(outputPaths) >= maxFiles {
		infos = append(infos, fmt.Sprintf("注意：已达到最多返回文件数 %d，更多文件未返回到工作台", maxFiles))
	}
	return &ExtractArchiveResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		ExtractInfo: "压缩包解包完成\n" + strings.Join(infos, "\n"),
	}, nil
}

func extractZipArchive(path, outputDir string, seen map[string]int, limit int) ([]string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	prefix := archiveBaseName(path)
	var outputPaths []string
	for _, entry := range reader.File {
		if len(outputPaths) >= limit {
			break
		}
		if entry.FileInfo().IsDir() || strings.HasPrefix(entry.Name, "__MACOSX/") || strings.HasSuffix(strings.ToLower(entry.Name), ".ds_store") {
			continue
		}
		entryReader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		outputName := flattenEntryName(prefix, entry.Name, seen)
		outputPath := filepath.Join(outputDir, outputName)
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			entryReader.Close()
			return nil, err
		}
		outputFile, err := os.Create(outputPath)
		if err != nil {
			entryReader.Close()
			return nil, err
		}
		_, copyErr := outputFile.ReadFrom(entryReader)
		closeErr := outputFile.Close()
		entryReader.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		outputPaths = append(outputPaths, outputPath)
	}
	return outputPaths, nil
}

func extractTarArchive(path, outputDir string, seen map[string]int, limit int) ([]string, error) {
	if err := ensureTar(); err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp(outputDir, "archive_extract_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	out, err := exec.Command("tar", "-xf", path, "-C", tempDir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tar 解包失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	prefix := archiveBaseName(path)
	var outputPaths []string
	err = filepath.WalkDir(tempDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if len(outputPaths) >= limit || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(tempDir, current)
		if err != nil {
			return err
		}
		outputName := flattenEntryName(prefix, rel, seen)
		outputPath := filepath.Join(outputDir, outputName)
		if err := copyRegularFile(current, outputPath); err != nil {
			return err
		}
		outputPaths = append(outputPaths, outputPath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return outputPaths, nil
}

var ExtractArchiveTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "解出通用压缩包",
		Desc:     `安全平铺解出 ZIP、TAR、TAR.GZ、TGZ、TAR.XZ 等压缩包中的文件，自动跳过目录和常见系统垃圾文件，并限制返回文件数量，适合处理用户上传资料包。`,
		Tags:     []string{"压缩包", "解包", "ZIP", "TAR", "资料包"},
		Request:  &ExtractArchiveReq{},
		Response: &ExtractArchiveResp{},
	},
}

func init() {
	packageContext.POST("extract.form", ExtractArchive, ExtractArchiveTemplate)
}
