package file

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ChecksumReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传文件;type:files;accept:*/*;max_size:2000MB;max_count:100" validate:"required"`
	Algorithm    string `json:"algorithm" widget:"name:哈希算法;type:select;options:SHA256,MD5,SHA1,全部;options_colors:409EFF,E6A23C,909399,67C23A;render_default:SHA256" validate:"required,oneof=SHA256 MD5 SHA1 全部"`
	OutputReport bool   `json:"output_report" widget:"name:输出校验报告文件;type:switch;render_default:true"`
}

type ChecksumResp struct {
	ChecksumText string `json:"checksum_text" widget:"name:校验结果;type:text_area"`
	OutputFile   string `json:"output_file" widget:"name:校验报告;type:files"`
	Summary      string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func Checksum(ctx *app.Context, resp response.Response) error {
	var req ChecksumReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoChecksum(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoChecksum(ctx *app.Context, req *ChecksumReq) (*ChecksumResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	algorithms := checksumAlgorithms(req.Algorithm)
	var lines []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		stat, _ := os.Stat(path)
		lines = append(lines, fmt.Sprintf("## %s", filepath.Base(path)))
		if stat != nil {
			lines = append(lines, fmt.Sprintf("- 大小: %s (%d bytes)", humanBytes(stat.Size()), stat.Size()))
		}
		for _, alg := range algorithms {
			sum, err := computeChecksum(path, alg)
			if err != nil {
				lines = append(lines, fmt.Sprintf("- %s: 计算失败: %v", alg, err))
			} else {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", alg, sum))
			}
		}
		lines = append(lines, "")
	}
	text := strings.TrimSpace(strings.Join(lines, "\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "checksums.txt")
		if err := os.WriteFile(outputPath, []byte(text), 0644); err != nil {
			return nil, fmt.Errorf("写入校验报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &ChecksumResp{
		ChecksumText: text,
		OutputFile:   outputFile,
		Summary:      fmt.Sprintf("文件哈希校验完成\n文件数: %d\n算法: %s", len(inputFiles), strings.Join(algorithms, ", ")),
	}, nil
}

func checksumAlgorithms(option string) []string {
	switch strings.TrimSpace(option) {
	case "MD5":
		return []string{"MD5"}
	case "SHA1":
		return []string{"SHA1"}
	case "全部":
		return []string{"SHA256", "MD5", "SHA1"}
	default:
		return []string{"SHA256"}
	}
}

func computeChecksum(path string, algorithm string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var h hash.Hash
	switch algorithm {
	case "MD5":
		h = md5.New()
	case "SHA1":
		h = sha1.New()
	default:
		h = sha256.New()
	}
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var ChecksumTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "文件哈希校验",
		Desc:     `计算上传文件的 SHA256、MD5、SHA1 哈希，支持多文件批量输出校验报告。适合文件去重、交付验收、下载完整性校验和数据归档。`,
		Tags:     []string{"文件", "哈希", "校验", "SHA256", "MD5", "SHA1", "去重"},
		Request:  &ChecksumReq{},
		Response: &ChecksumResp{},
	},
}

func init() {
	packageContext.POST("checksum.form", Checksum, ChecksumTemplate)
}
