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

// ServiceTreeService 服务目录管理服务
type ServiceTreeService struct {
	config           *config.AppManageServiceConfig
	appManageService *AppManageService // 用于编译和获取 diff
}

// NewServiceTreeService 创建服务目录管理服务
func NewServiceTreeService(config *config.AppManageServiceConfig) *ServiceTreeService {
	return &ServiceTreeService{
		config: config,
	}
}

// SetAppManageService 设置应用管理服务（用于编译和获取 diff）
func (s *ServiceTreeService) SetAppManageService(appManageService *AppManageService) {
	s.appManageService = appManageService
}

// CreateServiceTree 创建服务目录
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeRuntimeReq) (*dto.CreateServiceTreeRuntimeResp, error) {
	logger.Infof(ctx, "[ServiceTreeService] Creating service tree: %s/%s/%s", req.User, req.App, req.ServiceTree.Code)

	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := appPaths.APIDir()

	// 确保api目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create api directory: %w", err)
	}

	packagePath, err := validateBatchWritePackagePath(req.User, req.App, req.ServiceTree.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("invalid service tree path: %w", err)
	}

	// 创建包目录
	packageDir := filepath.Join(apiDir, packagePath)
	if err := ensurePathWithinBase(apiDir, packageDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create package directory: %w", err)
	}

	// 生成init_.go文件
	if err := writePackageInitFile(packageDir, req.ServiceTree.Code, "/"+packagePath, req.ServiceTree.Name, req.ServiceTree.Description); err != nil {
		return nil, fmt.Errorf("failed to generate init file: %w", err)
	}

	// 新增：自动更新main文件，添加新包的import
	if err := s.updateMainFileImports(ctx, req.User, req.App, packagePath); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to update main file imports: %v", err)
		// 不返回错误，因为服务目录已经创建成功，只是import可能需要手动添加
	} else {
		logger.Infof(ctx, "[ServiceTreeService] Main file updated successfully with new import")
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully: %s", packageDir)

	return &dto.CreateServiceTreeRuntimeResp{
		User:        req.User,
		App:         req.App,
		ServiceTree: req.ServiceTree.Code,
	}, nil
}

// DeleteServiceTree 删除服务目录（删磁盘目录，并从 main.go 移除该包的 import）
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, user, app, serviceTreeName string) error {
	logger.Infof(ctx, "[ServiceTreeService] Deleting service tree: %s/%s/%s", user, app, serviceTreeName)

	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, user, app)
	packagePath, err := validateRelativePackagePath(serviceTreeName)
	if err != nil {
		return fmt.Errorf("invalid service tree path: %w", err)
	}
	packageDir := filepath.Join(appPaths.APIDir(), packagePath)
	if err := ensurePathWithinBase(appPaths.APIDir(), packageDir); err != nil {
		return err
	}

	// 1. 删除磁盘目录
	if err := os.RemoveAll(packageDir); err != nil {
		return fmt.Errorf("failed to delete package directory: %w", err)
	}

	// 2. 从 main.go 移除该包的 import
	if err := s.removeMainFileImport(ctx, user, app, packagePath); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to remove import from main.go: %v", err)
		// 不返回错误，目录已删，import 可手动处理
	} else {
		logger.Infof(ctx, "[ServiceTreeService] Removed import from main.go: %s", packagePath)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree deleted successfully: %s", packageDir)
	return nil
}

// DeleteServiceTreeByReq 按请求删除服务目录（供 NATS 调用）
func (s *ServiceTreeService) DeleteServiceTreeByReq(ctx context.Context, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error) {
	if err := s.DeleteServiceTree(ctx, req.User, req.App, req.PackagePath); err != nil {
		return &dto.DeleteServiceTreeRuntimeResp{Success: false, Error: err.Error()}, nil
	}
	return &dto.DeleteServiceTreeRuntimeResp{Success: true}, nil
}

// removeMainFileImport 从 main.go 中移除指定包的 import 行
func (s *ServiceTreeService) removeMainFileImport(ctx context.Context, user, app, packagePath string) error {
	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, user, app)
	mainFilePath := appPaths.MainGoPath()
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		return nil
	}
	cleanPath, err := validateRelativePackagePath(packagePath)
	if err != nil {
		return err
	}
	importPath := appPaths.NamespaceAPIImport(cleanPath)

	changed, err := removeNamedImportFromGoFile(mainFilePath, "_", importPath)
	if err != nil {
		return fmt.Errorf("failed to remove import from main file: %w", err)
	}
	if !changed {
		logger.Infof(ctx, "[ServiceTreeService] Import not found in main.go: %s", importPath)
	}
	return nil
}

// updateMainFileImports 更新 main.go，引入服务目录对应的 blank import
func (s *ServiceTreeService) updateMainFileImports(ctx context.Context, user, app, packagePath string) error {
	logger.Infof(ctx, "[ServiceTreeService] Updating main file imports for package: %s", packagePath)

	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, user, app)
	mainFilePath := appPaths.MainGoPath()

	// 检查main文件是否存在
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		return fmt.Errorf("main file does not exist: %s", mainFilePath)
	}

	// 清理 packagePath：去掉首尾斜杠，确保不会生成有尾随斜杠的 import
	cleanPackagePath, err := validateRelativePackagePath(packagePath)
	if err != nil {
		return err
	}
	importPath := appPaths.NamespaceAPIImport(cleanPackagePath)

	changed, err := addNamedImportToGoFile(mainFilePath, "_", importPath)
	if err != nil {
		return fmt.Errorf("failed to update main file imports: %w", err)
	}
	if !changed {
		logger.Infof(ctx, "[ServiceTreeService] Import already exists: %s", importPath)
		return nil
	}

	logger.Infof(ctx, "[ServiceTreeService] Successfully added import: %s", importPath)
	return nil
}

// BatchCreateDirectoryTree 批量创建目录树（只处理目录，不处理文件）
// 文件写入请使用 BatchWriteFiles 方法
func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeRuntimeReq,
) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	logger.Infof(ctx, "[ServiceTreeService] 开始批量创建目录树: user=%s, app=%s, itemCount=%d",
		req.User, req.App, len(req.Items))

	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := appPaths.APIDir()

	// 确保 api 目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 api 目录失败: %w", err)
	}

	// 2. 过滤出目录项（只处理目录，不处理文件）
	directoryItems := make([]*dto.DirectoryTreeItem, 0)
	for _, item := range req.Items {
		if item.Type == "directory" {
			directoryItems = append(directoryItems, item)
		} else if item.Type == "file" {
			logger.Warnf(ctx, "[ServiceTreeService] 跳过文件项，文件写入请使用 BatchWriteFiles: path=%s", item.FullCodePath)
		}
	}

	// 3. 按路径排序，确保先创建父目录
	sortedItems := sortItemsByPath(directoryItems)

	directoryCount := 0
	createdPaths := make([]string, 0)

	// 4. 遍历所有目录项，逐个创建目录
	for _, item := range sortedItems {
		packagePath, packageDir, err := resolveDirectoryTarget(req.User, req.App, apiDir, item)
		if err != nil {
			return nil, fmt.Errorf("创建目录失败 (%s): %w", item.FullCodePath, err)
		}

		if err := os.MkdirAll(packageDir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 (%s): %w", item.FullCodePath, err)
		}
		directoryCount++
		createdPaths = append(createdPaths, item.FullCodePath)

		// 生成 init_.go 文件
		if err := writePackageInitFile(packageDir, packageCodeFromPath(packagePath), "/"+packagePath, item.Name, item.Description); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 生成 init_.go 失败: path=%s, error=%v",
				item.FullCodePath, err)
			// 不返回错误，因为目录已创建成功
		}

		// 更新 main.go 文件，添加新包的 import
		if packagePath != "" {
			if err := s.updateMainFileImports(ctx, req.User, req.App, packagePath); err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 更新 main.go import 失败: path=%s, error=%v",
					item.FullCodePath, err)
				// 不返回错误，因为目录已创建成功，只是 import 可能需要手动添加
			} else {
				logger.Infof(ctx, "[ServiceTreeService] Main.go import 更新成功: package=%s", packagePath)
			}
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量创建目录树完成: directoryCount=%d", directoryCount)

	return &dto.BatchCreateDirectoryTreeRuntimeResp{
		DirectoryCount: directoryCount,
		FileCount:      0, // 不再处理文件
		CreatedPaths:   createdPaths,
	}, nil
}

// BatchWriteFiles 批量写文件（批量写文件，编译，返回 diff）
func (s *ServiceTreeService) BatchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesRuntimeReq,
) (*dto.BatchWriteFilesRuntimeResp, error) {
	logger.Infof(ctx, "[ServiceTreeService] 开始批量写文件: user=%s, app=%s, fileCount=%d",
		req.User, req.App, len(req.Files))

	appPaths := newRuntimeAppPaths(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := appPaths.APIDir()

	// 确保 api 目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 api 目录失败: %w", err)
	}

	// 2. 前置校验
	if s.appManageService == nil {
		return nil, fmt.Errorf("appManageService 未设置，无法编译应用")
	}
	if len(req.Files) == 0 {
		return nil, fmt.Errorf("没有需要写入的文件")
	}

	state, err := s.writeBatchFilesToDisk(ctx, req.User, req.App, apiDir, req.Files)
	if err != nil {
		return nil, err
	}

	result, err := s.appManageService.finalizeWrittenAppChanges(ctx, req.User, req.App, appPaths)
	if err != nil {
		logger.Warnf(ctx, "[BatchWriteFiles] 编译失败，开始回滚已写入的文件: fileCount=%d", len(state.rollbackOrder))
		s.rollbackBatchWriteState(ctx, state)
		return nil, err
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量写文件并编译完成: oldVersion=%s, newVersion=%s", result.oldVersion, result.newVersion)

	return &dto.BatchWriteFilesRuntimeResp{
		FileCount:     len(state.writtenPaths),
		WrittenPaths:  state.writtenPaths,
		Diff:          result.diff,
		OldVersion:    result.oldVersion,
		NewVersion:    result.newVersion,
		GitCommitHash: result.gitCommitHash,
	}, nil
}
