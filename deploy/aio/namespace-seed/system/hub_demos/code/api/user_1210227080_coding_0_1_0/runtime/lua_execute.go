//<文件名>lua_execute.go</文件名>

package runtime

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

// LuaExecuteReq Lua代码执行请求结构体
type LuaExecuteReq struct {
	// 框架标签：widget:"type:text_area;placeholder:请输入Lua代码..." - 代码输入框
	LuaCode string `json:"lua_code" widget:"name:Lua代码;type:text_area;placeholder:请输入Lua代码...例如: print('Hello, World!')" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:例如: arg1,arg2" - 命令行参数（可选）
	Args string `json:"args" widget:"name:命令行参数;type:input;placeholder:例如: arg1,arg2（多个参数用逗号分隔）"`
}

// LuaExecuteResp Lua代码执行响应结构体
type LuaExecuteResp struct {
	// 执行结果输出
	Output string `json:"output" widget:"name:执行结果;type:text_area"`

	// 执行状态
	Status string `json:"status" widget:"name:执行状态;type:text"`
}

// LuaExecute Lua代码执行函数
func LuaExecute(ctx *app.Context, resp response.Response) error {
	var req LuaExecuteReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 直接从 PATH 查找 Lua，避免依赖额外环境变量配置。
	luaPath, err := exec.LookPath("lua")
	if err != nil {
		return fmt.Errorf("未找到 lua，请确认运行环境已安装 Lua")
	}

	// 2. 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()

	// 3. 将 Lua 代码写入临时文件
	luaScriptPath := filepath.Join(outputDir, "script.lua")
	err = os.WriteFile(luaScriptPath, []byte(req.LuaCode), 0644)
	if err != nil {
		logger.Errorf(ctx, "[LuaExecute] 写入Lua脚本失败: %v", err)
		return fmt.Errorf("写入Lua脚本失败: %v", err)
	}
	defer os.Remove(luaScriptPath) // 清理临时文件

	// 4. 构建 Lua 执行命令
	// lua script.lua [args...]
	args := []string{luaScriptPath}

	// 解析命令行参数（如果有）
	if req.Args != "" {
		// 按逗号分割参数
		argList := strings.Split(req.Args, ",")
		for _, arg := range argList {
			arg = strings.TrimSpace(arg)
			if arg != "" {
				args = append(args, arg)
			}
		}
	}

	// 5. 执行 Lua 脚本
	cmd := exec.Command(luaPath, args...)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	status := "成功"

	if err != nil {
		logger.Errorf(ctx, "[LuaExecute] 执行Lua脚本失败: %v, output: %s", err, outputStr)
		status = "失败"
		// 如果执行失败，将错误信息也包含在输出中
		outputStr = fmt.Sprintf("执行错误: %v\n\n输出:\n%s", err, outputStr)
	} else {
		logger.Infof(ctx, "[LuaExecute] 执行Lua脚本成功")
	}

	// 6. 如果输出为空，显示提示信息
	if outputStr == "" {
		outputStr = "（脚本执行完成，无输出）"
	}

	// 7. 构建响应
	return resp.Form(&LuaExecuteResp{
		Output: outputStr,
		Status: status,
	}).Build()
}

// LuaExecuteTemplate Lua代码执行配置
var LuaExecuteTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Lua代码执行",
		Desc:     `执行Lua脚本代码。支持输入Lua代码并执行，返回执行结果。可以传递命令行参数给脚本。使用系统安装的Lua解释器执行代码。应用场景：脚本执行、数据处理、自动化任务、快速原型开发等。`,
		Tags:     []string{"脚本执行", "Lua", "工具"},
		Request:  &LuaExecuteReq{},
		Response: &LuaExecuteResp{},
	},
}

func init() {
	// 注册Form函数 - Lua代码执行
	packageContext.POST("lua.form", LuaExecute, LuaExecuteTemplate)
}
