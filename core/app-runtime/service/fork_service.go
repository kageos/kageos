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
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// ForkService 函数组 Fork 服务
type ForkService struct {
	config *config.AppManageServiceConfig
}

// NewForkService 创建 Fork 服务
func NewForkService(config *config.AppManageServiceConfig) *ForkService {
	return &ForkService{
		config: config,
	}
}

// ForkFunctionGroup 批量 Fork 函数组（写入文件系统）
// 一次调用处理多个 package，每个 package 有自己的文件列表
func (s *ForkService) ForkFunctionGroup(ctx context.Context, req *dto.ForkFunctionGroupRuntimeReq) (*dto.ForkFunctionGroupRuntimeResp, error) {
	logger.Infof(ctx, "[ForkService] 开始批量 Fork 函数组: target=%s/%s, packageCount=%d", 
		req.TargetUser, req.TargetApp, len(req.Packages))

	// 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, req.TargetUser, req.TargetApp)
	apiDir := filepath.Join(appDir, "code", "api")

	totalFileCount := 0
	writtenFiles := make([]string, 0) // 记录已写入的文件路径，用于失败时回滚

	// 遍历所有 package，批量写入文件
	for _, pkgInfo := range req.Packages {
		packageDir := filepath.Join(apiDir, pkgInfo.Package)

		// 确保 package 目录存在
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			// 失败时删除已写入的文件
			s.rollbackFiles(ctx, writtenFiles)
			return nil, fmt.Errorf("创建 package 目录失败 (%s): %w", pkgInfo.Package, err)
		}

		// 批量写入该 package 下的文件
		for _, file := range pkgInfo.Files {
			// 处理源代码（替换 package）
			processedCode := s.replacePackageName(file.SourceCode, file.SourcePackage, pkgInfo.Package)

			// 构建目标文件路径
			targetFilePath := filepath.Join(packageDir, file.GroupCode+".go")

			// 写入文件
			if err := os.WriteFile(targetFilePath, []byte(processedCode), 0644); err != nil {
				logger.Errorf(ctx, "[ForkService] 写入文件失败: file=%s, error=%v", targetFilePath, err)
				// 失败时删除已写入的文件
				s.rollbackFiles(ctx, writtenFiles)
				return nil, fmt.Errorf("写入文件失败 %s/%s: %w", pkgInfo.Package, file.GroupCode, err)
			}

			// 记录已写入的文件
			writtenFiles = append(writtenFiles, targetFilePath)
			totalFileCount++
			logger.Infof(ctx, "[ForkService] 文件写入成功: %s", targetFilePath)
		}

		logger.Infof(ctx, "[ForkService] Package %s 的文件写入完成: fileCount=%d", pkgInfo.Package, len(pkgInfo.Files))
	}

	logger.Infof(ctx, "[ForkService] 批量 Fork 完成: totalFileCount=%d, packageCount=%d", totalFileCount, len(req.Packages))

	// 将绝对路径转换为相对路径（相对于应用目录）
	relativePaths := make([]string, 0, len(writtenFiles))
	for _, filePath := range writtenFiles {
		// 计算相对路径：从应用目录开始
		relPath, err := filepath.Rel(appDir, filePath)
		if err != nil {
			// 如果计算失败，使用完整路径
			relPath = filePath
		}
		relativePaths = append(relativePaths, relPath)
	}

	return &dto.ForkFunctionGroupRuntimeResp{
		Success:      true,
		Message:      fmt.Sprintf("成功 Fork %d 个函数组到 %d 个 package", totalFileCount, len(req.Packages)),
		WrittenFiles: relativePaths, // 返回相对路径列表，用于失败时回滚
	}, nil
}

// rollbackFiles 回滚已写入的文件（失败时调用）
func (s *ForkService) rollbackFiles(ctx context.Context, files []string) {
	logger.Warnf(ctx, "[ForkService] 开始回滚已写入的文件: fileCount=%d", len(files))
	for _, filePath := range files {
		if err := os.Remove(filePath); err != nil {
			logger.Errorf(ctx, "[ForkService] 删除文件失败: file=%s, error=%v", filePath, err)
		} else {
			logger.Infof(ctx, "[ForkService] 已删除文件: %s", filePath)
		}
	}
	logger.Infof(ctx, "[ForkService] 文件回滚完成")
}

// replacePackageName 替换源代码中的 package 声明
// sourcePackage 和 targetPackage 可能是多级路径（如 tools/cashier），但 Go 的 package 声明只能是单个标识符
// 所以需要从路径中提取最后一部分作为 package 名称
func (s *ForkService) replacePackageName(sourceCode string, sourcePackage string, targetPackage string) string {
	// 从路径中提取最后一部分作为 package 名称
	getPackageName := func(path string) string {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 {
			return path
		}
		return parts[len(parts)-1]
	}

	sourcePackageName := getPackageName(sourcePackage)
	targetPackageName := getPackageName(targetPackage)

	// 如果 package 相同，不需要替换
	if sourcePackageName == targetPackageName {
		return sourceCode
	}

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
	// 只替换第一个匹配
	processed := sourceCode[:loc[0]] + fmt.Sprintf("package %s", targetPackageName) + sourceCode[loc[1]:]

	return processed
}



