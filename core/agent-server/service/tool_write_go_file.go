package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/dto"
)

type WriteGoFileTool struct{}

type writeGoFileArgs struct {
	Directory    string `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"普通代码文件名，不能以 _test.go 结尾" schema_required:"true"`
	Content      string `json:"content" schema_desc:"代码全文" schema_required:"true"`
	SourceCode   string `json:"source_code" schema_ignore:"true"`
}

var writeGoFileToolDef = toolDefinition[writeGoFileArgs](
	"write_go_file",
	"在当前工作目录或指定 directory 下写入一个普通 .go 代码文件，不能写入 _test.go 文件；测试文件不会参与应用 API 注册。必填：file_name（如 attendance.go）、content（代码全文）。可选：directory（目标目录）。调用前要求：创建系统/新增 Form/Table/Chart 时，先由 product_manager 通过 write_prd 输出结构化 PRD 并得到用户确认；进入 app_developer 后按已确认 PRD 读取 1 到多个匹配案例，再生成结构体和函数代码。复杂图表、消息、事务、平台 API 等专项写法按当前角色 SOP 的明确 read_doc 路径读取。修改已有应用时，先读取目录和相关代码文件；search_replace_file 匹配失败后必须重新 read_go_file，再基于实际内容修改。数据库代码可按业务需要使用 ctx.GetGormDB()、GORM 链式 API、事务、Raw/Exec 等能力；仍需自行保证 SQL 参数化和业务数据安全。注意：write_go_file 只落盘、不编译；写入成功后会自动附带文件级非阻断代码诊断。创建或修改复杂系统时，不要因单个文件的非阻断诊断中断后续文件写入；先写完整轮次涉及的全部代码文件，再批量修复诊断并统一调用 build_workspace。若本工具返回 error，本次文件未落盘，必须先修正参数或内容，不要声称文件已创建。",
)

func (t *WriteGoFileTool) Definition() dto.ToolDef {
	return writeGoFileToolDef
}

func (t *WriteGoFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[writeGoFileArgs](call.Args)
	if err != nil {
		return toolResult("write_go_file 参数解析失败: "+err.Error(), true)
	}
	content, isError := runWriteGoFileTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runWriteGoFileTool 写代码文件；固定仅落盘，不触发编译
func runWriteGoFileTool(ctx context.Context, args writeGoFileArgs, currentFullCodePath string) (string, bool) {
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return "write_go_file 缺少参数 file_name，本次未落盘。请补充普通代码文件名（如 nps_statistics.go），不要继续假设该文件已创建。", true
	}
	content := args.Content
	if content == "" {
		content = args.SourceCode
	}
	if content == "" {
		return "write_go_file 缺少参数 content，本次未落盘。请补充完整代码后重试，不要继续假设该文件已创建。", true
	}
	if !strings.HasSuffix(fileName, ".go") {
		fileName += ".go"
	}
	if isGoTestFileName(fileName) {
		return "write_go_file 不允许写入 _test.go 文件；测试文件不会参与应用 API 注册。请改用普通业务代码文件名（如 nps_statistics.go）。本次未落盘。", true
	}
	if strings.TrimSuffix(fileName, ".go") == "init_" {
		return "不允许创建该文件，由脚手架自动生成。", true
	}

	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	msg, isError := runAddFunctionsCommand(ctx, addFunctionsCommand{
		FileName:   fileName,
		SourceCode: content,
	}, targetPath, false)
	if isError {
		return msg, true
	}
	return appendGoFileDiagnostics(msg, targetPath, fileName, content), false
}

func isGoTestFileName(fileName string) bool {
	base := strings.TrimSpace(fileName)
	for strings.HasSuffix(base, ".go") {
		base = strings.TrimSuffix(base, ".go")
	}
	return strings.HasSuffix(base, "_test")
}
