package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ProtectUnlockPDFReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	Operation     string `json:"operation" widget:"name:处理方式;type:select;options:加密PDF,解密PDF;options_colors:E6A23C,67C23A;render_default:加密PDF" validate:"required,oneof=加密PDF 解密PDF"`
	OpenPassword  string `json:"open_password" widget:"name:打开密码;type:input;placeholder:用户打开 PDF 时需要输入;desc:加密 PDF 时显示并生效" validate:"required_if=Operation 加密PDF"`
	OwnerPassword string `json:"owner_password" widget:"name:权限/所有者密码（可选）;type:input;placeholder:留空则与打开密码相同;desc:加密 PDF 时显示并生效" validate:"excluded_if=Operation 解密PDF"`
	InputPassword string `json:"input_password" widget:"name:原 PDF 密码;type:input;placeholder:用于解密已有加密 PDF;desc:解密 PDF 时显示并生效" validate:"required_if=Operation 解密PDF"`
}

type ProtectUnlockPDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:处理后的 PDF;type:files"`
	RunInfo     string `json:"run_info" widget:"name:处理信息;type:text_area"`
}

func ProtectUnlockPDF(ctx *app.Context, resp response.Response) error {
	var req ProtectUnlockPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoProtectUnlockPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoProtectUnlockPDF(ctx *app.Context, req *ProtectUnlockPDFReq) (*ProtectUnlockPDFResp, error) {
	if err := ensureQPDF(); err != nil {
		return nil, err
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
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		suffix := "_protected"
		if req.Operation == "解密PDF" {
			suffix = "_unlocked"
		}
		outputName := outputPDFName(filepath.Base(file), file, suffix, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args, err := buildProtectUnlockArgs(req, file, outputPath)
		if err != nil {
			return nil, err
		}
		out, err := exec.Command("qpdf", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PDF/ProtectUnlock] qpdf 失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功处理的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("PDF 密码处理完成\n处理方式: %s\n成功: %d\n失败: %d", req.Operation, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ProtectUnlockPDFResp{OutputFiles: fs.ResponseFiles(outputPaths), RunInfo: info}, nil
}

func buildProtectUnlockArgs(req *ProtectUnlockPDFReq, inputPath, outputPath string) ([]string, error) {
	switch req.Operation {
	case "加密PDF":
		openPassword := strings.TrimSpace(req.OpenPassword)
		if openPassword == "" {
			return nil, fmt.Errorf("打开密码不能为空")
		}
		ownerPassword := strings.TrimSpace(req.OwnerPassword)
		if ownerPassword == "" {
			ownerPassword = openPassword
		}
		return []string{"--encrypt", openPassword, ownerPassword, "256", "--", inputPath, outputPath}, nil
	case "解密PDF":
		password := strings.TrimSpace(req.InputPassword)
		if password == "" {
			return nil, fmt.Errorf("原 PDF 密码不能为空")
		}
		return []string{"--password=" + password, "--decrypt", inputPath, outputPath}, nil
	default:
		return nil, fmt.Errorf("不支持的处理方式: %s", req.Operation)
	}
}

var ProtectUnlockPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF 加密/解密",
		Desc:     `使用 qpdf 为 PDF 设置打开密码，或输入已有密码解密并输出无密码 PDF。适合合同、报价单、资料交付和受保护 PDF 的后续处理。`,
		Tags:     []string{"PDF", "加密", "解密", "密码", "qpdf", "保护"},
		Request:  &ProtectUnlockPDFReq{},
		Response: &ProtectUnlockPDFResp{},
	},
}

func init() {
	packageContext.POST("protect_unlock.form", ProtectUnlockPDF, ProtectUnlockPDFTemplate)
}
