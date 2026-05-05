package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type BuildWorkspaceTool struct{}

type buildWorkspaceArgs struct{}

var undefinedSDKSelectorRe = regexp.MustCompile(`undefined:\s*((?:app|types|chart|response|callback|statistics)\.[A-Z][A-Za-z0-9_]*)`)

var buildWorkspaceToolDef = toolDefinition[buildWorkspaceArgs](
	"build_workspace",
	"编译当前工作空间（Go 应用）。不写文件，仅基于当前已落盘的代码触发一次编译并部署。无需传参。连续写多个文件后可调用一次 build_workspace 再编译。",
)

func (t *BuildWorkspaceTool) Definition() dto.ToolDef {
	return buildWorkspaceToolDef
}

func (t *BuildWorkspaceTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	content, isError := runBuildWorkspaceTool(ctx, call.FullCodePath)
	return toolResult(content, isError)
}

// runBuildWorkspaceTool 编译当前工作空间（不写文件，仅触发编译并部署）；从当前工作目录解析 user/app，无需参数
func runBuildWorkspaceTool(ctx context.Context, currentFullCodePath string) (string, bool) {
	dir := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if dir == "" {
		return "build_workspace 无法获取当前工作目录，请确保在有效的工作台会话中操作", true
	}
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		return "build_workspace 当前目录格式应为 /user/app 或更长路径（如 /luobei/demo）", true
	}
	user, app := parts[0], parts[1]
	workspacePath := fmt.Sprintf("/%s/%s", user, app)
	resp, err := apicall.UpdateAppBuild(ctx, user, app)
	if err != nil {
		logger.Errorf(ctx, "[WorkspaceBuild] UpdateAppBuild 失败: %v", err)
		return "build_workspace 调用失败: " + enrichWorkspaceBuildError(err.Error(), workspacePath), true
	}
	return fmt.Sprintf("工作空间已编译并部署: workspace=%s, app=%s, 旧版本=%s, 新版本=%s", workspacePath, resp.App, resp.OldVersion, resp.NewVersion), false
}

func enrichWorkspaceBuildError(errText string, workspacePath ...string) string {
	prefix := workspaceBuildScopeMessage(workspacePath...)
	hints := workspaceBuildErrorHints(errText)
	if len(hints) == 0 {
		return prefix + errText
	}
	return prefix + errText + "\n\n常见修复建议:\n- " + strings.Join(hints, "\n- ")
}

func workspaceBuildScopeMessage(workspacePath ...string) string {
	if len(workspacePath) == 0 {
		return ""
	}
	scope := strings.TrimSpace(workspacePath[0])
	if scope == "" {
		return ""
	}
	return fmt.Sprintf("本次编译范围: %s。启动期校验只覆盖该工作空间内已注册路由；错误中的 router /xxx 是该工作空间内相对路由。\n", scope)
}

func workspaceBuildErrorHints(errText string) []string {
	var hints []string
	add := func(hint string) {
		for _, existing := range hints {
			if existing == hint {
				return
			}
		}
		hints = append(hints, hint)
	}

	if strings.Contains(errText, "options_colors") || strings.Contains(errText, "invalid color") || strings.Contains(errText, "contains invalid color") {
		add("options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，不能写 primary/success/default/rgb(...)，数量也必须和 options 一致。")
	}
	if strings.Contains(errText, "unsupported widget type") || strings.Contains(errText, "unsupported widget tag") || strings.Contains(errText, "invalid tag format") {
		add("widget 的 type 和配置 key 必须来自 widget-system 白名单；不要写 file/readonly/multiple 等未支持类型或参数。文件上传用 type:files + string，多文件用 max_count；只读展示用 display:\"scenes:list\" 或 widget:\"-\" 控制场景。")
	}
	if strings.Contains(errText, "number widget requires integer Go type") {
		add("数值组件要和 Go 类型匹配：type:number 只配 int/int64 等整数；float64 金额、评分、均值、比例使用 type:float。")
	}
	if strings.Contains(errText, "table request field") && strings.Contains(errText, "conflicts with table model field") {
		add("Table Request 不要重复声明任何 Model 字段 code；即使 Model 字段是 gorm:\"-\" 的计算/列表展示字段也会冲突。普通主表搜索写 Model 的 search 标签；计算字段筛选请把 Request 字段改名为 xxx_filter，或把列表展示字段改成 display_xxx。外键选择回调放在 Model 的 select/multiselect 字段上。")
	}
	if strings.Contains(errText, "undefined") && strings.Contains(errText, "Req has no field or method") {
		add("删除或合并 Table Request 字段后，要同步删除 Handler 里对 req.<字段> 的手写筛选；Model 字段搜索交给 search 标签和 AutoSearchFilterPaged。")
	}
	if strings.Contains(errText, "OnSelectFuzzyMap field") || strings.Contains(errText, "must use select or multiselect widget") {
		add("OnSelectFuzzyMap 的 key 必须对应 schema 中 type:select 或 type:multiselect 的字段；Table 新增/编辑要回调选择时，把外键字段定义在 Model 上并注册回调。")
	}
	if strings.Contains(errText, "GetPage undefined") || strings.Contains(errText, "GetPageSize undefined") ||
		strings.Contains(errText, "unknown field Total") || strings.Contains(errText, "unknown field DataList") {
		add("分页默认使用 resp.Table(&rows).AutoSearchFilterPaged(...)；不要调用 req.GetPage()/GetPageSize()，也不要用 Total/DataList 手工构造 query.PaginatedTable。")
	}
	if strings.Contains(errText, "Time.Format undefined") || strings.Contains(errText, "has no field or method Format") ||
		strings.Contains(errText, "type func() time.Time") || strings.Contains(errText, ".Time.Format") {
		add("types.Time 做格式化或比较时先调用 Time() 方法，例如 t.Time().Format(...)、t.Time().After(...)，不要写 t.Format(...) 或 t.Time.Format(...)。")
	}
	for _, match := range undefinedSDKSelectorRe.FindAllStringSubmatch(errText, -1) {
		add(fmt.Sprintf("代码使用了未确认的 SDK API %s；只允许使用已读文档、案例或源码中真实导出的符号，先读对应知识点或 SDK 源码，不要按命名猜 API。", match[1]))
	}
	if strings.Contains(errText, "does not implement chart.Charter") || strings.Contains(errText, "as chart.Charter value in argument to resp.Chart") {
		add("Chart Handler 必须把 SDK chart 包里的具体图表对象传给 resp.Chart；不要把自定义业务响应结构体传给 resp.Chart，附加指标放到图表 Metadata，多个图拆多个 .chart 路由。")
	}
	if strings.Contains(errText, "redeclared in this block") {
		add("同一个 Go package 里的模型、函数和方法只能定义一次；共享模型应放在一个文件中，其他文件直接复用。")
	}
	return hints
}
