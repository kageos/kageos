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

