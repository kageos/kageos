// create_flowchart.go：根据 JSON 结构化数据自动生成流程图，路由 POST /graphviz/create_flowchart.form

package diagram

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// GraphNode 节点定义
type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Shape string `json:"shape,omitempty"` // box, ellipse, diamond, circle, record, plaintext, ...
	Color string `json:"color,omitempty"`
	Style string `json:"style,omitempty"` // filled, dashed, bold, ...
}

// GraphEdge 边定义
type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Color string `json:"color,omitempty"`
	Style string `json:"style,omitempty"` // dashed, bold, dotted, ...
}

// GraphData JSON 图数据
type GraphData struct {
	Title    string      `json:"title,omitempty"`
	Directed bool        `json:"directed"` // true=有向图(digraph)，false=无向图(graph)
	Nodes    []GraphNode `json:"nodes"`
	Edges    []GraphEdge `json:"edges"`
}

type CreateFlowchartReq struct {
	GraphJSON    string `json:"graph_json" widget:"name:图数据(JSON);type:text_area;placeholder:{\"title\":\"用户注册流程\",\"directed\":true,\"nodes\":[{\"id\":\"start\",\"label\":\"开始\",\"shape\":\"ellipse\"},{\"id\":\"input\",\"label\":\"输入信息\",\"shape\":\"box\"},{\"id\":\"check\",\"label\":\"校验?\",\"shape\":\"diamond\"},{\"id\":\"ok\",\"label\":\"注册成功\",\"shape\":\"ellipse\"}],\"edges\":[{\"from\":\"start\",\"to\":\"input\"},{\"from\":\"input\",\"to\":\"check\"},{\"from\":\"check\",\"to\":\"ok\",\"label\":\"通过\"},{\"from\":\"check\",\"to\":\"input\",\"label\":\"失败\"}]}" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:png,svg,pdf;render_default:png" validate:"required"`
	Layout       string `json:"layout" widget:"name:布局引擎;type:select;options:dot,neato,fdp,sfdp,circo,twopi;render_default:dot"`
	FileName     string `json:"file_name" widget:"name:输出文件名;type:input;render_default:flowchart"`
}

type CreateFlowchartResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	DotSource  string `json:"dot_source" widget:"name:生成的 DOT 源码;type:text_area"`
	Message    string `json:"message" widget:"name:说明;type:text_area"`
}

func CreateFlowchart(ctx *app.Context, resp response.Response) error {
	var req CreateFlowchartReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCreateFlowchart(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCreateFlowchart(ctx *app.Context, req *CreateFlowchartReq) (*CreateFlowchartResp, error) {
	var data GraphData
	if err := json.Unmarshal([]byte(req.GraphJSON), &data); err != nil {
		return nil, fmt.Errorf("图数据不是合法 JSON: %w", err)
	}
	if len(data.Nodes) == 0 && len(data.Edges) == 0 {
		return nil, fmt.Errorf("图数据不能为空（至少需要节点或边）")
	}

	dotSrc := buildDOT(&data)

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
		baseName = "flowchart"
	}
	baseName = strings.TrimSuffix(baseName, "."+format)

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	dotPath := filepath.Join(outputDir, baseName+".dot")
	if err := os.WriteFile(dotPath, []byte(dotSrc), 0644); err != nil {
		return nil, fmt.Errorf("写入 DOT 文件失败: %w", err)
	}
	defer os.Remove(dotPath)

	outPath := filepath.Join(outputDir, baseName+"."+format)
	cmd := exec.Command(layout, "-T"+format, dotPath, "-o", outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[graphviz/CreateFlowchart] 执行失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("Graphviz 生成失败: %v\n%s", err, string(out))
	}

	if _, err := os.Stat(outPath); err != nil {
		return nil, fmt.Errorf("未生成输出文件 %s: %w", outPath, err)
	}

	outputFiles := fs.ResponseFiles([]string{outPath})

	return &CreateFlowchartResp{
		OutputFile: outputFiles,
		DotSource:  dotSrc,
		Message:    fmt.Sprintf("已生成 %s（%d 节点, %d 边, 布局: %s）", baseName+"."+format, len(data.Nodes), len(data.Edges), layout),
	}, nil
}

// buildDOT 将结构化 GraphData 转为 DOT 语言
func buildDOT(data *GraphData) string {
	var b strings.Builder

	graphType := "digraph"
	edgeOp := " -> "
	if !data.Directed {
		graphType = "graph"
		edgeOp = " -- "
	}

	title := "G"
	if data.Title != "" {
		title = sanitizeDotID(data.Title)
	}

	b.WriteString(fmt.Sprintf("%s %s {\n", graphType, title))
	b.WriteString("    rankdir=TB;\n")
	b.WriteString("    node [fontname=\"Noto Sans CJK SC\"];\n")
	b.WriteString("    edge [fontname=\"Noto Sans CJK SC\"];\n")

	for _, n := range data.Nodes {
		if n.ID == "" {
			continue
		}
		var attrs []string
		label := n.Label
		if label == "" {
			label = n.ID
		}
		attrs = append(attrs, fmt.Sprintf("label=%q", label))
		if n.Shape != "" {
			attrs = append(attrs, fmt.Sprintf("shape=%s", n.Shape))
		}
		if n.Color != "" {
			attrs = append(attrs, fmt.Sprintf("color=%q", n.Color))
			if n.Style == "" {
				attrs = append(attrs, "style=filled", fmt.Sprintf("fillcolor=%q", n.Color))
			}
		}
		if n.Style != "" {
			attrs = append(attrs, fmt.Sprintf("style=%q", n.Style))
		}
		b.WriteString(fmt.Sprintf("    %s [%s];\n", sanitizeDotID(n.ID), strings.Join(attrs, ", ")))
	}

	for _, e := range data.Edges {
		if e.From == "" || e.To == "" {
			continue
		}
		var attrs []string
		if e.Label != "" {
			attrs = append(attrs, fmt.Sprintf("label=%q", e.Label))
		}
		if e.Color != "" {
			attrs = append(attrs, fmt.Sprintf("color=%q", e.Color))
		}
		if e.Style != "" {
			attrs = append(attrs, fmt.Sprintf("style=%q", e.Style))
		}
		attrStr := ""
		if len(attrs) > 0 {
			attrStr = " [" + strings.Join(attrs, ", ") + "]"
		}
		b.WriteString(fmt.Sprintf("    %s%s%s%s;\n", sanitizeDotID(e.From), edgeOp, sanitizeDotID(e.To), attrStr))
	}

	b.WriteString("}\n")
	return b.String()
}

// sanitizeDotID 处理节点 ID：如果包含特殊字符或中文则加双引号
func sanitizeDotID(id string) string {
	for _, r := range id {
		if r > 127 || r == ' ' || r == '-' || r == '.' || r == '/' {
			return fmt.Sprintf("%q", id)
		}
	}
	if id == "" {
		return `""`
	}
	return id
}

var CreateFlowchartTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name: "根据 JSON 数据生成流程图",
		Desc: `传入结构化的节点和边数据（JSON），自动生成 DOT 并渲染为图片。
JSON 格式：{"title":"标题", "directed":true, "nodes":[{"id":"a","label":"开始","shape":"ellipse"},...], "edges":[{"from":"a","to":"b","label":"下一步"},...]}.
节点 shape 可选：box(矩形)、ellipse(椭圆)、diamond(菱形/判断)、circle(圆)、record、plaintext 等。
边支持 label(标签)、color(颜色)、style(dashed/bold/dotted)。
适合大模型根据用户描述自动组织节点和边，生成流程图/架构图/关系图。`,
		Tags:     []string{"Graphviz", "流程图", "架构图", "JSON", "自动生成"},
		Request:  &CreateFlowchartReq{},
		Response: &CreateFlowchartResp{},
	},
}

func init() {
	packageContext.POST("flowchart.form", CreateFlowchart, CreateFlowchartTemplate)
}
