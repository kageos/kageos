// create_graph.go：根据 DOT 内容生成图片，路由 POST /graphviz/create_graph.form

package diagram

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type CreateGraphReq struct {
	DotContent   string `json:"dot_content" widget:"name:DOT 图描述;type:text_area;placeholder:digraph G { A -> B -> C -> A }" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:png,svg,pdf;render_default:png" validate:"required"`
	Layout       string `json:"layout" widget:"name:布局引擎;type:select;options:dot,neato,fdp,sfdp,circo,twopi;render_default:dot"`
	FileName     string `json:"file_name" widget:"name:输出文件名;type:input;render_default:graph"`
}

type CreateGraphResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	Message    string `json:"message" widget:"name:说明;type:text_area"`
}

func CreateGraph(ctx *app.Context, resp response.Response) error {
	var req CreateGraphReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCreateGraph(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCreateGraph(ctx *app.Context, req *CreateGraphReq) (*CreateGraphResp, error) {
	format := strings.TrimSpace(strings.ToLower(req.OutputFormat))
	if format == "" {
		format = "png"
	}
	layout := strings.TrimSpace(strings.ToLower(req.Layout))
	if layout == "" {
		layout = "dot"
	}
	baseName := strings.TrimSpace(req.FileName)
	if baseName == "" {
		baseName = "graph"
	}
	baseName = strings.TrimSuffix(baseName, "."+format)

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()

	dotPath := filepath.Join(outputDir, baseName+".dot")
	if err := os.WriteFile(dotPath, []byte(req.DotContent), 0644); err != nil {
		return nil, fmt.Errorf("写入 DOT 文件失败: %w", err)
	}
	defer os.Remove(dotPath)

	outPath := filepath.Join(outputDir, baseName+"."+format)
	cmd := exec.Command(layout, "-T"+format, dotPath, "-o", outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[graphviz/CreateGraph] 执行失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("Graphviz 生成失败: %v\n%s", err, string(out))
	}

	if _, err := os.Stat(outPath); err != nil {
		return nil, fmt.Errorf("未生成输出文件 %s: %w", outPath, err)
	}

	outputFiles := fs.ResponseFiles([]string{outPath})

	return &CreateGraphResp{
		OutputFile: outputFiles,
		Message:    fmt.Sprintf("已生成 %s（布局: %s，格式: %s）", baseName+"."+format, layout, format),
	}, nil
}

var CreateGraphTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "根据 DOT 描述生成图",
		Desc:     `直接输入 DOT 语言描述，自动生成流程图/架构图/关系图。支持 png/svg/pdf 输出，支持 dot/neato/fdp/circo/twopi 等布局引擎。示例：digraph G { 开始 -> 处理 -> 结束; }`,
		Tags:     []string{"Graphviz", "绘图", "流程图", "架构图", "DOT"},
		Request:  &CreateGraphReq{},
		Response: &CreateGraphResp{},
	},
}

func init() {
	packageContext.POST("dot.form", CreateGraph, CreateGraphTemplate)
}
