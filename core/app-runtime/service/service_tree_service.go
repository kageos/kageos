package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// ServiceTreeService 服务目录管理服务
type ServiceTreeService struct {
	config *config.AppManageServiceConfig
}

// NewServiceTreeService 创建服务目录管理服务
func NewServiceTreeService(config *config.AppManageServiceConfig) *ServiceTreeService {
	return &ServiceTreeService{
		config: config,
	}
}

// CreateServiceTree 创建服务目录
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeRuntimeReq) (*dto.CreateServiceTreeRuntimeResp, error) {
	logger.Infof(ctx, "[ServiceTreeService] Creating service tree: %s/%s/%s", req.User, req.App, req.ServiceTree.Name)

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

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully: %s", packageDir)

	return &dto.CreateServiceTreeRuntimeResp{
		User:        req.User,
		App:         req.App,
		ServiceTree: req.ServiceTree.Name,
		Status:      "created",
		Message:     fmt.Sprintf("Service tree created at %s", packageDir),
	}, nil
}

// calculatePackagePath 计算包路径
func (s *ServiceTreeService) calculatePackagePath(ctx context.Context, serviceTree *dto.ServiceTreeRuntimeData) (string, error) {
	// 如果父目录ID为0，说明是根目录
	if serviceTree.ParentID == 0 {
		return serviceTree.Name, nil
	}

	// 这里需要根据父目录ID获取父目录的路径
	// 由于我们没有数据库连接，这里简化处理
	// 实际实现中，应该通过NATS消息查询父目录信息
	// 或者维护一个内存中的目录结构映射

	// 简化实现：假设父目录路径已经包含在FullNamePath中
	// 去掉开头的"/"并转换为包路径
	path := strings.TrimPrefix(serviceTree.FullNamePath, "/")
	path = strings.ReplaceAll(path, "/", string(filepath.Separator))

	return path, nil
}

// generateInitFile 生成init_.go文件
func (s *ServiceTreeService) generateInitFile(packageDir string, serviceTree *dto.ServiceTreeRuntimeData) error {
	// 计算RouterGroup
	routerGroup := serviceTree.FullNamePath
	if routerGroup == "" {
		routerGroup = "/" + serviceTree.Name
	}

	// 生成init_.go文件内容
	content := fmt.Sprintf(`package %s

import "fmt"

const (
	RouterGroup = "%s"
)

func WithCurrentRouterGroup(router string) string {
	return fmt.Sprintf("%%s/%%s", RouterGroup, router)
}
`, serviceTree.Name, routerGroup)

	// 写入文件
	initFilePath := filepath.Join(packageDir, "init_.go")
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

// UpdateServiceTree 更新服务目录
func (s *ServiceTreeService) UpdateServiceTree(ctx context.Context, user, app, oldName, newName string) error {
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
