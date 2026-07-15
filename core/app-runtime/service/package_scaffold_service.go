package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

// PackageScaffoldService 管理 package 目录脚手架，包括目录创建、init_.go 维护和 main.go import 同步。
type PackageScaffoldService struct {
	config             *config.AppManageServiceConfig
	appDatabaseService *AppDatabaseService
}

// NewPackageScaffoldService 创建 package 脚手架服务。
func NewPackageScaffoldService(config *config.AppManageServiceConfig) *PackageScaffoldService {
	return &PackageScaffoldService{config: config}
}

func (s *PackageScaffoldService) SetAppDatabaseService(appDatabaseService *AppDatabaseService) {
	s.appDatabaseService = appDatabaseService
}

// DeleteServiceTree 删除目录脚手架，并从 main.go 移除 blank import。
func (s *PackageScaffoldService) DeleteServiceTree(ctx context.Context, user, app, packagePath string) error {
	logger.Infof(ctx, "[PackageScaffoldService] Deleting service tree: %s/%s/%s", user, app, packagePath)

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	cleanPath, err := validateRelativePackagePath(packagePath)
	if err != nil {
		return fmt.Errorf("invalid service tree path: %w", err)
	}

	packageDir := filepath.Join(appPaths.APIDir(), cleanPath)
	if err := ensurePathWithinBase(appPaths.APIDir(), packageDir); err != nil {
		return err
	}
	if err := os.RemoveAll(packageDir); err != nil {
		return fmt.Errorf("failed to delete package directory: %w", err)
	}

	if err := s.removeMainFileImport(ctx, user, app, cleanPath); err != nil {
		logger.Warnf(ctx, "[PackageScaffoldService] Failed to remove import from main.go: %v", err)
	} else {
		logger.Infof(ctx, "[PackageScaffoldService] Removed import from main.go: %s", cleanPath)
	}

	logger.Infof(ctx, "[PackageScaffoldService] Service tree deleted successfully: %s", packageDir)
	return nil
}

// BatchCreateDirectoryTree 批量创建目录脚手架。
func (s *PackageScaffoldService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeRuntimeReq,
) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	logger.Infof(ctx, "[PackageScaffoldService] 开始批量创建目录树: user=%s, app=%s, itemCount=%d",
		req.User, req.App, len(req.Items))

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), req.User, req.App)
	apiDir := appPaths.APIDir()
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 api 目录失败: %w", err)
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	for _, item := range req.Items {
		directoryItems = append(directoryItems, item)
	}

	sortedItems := sortDirectoryItemsByPath(directoryItems)
	directoryCount := 0
	createdPaths := make([]string, 0, len(sortedItems))

	for _, item := range sortedItems {
		packagePath, packageDir, err := resolveDirectoryTarget(req.User, req.App, apiDir, item)
		if err != nil {
			return nil, fmt.Errorf("创建目录失败 (%s): %w", item.FullCodePath, err)
		}

		if err := os.MkdirAll(packageDir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 (%s): %w", item.FullCodePath, err)
		}
		if s.appDatabaseService != nil && s.appDatabaseService.IsEnabled() && packagePath != "" {
			if err := s.appDatabaseService.EnsureDatabaseForPackage(ctx, req.User, req.App, packagePath); err != nil {
				return nil, fmt.Errorf("创建目录数据库失败 (%s): %w", item.FullCodePath, err)
			}
		}
		directoryCount++
		createdPaths = append(createdPaths, item.FullCodePath)

		if err := writePackageInitFile(packageDir, packageCodeFromPath(packagePath), "/"+packagePath, item.Name, item.Description); err != nil {
			logger.Warnf(ctx, "[PackageScaffoldService] 生成 init_.go 失败: path=%s, error=%v", item.FullCodePath, err)
		}

		if packagePath != "" {
			if err := s.updateMainFileImports(ctx, req.User, req.App, packagePath); err != nil {
				logger.Warnf(ctx, "[PackageScaffoldService] 更新 main.go import 失败: path=%s, error=%v", item.FullCodePath, err)
			} else {
				logger.Infof(ctx, "[PackageScaffoldService] Main.go import 更新成功: package=%s", packagePath)
			}
		}
	}

	logger.Infof(ctx, "[PackageScaffoldService] 批量创建目录树完成: directoryCount=%d", directoryCount)
	return &dto.BatchCreateDirectoryTreeRuntimeResp{
		DirectoryCount: directoryCount,
		FileCount:      0,
		CreatedPaths:   createdPaths,
	}, nil
}

// removeMainFileImport 从 main.go 中移除指定包的 import 行。
func (s *PackageScaffoldService) removeMainFileImport(ctx context.Context, user, app, packagePath string) error {
	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	mainFilePath := appPaths.MainGoPath()
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		return nil
	}
	cleanPath, err := validateRelativePackagePath(packagePath)
	if err != nil {
		return err
	}
	importPath := appPaths.NamespaceAPIImport(cleanPath)

	changed, err := removeNamedImportsWithPathPrefixFromGoFile(mainFilePath, "_", importPath)
	if err != nil {
		return fmt.Errorf("failed to remove import from main file: %w", err)
	}
	if !changed {
		logger.Infof(ctx, "[PackageScaffoldService] Import prefix not found in main.go: %s", importPath)
	}
	return nil
}

// updateMainFileImports 更新 main.go，引入服务目录对应的 blank import。
func (s *PackageScaffoldService) updateMainFileImports(ctx context.Context, user, app, packagePath string) error {
	logger.Infof(ctx, "[PackageScaffoldService] Updating main file imports for package: %s", packagePath)

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	mainFilePath := appPaths.MainGoPath()
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		return fmt.Errorf("main file does not exist: %s", mainFilePath)
	}

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
		logger.Infof(ctx, "[PackageScaffoldService] Import already exists: %s", importPath)
		return nil
	}

	logger.Infof(ctx, "[PackageScaffoldService] Successfully added import: %s", importPath)
	return nil
}
