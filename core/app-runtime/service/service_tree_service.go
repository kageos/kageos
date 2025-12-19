package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	appPkg "github.com/ai-agent-os/ai-agent-os/pkg/app"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// ServiceTreeService 服务目录管理服务
type ServiceTreeService struct {
	config          *config.AppManageServiceConfig
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

	// 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := filepath.Join(appDir, "code", "api")

	// 确保api目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create api directory: %w", err)
	}

	// 根据父目录ID计算完整路径
	packagePath, err := s.calculatePackagePath(ctx, req.ServiceTree)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate package path: %w", err)
	}

	// 创建包目录
	packageDir := filepath.Join(apiDir, packagePath)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create package directory: %w", err)
	}

	// 生成init_.go文件
	if err := s.generateInitFile(packageDir, req.ServiceTree); err != nil {
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

// calculatePackagePath 计算包路径
func (s *ServiceTreeService) calculatePackagePath(ctx context.Context, serviceTree *dto.ServiceTreeRuntimeData) (string, error) {

	// 这里需要根据父目录ID获取父目录的路径
	// 由于我们没有数据库连接，这里简化处理
	// 实际实现中，应该通过NATS消息查询父目录信息
	// 或者维护一个内存中的目录结构映射

	// 简化实现：假设父目录路径已经包含在FullNamePath中
	// 去掉开头的"/"并转换为包路径
	subPath := serviceTree.GetSubPath()

	// 清理路径：去掉首尾斜杠，确保不会返回 "/" 或空字符串
	cleanPath := strings.Trim(subPath, "/")
	if cleanPath == "" {
		// 如果清理后为空，使用 Code 作为 fallback
		return serviceTree.Code, nil
	}

	return cleanPath, nil
}

// generateInitFile 生成init_.go文件
func (s *ServiceTreeService) generateInitFile(packageDir string, serviceTree *dto.ServiceTreeRuntimeData) error {
	// 计算RouterGroup
	routerGroup := serviceTree.GetSubPath()
	if routerGroup == "" {
		routerGroup = "/" + serviceTree.Code
	}

	// 生成init_.go文件内容（新格式：使用PackageContext）
	content := fmt.Sprintf(`package %s

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "%s",
}
`, serviceTree.Code, routerGroup)

	// 写入文件
	initFilePath := filepath.Join(packageDir, "init_.go")
	logger.Infof(context.Background(), "[ServiceTreeService] Generate init file with packageContext: %s", initFilePath)
	if err := os.WriteFile(initFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write init file: %w", err)
	}

	return nil
}

// DeleteServiceTree 删除服务目录
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, user, app, serviceTreeName string) error {
	logger.Infof(ctx, "[ServiceTreeService] Deleting service tree: %s/%s/%s", user, app, serviceTreeName)

	// 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, user, app)
	apiDir := filepath.Join(appDir, "code", "api")
	packageDir := filepath.Join(apiDir, serviceTreeName)

	// 删除目录
	if err := os.RemoveAll(packageDir); err != nil {
		return fmt.Errorf("failed to delete package directory: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree deleted successfully: %s", packageDir)
	return nil
}

// RenameServiceTree 重命名服务目录（旧方法，保留兼容性）
func (s *ServiceTreeService) RenameServiceTree(ctx context.Context, user, app, oldName, newName string) error {
	logger.Infof(ctx, "[ServiceTreeService] Updating service tree: %s/%s/%s -> %s", user, app, oldName, newName)

	// 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, user, app)
	apiDir := filepath.Join(appDir, "code", "api")
	oldPackageDir := filepath.Join(apiDir, oldName)
	newPackageDir := filepath.Join(apiDir, newName)

	// 重命名目录
	if err := os.Rename(oldPackageDir, newPackageDir); err != nil {
		return fmt.Errorf("failed to rename package directory: %w", err)
	}

	// 更新init_.go文件中的包名
	initFilePath := filepath.Join(newPackageDir, "init_.go")
	if err := s.updateInitFilePackageName(initFilePath, newName); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to update init file package name: %v", err)
		// 不返回错误，因为目录重命名已经成功
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree updated successfully: %s -> %s", oldPackageDir, newPackageDir)
	return nil
}

// updateInitFilePackageName 更新init_.go文件中的包名
func (s *ServiceTreeService) updateInitFilePackageName(initFilePath, newPackageName string) error {
	// 读取文件内容
	content, err := os.ReadFile(initFilePath)
	if err != nil {
		return fmt.Errorf("failed to read init file: %w", err)
	}

	// 替换包名
	oldContent := string(content)
	newContent := strings.Replace(oldContent, "package "+strings.Split(oldContent, "\n")[0], "package "+newPackageName, 1)

	// 写回文件
	if err := os.WriteFile(initFilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write init file: %w", err)
	}

	return nil
}

// updateMainFileImports 更新main文件，添加新包的import（简化版本）
func (s *ServiceTreeService) updateMainFileImports(ctx context.Context, user, app, packagePath string) error {
	logger.Infof(ctx, "[ServiceTreeService] Updating main file imports for package: %s", packagePath)

	// 构建main文件路径
	appDir := filepath.Join(s.config.AppDir.BasePath, user, app)
	mainFilePath := filepath.Join(appDir, "code", "cmd", "app", "main.go")

	// 检查main文件是否存在
	if _, err := os.Stat(mainFilePath); os.IsNotExist(err) {
		return fmt.Errorf("main file does not exist: %s", mainFilePath)
	}

	// 读取main文件内容
	content, err := os.ReadFile(mainFilePath)
	if err != nil {
		return fmt.Errorf("failed to read main file: %w", err)
	}

	contentStr := string(content)

	// 找到 app SDK 的 import 行
	appSDKImport := `"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"`
	if !strings.Contains(contentStr, appSDKImport) {
		return fmt.Errorf("cannot find app SDK import in main file")
	}

	// 生成新的import语句
	// 清理 packagePath：去掉首尾斜杠，确保不会生成有尾随斜杠的 import
	cleanPackagePath := strings.Trim(packagePath, "/")
	if cleanPackagePath == "" {
		// 如果 packagePath 为空，跳过（不应该为空，但防止错误）
		logger.Warnf(ctx, "[ServiceTreeService] Package path is empty, skipping import update")
		return nil
	}
	newImport := fmt.Sprintf(`_ "github.com/ai-agent-os/ai-agent-os/namespace/%s/%s/code/api/%s"`, user, app, cleanPackagePath)

	// 检查import是否已存在
	if strings.Contains(contentStr, newImport) {
		logger.Infof(ctx, "[ServiceTreeService] Import already exists: %s", newImport)
		return nil
	}

	// 根据 app SDK import 行分割内容
	parts := strings.Split(contentStr, appSDKImport)
	if len(parts) != 2 {
		return fmt.Errorf("unexpected main file format")
	}

	// 重新组装内容：第一部分 + 新import + app SDK import + 第二部分
	newContent := parts[0] + "\n\t" + newImport + "\n" + appSDKImport + parts[1]

	// 写回main文件
	if err := os.WriteFile(mainFilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write main file: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Successfully added import: %s", newImport)
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

	// 1. 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := filepath.Join(appDir, "code", "api")

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

	// 4. 创建目录映射（用于快速查找父目录）
	createdDirs := make(map[string]bool)
	directoryCount := 0
	createdPaths := make([]string, 0)

	// 5. 遍历所有目录项，递归创建
	for _, item := range sortedItems {
		// 创建目录
		if err := s.createDirectoryRecursively(ctx, apiDir, item, createdDirs); err != nil {
			return nil, fmt.Errorf("创建目录失败 (%s): %w", item.FullCodePath, err)
		}
		createdDirs[item.FullCodePath] = true
		directoryCount++
		createdPaths = append(createdPaths, item.FullCodePath)

		// 生成 init_.go 文件
		if err := s.generateInitFileForPath(ctx, apiDir, item); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 生成 init_.go 失败: path=%s, error=%v",
				item.FullCodePath, err)
			// 不返回错误，因为目录已创建成功
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

	// 1. 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := filepath.Join(appDir, "code", "api")

	// 确保 api 目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 api 目录失败: %w", err)
	}

	// 2. 批量写入文件（req.Files 只包含文件，不需要检查 Type）
	if len(req.Files) == 0 {
		return nil, fmt.Errorf("没有需要写入的文件")
	}

	// 3. 批量写入文件
	writtenPaths := make([]string, 0)                    // FullCodePath 列表
	writtenFilePaths := make([]string, 0)                // 实际文件路径列表（用于回滚）

	for _, item := range req.Files {
		// 从 FullCodePath 提取 package 路径和文件名
		packagePath := extractPackagePath(item.FullCodePath)
		if packagePath == "" {
			s.rollbackFiles(ctx, writtenFilePaths)
			return nil, fmt.Errorf("无效的文件路径: %s", item.FullCodePath)
		}

		fileName := item.FileName
		if fileName == "" {
			// 从 FullCodePath 提取文件名
			pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
			if len(pathParts) == 0 {
				s.rollbackFiles(ctx, writtenFilePaths)
				return nil, fmt.Errorf("无法从路径提取文件名: %s", item.FullCodePath)
			}
			fileName = pathParts[len(pathParts)-1]
			// 去掉文件扩展名
			if ext := filepath.Ext(fileName); ext != "" {
				fileName = strings.TrimSuffix(fileName, ext)
			}
		}

		// 确保目录存在
		packageDir := filepath.Join(apiDir, packagePath)
		if err := os.MkdirAll(packageDir, 0755); err != nil {
			s.rollbackFiles(ctx, writtenFilePaths)
			return nil, fmt.Errorf("创建目录失败: %w", err)
		}

		// 构建文件路径
		fileExt := getFileExtension(item.FileType)
		filePath := filepath.Join(packageDir, fileName+fileExt)

		// 写入文件内容
		if err := os.WriteFile(filePath, []byte(item.Content), 0644); err != nil {
			s.rollbackFiles(ctx, writtenFilePaths)
			return nil, fmt.Errorf("写入文件失败 (%s): %w", item.FullCodePath, err)
		}

		writtenPaths = append(writtenPaths, item.FullCodePath)
		writtenFilePaths = append(writtenFilePaths, filePath)
		logger.Infof(ctx, "[ServiceTreeService] 文件写入成功: %s", filePath)
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量写文件完成: fileCount=%d", len(writtenPaths))

	// 4. 编译应用并获取 diff（需要 appManageService）
	if s.appManageService == nil {
		return nil, fmt.Errorf("appManageService 未设置，无法编译应用")
	}

	// 获取当前版本
	vm := appPkg.NewVersionManager(filepath.Join(s.config.AppDir.BasePath, req.User), req.App)
	oldVersion, err := vm.GetCurrentVersion()
	if err != nil {
		logger.Warnf(ctx, "[BatchWriteFiles] 获取当前版本失败: %v，使用 unknown", err)
		oldVersion = "unknown"
	}

	// 编译应用
	sourceDir := filepath.Join(appDir, "code/cmd/app")
	outputDir := filepath.Join(appDir, s.config.Build.OutputDir)

	buildOpts := &BuildOpts{
		SourceDir:        sourceDir,
		OutputDir:        outputDir,
		Platform:         s.config.Build.Platform,
		BinaryNameFormat: s.config.Build.BinaryNameFormat,
	}

	buildResult, err := s.appManageService.BuildApp(ctx, req.User, req.App, buildOpts)
	if err != nil {
		// 编译失败时，回滚已写入的文件
		logger.Warnf(ctx, "[BatchWriteFiles] 编译失败，开始回滚已写入的文件: fileCount=%d", len(writtenFilePaths))
		s.rollbackFiles(ctx, writtenFilePaths)
		return nil, fmt.Errorf("编译应用失败: %w", err)
	}

	newVersion := buildResult.Version

	// 5. 更新版本信息
	metadataDir := filepath.Join(appDir, "workplace/metadata")
	versionFile := filepath.Join(metadataDir, "version.json")

	// 检查版本文件是否存在，如果不存在则创建
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		logger.Infof(ctx, "[BatchWriteFiles] Version file not found, creating initial version file...")
		if err := s.appManageService.createVersionFiles(metadataDir, req.User, req.App); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 创建版本文件失败: %v，继续执行", err)
		}
	}

	// 更新版本信息
	if err := s.appManageService.updateVersionJson(appDir, req.User, req.App, newVersion); err != nil {
		logger.Warnf(ctx, "[BatchWriteFiles] 更新版本信息失败: %v，继续执行", err)
	}

	// 6. Git 提交（可选，如果失败不影响主流程）
	var gitCommitHash string
	if hash, err := s.appManageService.commitToGit(ctx, req.User, req.App, newVersion, "", ""); err != nil {
		logger.Warnf(ctx, "[BatchWriteFiles] Git 提交失败: %v，继续执行", err)
	} else {
		gitCommitHash = hash
	}

	// 7. 获取 diff（通过启动应用并获取回调，参考 UpdateApp 的逻辑）
	var diff *dto.DiffData
	if s.appManageService != nil {
		// 创建新版本容器并启动（用于获取 diff）
		waiterChan := s.appManageService.registerStartupWaiter(req.User, req.App, newVersion)
		defer s.appManageService.unregisterStartupWaiter(req.User, req.App, newVersion)

		if s.appManageService.containerService != nil {
			// 创建新版本容器
			appDirRel := filepath.Join(s.config.AppDir.BasePath, req.User, req.App)
			if err := s.appManageService.createVersionContainer(ctx, req.User, req.App, newVersion, appDirRel); err != nil {
				logger.Warnf(ctx, "[BatchWriteFiles] 创建容器失败: %v，继续执行（不获取 diff）", err)
			} else {
				// 等待新版本启动
				logger.Infof(ctx, "[BatchWriteFiles] 等待新版本启动: %s/%s/%s", req.User, req.App, newVersion)
				select {
				case <-waiterChan:
					logger.Infof(ctx, "[BatchWriteFiles] ✅ 新版本启动成功: %s/%s/%s", req.User, req.App, newVersion)

					// 发送更新回调请求获取 diff
					updateCallbackResponse, callbackErr := s.appManageService.sendUpdateCallbackAndWait(ctx, req.User, req.App, newVersion)
					if callbackErr != nil {
						logger.Warnf(ctx, "[BatchWriteFiles] ❌ 获取 diff 失败: %v", callbackErr)
					} else {
						logger.Infof(ctx, "[BatchWriteFiles] ✅ 获取 diff 成功: %+v", updateCallbackResponse)
						// 类型断言，将 interface{} 转换为 *dto.DiffData
						if diffData, ok := updateCallbackResponse.Data.(*dto.DiffData); ok {
							diff = diffData
						} else {
							logger.Warnf(ctx, "[BatchWriteFiles] diff 数据格式不正确，期望 *dto.DiffData，实际类型: %T", updateCallbackResponse.Data)
						}
					}

					// 停止并删除临时容器（因为我们只是用来获取 diff）
					if err := s.appManageService.stopOldVersionContainer(ctx, req.User, req.App, newVersion); err != nil {
						logger.Warnf(ctx, "[BatchWriteFiles] 停止临时容器失败: %v", err)
					}
				case <-time.After(60 * time.Second):
					logger.Warnf(ctx, "[BatchWriteFiles] ⚠️ 等待新版本启动超时，不获取 diff")
				}
			}
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量写文件并编译完成: oldVersion=%s, newVersion=%s", oldVersion, newVersion)

	return &dto.BatchWriteFilesRuntimeResp{
		FileCount:     len(writtenPaths),
		WrittenPaths:  writtenPaths,
		Diff:          diff,
		OldVersion:    oldVersion,
		NewVersion:    newVersion,
		GitCommitHash: gitCommitHash,
	}, nil
}

// rollbackFiles 回滚已写入的文件（内部方法，失败时调用）
// filePaths: 实际的文件路径列表（绝对路径）
func (s *ServiceTreeService) rollbackFiles(ctx context.Context, filePaths []string) {
	logger.Warnf(ctx, "[ServiceTreeService] 开始回滚已写入的文件: fileCount=%d", len(filePaths))

	deletedCount := 0
	for _, filePath := range filePaths {
		if err := os.Remove(filePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.Errorf(ctx, "[ServiceTreeService] 删除文件失败: file=%s, error=%v", filePath, err)
		} else {
			deletedCount++
			logger.Infof(ctx, "[ServiceTreeService] 已删除文件: %s", filePath)
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] 文件回滚完成: deletedCount=%d, totalCount=%d", deletedCount, len(filePaths))
}

// extractPackagePath 从 FullCodePath 提取 package 路径（去掉 /user/app 前缀）
func extractPackagePath(fullCodePath string) string {
	// 去掉首尾斜杠并分割路径
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) <= 2 {
		return ""
	}
	// 返回去掉前两个部分（user/app）后的路径
	subParts := pathParts[2:]
	return strings.Join(subParts, "/")
}

// getParentPath 获取父目录路径
func getParentPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) <= 3 {
		return "" // 没有父目录（已经是根目录）
	}
	// 去掉最后一部分，重新组合
	parentParts := pathParts[:len(pathParts)-1]
	return "/" + strings.Join(parentParts, "/")
}

// createDirectoryRecursively 递归创建目录
func (s *ServiceTreeService) createDirectoryRecursively(
	ctx context.Context,
	apiDir string,
	item *dto.DirectoryTreeItem,
	createdDirs map[string]bool,
) error {
	// 从 FullCodePath 提取 package 路径（去掉 /user/app 前缀）
	packagePath := extractPackagePath(item.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的目录路径: %s", item.FullCodePath)
	}

	packageDir := filepath.Join(apiDir, packagePath)

	// 如果目录已创建，跳过
	if createdDirs[item.FullCodePath] {
		return nil
	}

	// 确保父目录存在（递归创建）
	parentPath := getParentPath(item.FullCodePath)
	if parentPath != "" && !createdDirs[parentPath] {
		// 递归创建父目录（直接创建目录结构，不依赖 items）
		parentPackagePath := extractPackagePath(parentPath)
		if parentPackagePath != "" {
			parentPackageDir := filepath.Join(apiDir, parentPackagePath)
			if err := os.MkdirAll(parentPackageDir, 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %w", err)
			}
			createdDirs[parentPath] = true
		}
	}

	// 创建目录
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	return nil
}

// createFileForItem 创建文件
func (s *ServiceTreeService) createFileForItem(
	ctx context.Context,
	apiDir string,
	item *dto.DirectoryTreeItem,
	createdDirs map[string]bool,
) error {
	// 从 FullCodePath 提取 package 路径和文件名
	packagePath := extractPackagePath(item.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的文件路径: %s", item.FullCodePath)
	}

	fileName := item.FileName
	if fileName == "" {
		// 从 FullCodePath 提取文件名
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		if len(pathParts) == 0 {
			return fmt.Errorf("无法从路径提取文件名: %s", item.FullCodePath)
		}
		fileName = pathParts[len(pathParts)-1]
		// 去掉文件扩展名
		if ext := filepath.Ext(fileName); ext != "" {
			fileName = strings.TrimSuffix(fileName, ext)
		}
	}

	// 确保目录存在
	packageDir := filepath.Join(apiDir, packagePath)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件路径
	fileExt := getFileExtension(item.FileType)
	filePath := filepath.Join(packageDir, fileName+fileExt)

	// 写入文件内容
	if err := os.WriteFile(filePath, []byte(item.Content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 文件创建成功: %s", filePath)
	return nil
}

// getFileExtension 获取文件扩展名
func getFileExtension(fileType string) string {
	if fileType == "" {
		return ".go" // 默认扩展名
	}
	// 如果 fileType 已经包含点，直接返回
	if strings.HasPrefix(fileType, ".") {
		return fileType
	}
	return "." + fileType
}

// generateInitFileForPath 为指定路径生成 init_.go 文件
func (s *ServiceTreeService) generateInitFileForPath(
	ctx context.Context,
	apiDir string,
	item *dto.DirectoryTreeItem,
) error {
	// 从 FullCodePath 提取 package 路径
	packagePath := extractPackagePath(item.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的目录路径: %s", item.FullCodePath)
	}

	packageDir := filepath.Join(apiDir, packagePath)

	// 提取目录代码（路径的最后一部分）
	pathParts := strings.Split(packagePath, "/")
	dirCode := pathParts[len(pathParts)-1]

	// 计算 RouterGroup（从 FullCodePath 提取，去掉 /user/app 前缀）
	routerGroup := extractPackagePath(item.FullCodePath)
	if routerGroup == "" {
		routerGroup = "/" + dirCode
	} else {
		routerGroup = "/" + routerGroup
	}

	// 生成 init_.go 文件内容
	content := fmt.Sprintf(`package %s

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "%s",
}
`, dirCode, routerGroup)

	// 写入文件
	initFilePath := filepath.Join(packageDir, "init_.go")
	if err := os.WriteFile(initFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 init_.go 文件失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 生成 init_.go 文件: %s", initFilePath)
	return nil
}

// sortItemsByPath 按路径排序（确保先创建父目录）
func sortItemsByPath(items []*dto.DirectoryTreeItem) []*dto.DirectoryTreeItem {
	sorted := make([]*dto.DirectoryTreeItem, len(items))
	copy(sorted, items)

	sort.Slice(sorted, func(i, j int) bool {
		// 先按路径长度排序（短的在前）
		lenI := len(sorted[i].FullCodePath)
		lenJ := len(sorted[j].FullCodePath)
		if lenI != lenJ {
			return lenI < lenJ
		}
		// 长度相同，目录优先于文件
		if sorted[i].Type != sorted[j].Type {
			return sorted[i].Type == "directory"
		}
		// 类型相同，按路径字符串排序
		return sorted[i].FullCodePath < sorted[j].FullCodePath
	})

	return sorted
}

// UpdateServiceTreeMetadata 更新服务树元数据（已废弃）
// 注意：此方法已废弃，请使用 BatchCreateDirectoryTree 和 BatchWriteFiles 替代
// 此方法保留用于向后兼容，但不应再使用
func (s *ServiceTreeService) UpdateServiceTreeMetadata(
	ctx context.Context,
	req interface{}, // 使用 interface{} 避免依赖已删除的 DTO
) (interface{}, error) {
	return nil, fmt.Errorf("UpdateServiceTreeMetadata 已废弃，请使用 BatchCreateDirectoryTree 和 BatchWriteFiles 替代")
}
