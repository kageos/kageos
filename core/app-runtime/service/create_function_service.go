package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/gofmt"
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

		// ⭐ 从 package 路径中提取目录代码（最后一部分）作为正确的 package 名称
		// packagePath 格式：例如 "newtask12/requirement"，需要提取 "requirement"
		packagePathParts := strings.Split(strings.Trim(funcInfo.Package, "/"), "/")
		correctPackageName := packagePathParts[len(packagePathParts)-1]
		if correctPackageName == "" {
			// 如果提取失败，使用 GroupCode 作为 fallback
			correctPackageName = funcInfo.GroupCode
		}

		// ⭐ 修复代码中的 package 声明，确保与目录代码一致
		codeToWrite := s.fixPackageDeclaration(funcInfo.SourceCode, correctPackageName)

		// 尝试修复 Go 代码的 import 语句（防止编译不通过）
		// 如果修复失败，使用原代码（代码来自快照，应该已经是正确的）
		fixedCode, err := gofmt.FixGoImport(targetFilePath, []byte(codeToWrite))
		if err != nil {
			// 修复失败时，记录警告但继续使用原代码
			// 因为代码来自快照，package 路径已经正确，可能不需要修复
			logger.Warnf(ctx, "[CreateFunctionService] 修复 import 失败，使用原代码: file=%s, error=%v", targetFilePath, err)
			codeToWrite = codeToWrite
		} else {
			codeToWrite = fixedCode
		}

		// 写入文件
		if err := os.WriteFile(targetFilePath, []byte(codeToWrite), 0644); err != nil {
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
			// 如果计算失败，使用完整路径
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

// fixPackageDeclaration 修复代码中的 package 声明，确保与目录代码一致
// sourceCode: 原始代码
// correctPackageName: 正确的 package 名称（目录代码）
func (s *CreateFunctionService) fixPackageDeclaration(sourceCode string, correctPackageName string) string {
	// 匹配：package old_package_name（使用多行模式，支持文件开头有注释的情况）
	// (?m) 表示多行模式，^ 匹配每行的开始
	// 匹配第一个 package 声明（Go 文件只能有一个 package 声明）
	re := regexp.MustCompile(`(?m)^package\s+\w+`)
	// 找到第一个匹配的位置
	loc := re.FindStringIndex(sourceCode)
	if loc == nil {
		// 如果没有找到 package 声明，返回原代码
		return sourceCode
	}

	// 提取当前的 package 名称
	currentPackageDecl := sourceCode[loc[0]:loc[1]]
	currentPackageName := strings.TrimSpace(strings.TrimPrefix(currentPackageDecl, "package"))

	// 如果 package 名称已经正确，不需要替换
	if currentPackageName == correctPackageName {
		return sourceCode
	}

	// 替换 package 声明
	processed := sourceCode[:loc[0]] + fmt.Sprintf("package %s", correctPackageName) + sourceCode[loc[1]:]

	return processed
}

