# CreateFunctions 功能分析

## 需求概述

在 `UpdateApp` 中新增 `CreateFunctions` 参数，用于创建新生成的函数文件。相比 `ForkPackages`，`CreateFunctions` 更简单，主要用于：
- 获取目标目录路径（从 ServiceTree 获取）
- 基于 `group_code` 在指定目录创建 `.go` 文件
- 不需要替换 package（代码已经正确）

## 与 ForkPackages 的对比

### ForkPackages（复杂）
- **用途**：从源应用复制函数组到目标应用
- **需要处理**：
  1. 替换 package 名称（从源 package 替换为目标 package）
  2. 批量处理多个 package，每个 package 有多个文件
  3. 处理源代码转换
- **文件结构**：`ForkPackageInfo` → `ForkFunctionGroupFile`（包含 SourceCode、SourcePackage、GroupCode）

### CreateFunctions（简单）
- **用途**：创建新生成的函数文件（从 LLM 生成）
- **需要处理**：
  1. 直接写入文件（代码已经正确，不需要替换 package）
  2. 只需要目标目录路径和 group_code
  3. 代码内容从 `FunctionGenResult` 中获取
- **文件结构**：`CreateFunctionInfo`（包含 Package、GroupCode、SourceCode）

## 实现方案

### 1. DTO 定义

在 `dto/app_runtime_namespace.go` 中添加：

```go
// CreateFunctionInfo 创建函数信息
type CreateFunctionInfo struct {
	Package    string `json:"package"`     // 目标 package 路径（如 "crm" 或 "plugins/cashier"）
	GroupCode  string `json:"group_code"`  // 函数组代码（文件名，不含 .go）
	SourceCode string `json:"source_code"` // 源代码内容
}

// UpdateAppReq 更新应用请求
type UpdateAppReq struct {
	User            string                `json:"user" swaggerignore:"true"`
	App             string                `json:"app" binding:"required" example:"myapp"`
	ForkPackages    []*ForkPackageInfo   `json:"fork_packages,omitempty"`    // 可选的 Fork 包列表
	CreateFunctions []*CreateFunctionInfo `json:"create_functions,omitempty"` // 可选的新建函数列表
}
```

### 2. 服务层实现

在 `core/app-runtime/service/` 中创建 `create_function_service.go`：

```go
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// CreateFunctionService 创建函数服务
type CreateFunctionService struct {
	config *config.AppManageServiceConfig
}

// NewCreateFunctionService 创建服务
func NewCreateFunctionService(config *config.AppManageServiceConfig) *CreateFunctionService {
	return &CreateFunctionService{
		config: config,
	}
}

// CreateFunctions 批量创建函数文件
func (s *CreateFunctionService) CreateFunctions(ctx context.Context, user, app string, functions []*dto.CreateFunctionInfo) (*dto.CreateFunctionsResp, error) {
	logger.Infof(ctx, "[CreateFunctionService] 开始创建函数: target=%s/%s, functionCount=%d",
		user, app, len(functions))

	// 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, user, app)
	apiDir := filepath.Join(appDir, "code", "api")

	writtenFiles := make([]string, 0) // 记录已写入的文件路径，用于失败时回滚

	// 遍历所有函数，批量写入文件
	for _, funcInfo := range functions {
		packageDir := filepath.Join(apiDir, funcInfo.Package)

		// 确保 package 目录存在
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			// 失败时删除已写入的文件
			s.rollbackFiles(ctx, writtenFiles)
			return nil, fmt.Errorf("创建 package 目录失败 (%s): %w", funcInfo.Package, err)
		}

		// 构建目标文件路径
		targetFilePath := filepath.Join(packageDir, funcInfo.GroupCode+".go")

		// 写入文件（代码已经正确，不需要替换 package）
		if err := os.WriteFile(targetFilePath, []byte(funcInfo.SourceCode), 0644); err != nil {
			logger.Errorf(ctx, "[CreateFunctionService] 写入文件失败: file=%s, error=%v", targetFilePath, err)
			// 失败时删除已写入的文件
			s.rollbackFiles(ctx, writtenFiles)
			return nil, fmt.Errorf("写入文件失败 %s/%s: %w", funcInfo.Package, funcInfo.GroupCode, err)
		}

		// 记录已写入的文件
		writtenFiles = append(writtenFiles, targetFilePath)
		logger.Infof(ctx, "[CreateFunctionService] 文件写入成功: %s", targetFilePath)
	}

	logger.Infof(ctx, "[CreateFunctionService] 批量创建函数完成: fileCount=%d", len(writtenFiles))

	// 将绝对路径转换为相对路径（相对于应用目录）
	relativePaths := make([]string, 0, len(writtenFiles))
	for _, filePath := range writtenFiles {
		relPath, err := filepath.Rel(appDir, filePath)
		if err != nil {
			relPath = filePath
		}
		relativePaths = append(relativePaths, relPath)
	}

	return &dto.CreateFunctionsResp{
		Success:      true,
		Message:      fmt.Sprintf("成功创建 %d 个函数文件", len(writtenFiles)),
		WrittenFiles: relativePaths,
	}, nil
}

// rollbackFiles 回滚已写入的文件（失败时调用）
func (s *CreateFunctionService) rollbackFiles(ctx context.Context, files []string) {
	logger.Warnf(ctx, "[CreateFunctionService] 开始回滚已写入的文件: fileCount=%d", len(files))
	for _, filePath := range files {
		if err := os.Remove(filePath); err != nil {
			logger.Errorf(ctx, "[CreateFunctionService] 删除文件失败: file=%s, error=%v", filePath, err)
		} else {
			logger.Infof(ctx, "[CreateFunctionService] 已删除文件: %s", filePath)
		}
	}
	logger.Infof(ctx, "[CreateFunctionService] 文件回滚完成")
}
```

在 `dto/app_runtime_namespace.go` 中添加响应：

```go
// CreateFunctionsResp 创建函数响应
type CreateFunctionsResp struct {
	Success      bool     `json:"success" example:"true"`
	Message      string   `json:"message" example:"文件创建成功"`
	WrittenFiles []string `json:"written_files"` // 已写入的文件路径列表（用于失败时回滚）
}
```

### 3. 在 UpdateApp 中集成

在 `core/app-runtime/service/app_manage_service.go` 的 `UpdateApp` 方法中：

```go
// UpdateApp 更新应用（重新编译并重启容器）
// 如果提供了 ForkPackages，先执行 fork 操作
// 如果提供了 CreateFunctions，先执行创建函数操作，再执行更新
func (s *AppManageService) UpdateApp(ctx context.Context, user, app string, forkPackages []*sharedDto.ForkPackageInfo, createFunctions []*sharedDto.CreateFunctionInfo) (*dto.UpdateAppResp, error) {
	// 0. 如果有 CreateFunctions，先执行创建函数操作
	var writtenFiles []string
	if len(createFunctions) > 0 {
		logger.Infof(ctx, "[UpdateApp] 检测到 CreateFunctions，先执行创建函数操作: functionCount=%d", len(createFunctions))
		
		createResp, err := s.createFunctionService.CreateFunctions(ctx, user, app, createFunctions)
		if err != nil {
			logger.Errorf(ctx, "[UpdateApp] 创建函数失败: error=%v", err)
			return nil, fmt.Errorf("创建函数失败: %w", err)
		}
		
		if !createResp.Success {
			logger.Errorf(ctx, "[UpdateApp] 创建函数失败: %s", createResp.Message)
			if len(createResp.WrittenFiles) > 0 {
				s.rollbackCreateFunctionFiles(ctx, user, app, createResp.WrittenFiles)
			}
			return nil, fmt.Errorf("创建函数失败: %s", createResp.Message)
		}
		
		writtenFiles = createResp.WrittenFiles
		logger.Infof(ctx, "[UpdateApp] 创建函数成功: fileCount=%d", len(writtenFiles))
	}

	// 1. 如果有 ForkPackages，执行 fork 操作（与现有逻辑相同）
	// ...
	
	// 2. 继续执行更新逻辑
	// ...
}
```

### 4. 在 app-server 中调用

在 `core/app-server/server/server.go` 的 `handleFunctionGenResult` 中：

```go
// handleFunctionGenResult 处理函数生成结果消息
func (s *Server) handleFunctionGenResult(msg *nats.Msg) {
	// ... 解析消息 ...
	
	// 1. 根据 TreeID 获取 ServiceTree，获取 package 路径
	serviceTree, err := s.serviceTreeRepo.GetByID(result.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[Server] 获取 ServiceTree 失败: %v", err)
		return
	}
	
	// 2. 解析生成的代码，提取 group_code
	// （可以从代码中解析，或者从 ServiceTree 的 code 字段获取）
	groupCode := serviceTree.Code // 或者从代码中解析
	
	// 3. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    serviceTree.GetPackagePath(), // 需要实现这个方法
		GroupCode:  groupCode,
		SourceCode: result.Code,
	}
	
	// 4. 调用 UpdateApp，传入 CreateFunctions
	updateReq := &dto.UpdateAppReq{
		User:            result.User,
		App:             app.Code, // 需要从 ServiceTree 获取 app
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}
	
	// 调用 app-runtime 的 UpdateApp
	_, err = s.appRuntime.UpdateApp(ctx, app.HostID, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[Server] UpdateApp 失败: %v", err)
		return
	}
}
```

## 优势

1. **简单**：不需要替换 package，直接写入文件
2. **统一**：与 ForkPackages 使用相同的 UpdateApp 流程
3. **可回滚**：失败时可以删除已写入的文件
4. **批量处理**：支持一次创建多个函数文件

## 注意事项

1. **Package 路径获取**：需要从 ServiceTree 中获取 package 路径（可能需要实现 `GetPackagePath()` 方法）
2. **GroupCode 获取**：可以从 ServiceTree 的 code 字段获取，或者从生成的代码中解析
3. **代码解析**：如果需要从代码中提取 group_code，可能需要简单的代码解析逻辑
4. **错误处理**：需要确保失败时能够正确回滚已写入的文件

## 总结

`CreateFunctions` 功能是可行的，实现相对简单，主要优势是：
- 不需要复杂的 package 替换逻辑
- 可以直接复用 ForkPackages 的文件写入和回滚机制
- 与现有的 UpdateApp 流程完美集成

