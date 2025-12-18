package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
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

// BatchCreateDirectoryTree 批量创建目录树（递归创建目录和文件）
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

	// 2. 按路径排序，确保先创建父目录，再创建子目录和文件
	sortedItems := sortItemsByPath(req.Items)

	// 3. 创建目录映射（用于快速查找父目录）
	createdDirs := make(map[string]bool)
	directoryCount := 0
	fileCount := 0
	createdPaths := make([]string, 0)

	// 4. 遍历所有项，递归创建
	for _, item := range sortedItems {
		if item.Type == "directory" {
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
		} else if item.Type == "file" {
			// 创建文件
			if err := s.createFileForItem(ctx, apiDir, item, createdDirs); err != nil {
				return nil, fmt.Errorf("创建文件失败 (%s): %w", item.FullCodePath, err)
			}
			fileCount++
			createdPaths = append(createdPaths, item.FullCodePath)
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量创建目录树完成: directoryCount=%d, fileCount=%d",
		directoryCount, fileCount)

	return &dto.BatchCreateDirectoryTreeRuntimeResp{
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		CreatedPaths:   createdPaths,
	}, nil
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

// UpdateServiceTree 更新服务树（通用方法，支持新增、更新、删除节点，返回 diff 信息）
func (s *ServiceTreeService) UpdateServiceTree(
	ctx context.Context,
	req *dto.UpdateServiceTreeRuntimeReq,
) (*dto.UpdateServiceTreeRuntimeResp, error) {
	logger.Infof(ctx, "[ServiceTreeService] 开始更新服务树: user=%s, app=%s, nodeCount=%d",
		req.User, req.App, len(req.Nodes))

	// 1. 构建应用目录路径
	appDir := filepath.Join(s.config.AppDir.BasePath, req.User, req.App)
	apiDir := filepath.Join(appDir, "code", "api")

	// 确保 api 目录存在
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 api 目录失败: %w", err)
	}

	// 2. 分离目录节点和文件节点
	directoryNodes := make([]*dto.ServiceTreeNode, 0)
	fileNodes := make([]*dto.ServiceTreeNode, 0)
	hasFileChanges := false

	for _, node := range req.Nodes {
		if node.Type == "directory" {
			directoryNodes = append(directoryNodes, node)
		} else if node.Type == "file" {
			fileNodes = append(fileNodes, node)
			// 如果有文件变更（新增、更新、删除），需要编译
			if node.Operation == dto.ServiceTreeNodeOpAdd ||
				node.Operation == dto.ServiceTreeNodeOpUpdate ||
				node.Operation == dto.ServiceTreeNodeOpDelete {
				hasFileChanges = true
			}
		}
	}

	// 3. 处理目录节点（只创建目录，不处理文件）
	directoryCount := 0
	createdDirs := make(map[string]bool)

	// 按路径排序，确保先创建父目录
	sortedDirNodes := sortServiceTreeNodesByPath(directoryNodes)

	for _, node := range sortedDirNodes {
		if node.Operation == dto.ServiceTreeNodeOpAdd {
			// 创建目录
			if err := s.createDirectoryForNode(ctx, apiDir, node, createdDirs); err != nil {
				return nil, fmt.Errorf("创建目录失败 (%s): %w", node.FullCodePath, err)
			}
			createdDirs[node.FullCodePath] = true
			directoryCount++

			// 生成 init_.go 文件
			if err := s.generateInitFileForServiceTreeNode(ctx, apiDir, node); err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 生成 init_.go 失败: path=%s, error=%v",
					node.FullCodePath, err)
			}
		} else if node.Operation == dto.ServiceTreeNodeOpDelete {
			// 删除目录
			if err := s.deleteDirectoryForNode(ctx, apiDir, node); err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 删除目录失败: path=%s, error=%v",
					node.FullCodePath, err)
			}
		}
		// update 操作暂不支持（目录更新通常不需要）
	}

	// 4. 处理文件节点
	fileCount := 0
	for _, node := range fileNodes {
		switch node.Operation {
		case dto.ServiceTreeNodeOpAdd, dto.ServiceTreeNodeOpUpdate:
			// 创建或更新文件
			if err := s.createOrUpdateFileForNode(ctx, apiDir, node, createdDirs); err != nil {
				return nil, fmt.Errorf("创建/更新文件失败 (%s): %w", node.FullCodePath, err)
			}
			fileCount++
		case dto.ServiceTreeNodeOpDelete:
			// 删除文件
			if err := s.deleteFileForNode(ctx, apiDir, node); err != nil {
				logger.Warnf(ctx, "[ServiceTreeService] 删除文件失败: path=%s, error=%v",
					node.FullCodePath, err)
			}
			fileCount++
		}
	}

	// 5. 如果有文件变更，编译应用并获取 diff
	var diff *dto.DiffData
	var oldVersion, newVersion, gitCommitHash string

	if hasFileChanges && s.appManageService != nil {
		logger.Infof(ctx, "[ServiceTreeService] 检测到文件变更，开始编译应用并获取 diff")

		// 调用 AppManageService 的 UpdateApp 来编译和获取 diff
		// 注意：这里不传递 CreateFunctions，因为文件已经通过上面的逻辑创建了
		updateResp, err := s.appManageService.UpdateApp(ctx, req.User, req.App, nil, nil, "", "")
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 编译应用失败: %v", err)
			return nil, fmt.Errorf("编译应用失败: %w", err)
		}

		// 提取 diff 信息
		if updateResp.Diff != nil {
			// 尝试将 interface{} 转换为 *dto.DiffData
			// 注意：UpdateAppResp.Diff 是 interface{} 类型，需要类型断言或 JSON 转换
			if diffData, ok := updateResp.Diff.(*dto.DiffData); ok {
				diff = diffData
			} else {
				// 如果类型不匹配，尝试通过 JSON 序列化/反序列化转换
				diffJSON, err := json.Marshal(updateResp.Diff)
				if err != nil {
					logger.Warnf(ctx, "[ServiceTreeService] 序列化 diff 失败: %v", err)
				} else {
					var diffData dto.DiffData
					if err := json.Unmarshal(diffJSON, &diffData); err != nil {
						logger.Warnf(ctx, "[ServiceTreeService] 反序列化 diff 失败: %v", err)
					} else {
						diff = &diffData
					}
				}
			}
		}

		oldVersion = updateResp.OldVersion
		newVersion = updateResp.NewVersion
		gitCommitHash = updateResp.GitCommitHash
	}

	logger.Infof(ctx, "[ServiceTreeService] 更新服务树完成: directoryCount=%d, fileCount=%d, hasDiff=%v",
		directoryCount, fileCount, diff != nil)

	return &dto.UpdateServiceTreeRuntimeResp{
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		Diff:           diff,
		OldVersion:     oldVersion,
		NewVersion:     newVersion,
		GitCommitHash:  gitCommitHash,
	}, nil
}

// 辅助方法：处理目录和文件节点
func (s *ServiceTreeService) createDirectoryForNode(
	ctx context.Context,
	apiDir string,
	node *dto.ServiceTreeNode,
	createdDirs map[string]bool,
) error {
	packagePath := extractPackagePath(node.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的目录路径: %s", node.FullCodePath)
	}

	packageDir := filepath.Join(apiDir, packagePath)

	// 如果目录已创建，跳过
	if createdDirs[node.FullCodePath] {
		return nil
	}

	// 确保父目录存在
	parentPath := getParentPath(node.FullCodePath)
	if parentPath != "" && !createdDirs[parentPath] {
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

func (s *ServiceTreeService) deleteDirectoryForNode(
	ctx context.Context,
	apiDir string,
	node *dto.ServiceTreeNode,
) error {
	packagePath := extractPackagePath(node.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的目录路径: %s", node.FullCodePath)
	}

	packageDir := filepath.Join(apiDir, packagePath)
	return os.RemoveAll(packageDir)
}

func (s *ServiceTreeService) createOrUpdateFileForNode(
	ctx context.Context,
	apiDir string,
	node *dto.ServiceTreeNode,
	createdDirs map[string]bool,
) error {
	packagePath := extractPackagePath(node.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的文件路径: %s", node.FullCodePath)
	}

	fileName := node.FileName
	if fileName == "" {
		pathParts := strings.Split(strings.Trim(node.FullCodePath, "/"), "/")
		if len(pathParts) == 0 {
			return fmt.Errorf("无法从路径提取文件名: %s", node.FullCodePath)
		}
		fileName = pathParts[len(pathParts)-1]
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
	fileExt := getFileExtension(node.FileType)
	filePath := filepath.Join(packageDir, fileName+fileExt)

	// 写入文件内容
	if err := os.WriteFile(filePath, []byte(node.Content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 文件创建/更新成功: %s", filePath)
	return nil
}

func (s *ServiceTreeService) deleteFileForNode(
	ctx context.Context,
	apiDir string,
	node *dto.ServiceTreeNode,
) error {
	packagePath := extractPackagePath(node.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的文件路径: %s", node.FullCodePath)
	}

	fileName := node.FileName
	if fileName == "" {
		pathParts := strings.Split(strings.Trim(node.FullCodePath, "/"), "/")
		if len(pathParts) == 0 {
			return fmt.Errorf("无法从路径提取文件名: %s", node.FullCodePath)
		}
		fileName = pathParts[len(pathParts)-1]
	}

	fileExt := getFileExtension(node.FileType)
	filePath := filepath.Join(apiDir, packagePath, fileName+fileExt)

	return os.Remove(filePath)
}

func (s *ServiceTreeService) generateInitFileForServiceTreeNode(
	ctx context.Context,
	apiDir string,
	node *dto.ServiceTreeNode,
) error {
	packagePath := extractPackagePath(node.FullCodePath)
	if packagePath == "" {
		return fmt.Errorf("无效的目录路径: %s", node.FullCodePath)
	}

	packageDir := filepath.Join(apiDir, packagePath)

	// 提取目录代码
	pathParts := strings.Split(packagePath, "/")
	dirCode := pathParts[len(pathParts)-1]

	// 计算 RouterGroup
	routerGroup := "/" + packagePath

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

	return nil
}

// sortServiceTreeNodesByPath 按路径排序服务树节点
func sortServiceTreeNodesByPath(nodes []*dto.ServiceTreeNode) []*dto.ServiceTreeNode {
	sorted := make([]*dto.ServiceTreeNode, len(nodes))
	copy(sorted, nodes)

	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].FullCodePath) < len(sorted[j].FullCodePath)
	})

	return sorted
}

