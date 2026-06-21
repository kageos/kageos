package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ListArchiveReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传压缩包;type:files;accept:.zip,.tar,.tgz,.tar.gz,.tar.xz,.txz,.tar.bz2,.tbz2,application/zip,application/x-tar,*/*;max_size:2000MB;max_count:20" validate:"required"`
	MaxEntries   int    `json:"max_entries" widget:"name:最多预览条目;type:integer;min:0;max:5000;render_default:300;placeholder:0 表示不限制" validate:"min=0,max=5000"`
	OutputReport bool   `json:"output_report" widget:"name:输出目录清单文件;type:switch;render_default:true"`
}

type ListArchiveResp struct {
	EntriesText string `json:"entries_text" widget:"name:压缩包内容;type:text_area"`
	OutputFile  string `json:"output_file" widget:"name:目录清单;type:files"`
	Summary     string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func ListArchive(ctx *app.Context, resp response.Response) error {
	var req ListArchiveReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoListArchive(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoListArchive(ctx *app.Context, req *ListArchiveReq) (*ListArchiveResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxEntries := req.MaxEntries
	if maxEntries < 0 {
		maxEntries = 300
	}

	var blocks []string
	var summaries []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		kind := archiveKind(path)
		var entries []string
		var total int
		var err error
		switch kind {
		case "zip":
			entries, total, err = listZipEntries(path, maxEntries)
		case "tar":
			entries, total, err = listTarEntries(path, maxEntries)
		default:
			err = fmt.Errorf("暂不支持该压缩包格式")
		}
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		text := strings.Join(entries, "\n")
		if maxEntries > 0 && total > len(entries) {
			text += fmt.Sprintf("\n... 还有 %d 个条目未显示", total-len(entries))
		}
		blocks = append(blocks, fmt.Sprintf("## %s\n格式: %s\n文件条目数: %d\n\n%s", filepath.Base(path), kind, total, text))
		summaries = append(summaries, fmt.Sprintf("成功 %s: %d 个文件条目", filepath.Base(path), total))
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的压缩包\n%s", strings.Join(summaries, "\n"))
	}

	content := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "archive_entries.txt")
		if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("写入目录清单失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &ListArchiveResp{
		EntriesText: content,
		OutputFile:  outputFile,
		Summary:     "压缩包目录读取完成\n" + strings.Join(summaries, "\n"),
	}, nil
}

var ListArchiveTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "查看压缩包内容",
		Desc:     `读取 ZIP、TAR、TAR.GZ、TGZ、TAR.XZ 等压缩包目录，不解压即可预览内部文件列表，并可输出目录清单文本。`,
		Tags:     []string{"压缩包", "ZIP", "TAR", "目录清单", "预览"},
		Request:  &ListArchiveReq{},
		Response: &ListArchiveResp{},
	},
}

func init() {
	packageContext.POST("list.form", ListArchive, ListArchiveTemplate)
}
