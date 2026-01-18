package service

import (
	"context"
	"errors"
	"fmt"

	appmodel "github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

const (
	// SystemUsername 系统用户名
	SystemUsername = "system"
	// SystemUserEmail 系统用户邮箱
	SystemUserEmail = "system@ai-agent-os.local"
)

// InitSystemWorkspace 初始化系统工作空间（只初始化 official 工作空间）
// 在 app-server 启动时调用，确保 system/official 工作空间存在
// 注意：system 用户应该在 hr-server 中初始化，这里只初始化工作空间
func InitSystemWorkspace(ctx context.Context, appService *AppService) error {
	logger.Infof(ctx, "[SystemWorkspace] 开始初始化系统工作空间...")

	// 初始化内置应用（通过 AppService 创建，会调用 runtime）
	// 设置 context 的 requestUser 为 system
	systemCtx := context.WithValue(ctx, contextx.RequestUserHeader, SystemUsername)
	if err := initSystemApps(systemCtx, appService); err != nil {
		return fmt.Errorf("初始化内置应用失败: %w", err)
	}

	logger.Infof(ctx, "[SystemWorkspace] 系统工作空间初始化完成")
	return nil
}

// initSystemApps 初始化内置应用
// 只初始化一个 official 官方库工作空间
// 通过 AppService.CreateApp 创建应用，会调用 runtime
func initSystemApps(ctx context.Context, appService *AppService) error {
	appRepo := appService.appRepo

	// 只创建一个 official 官方库工作空间
	appCode := "official"
	appName := "官方库"
	
	// 检查应用是否已存在
	existingApp, err := appRepo.GetAppByUserName(SystemUsername, appCode)
	if err == nil && existingApp != nil {
		// 已存在，检查类型是否正确
		if existingApp.Type != appmodel.AppTypeSystem {
			// 更新类型为系统空间
			existingApp.Type = appmodel.AppTypeSystem
			if err := appRepo.UpdateApp(existingApp); err != nil {
				return fmt.Errorf("更新应用 %s/%s 类型失败: %w", SystemUsername, appCode, err)
			}
			logger.Infof(ctx, "[SystemWorkspace] 已更新应用类型: %s/%s", SystemUsername, appCode)
		} else {
			logger.Infof(ctx, "[SystemWorkspace] 应用已存在: %s/%s", SystemUsername, appCode)
		}
		return nil
	}
	
	// 如果错误不是 record not found，说明是其他错误，需要返回
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询应用 %s/%s 失败: %w", SystemUsername, appCode, err)
	}
	
	// err == gorm.ErrRecordNotFound，说明应用不存在，需要创建

	// 不存在，通过 AppService 创建应用（会调用 runtime）
	isPublic := true
	createReq := &dto.CreateAppReq{
		User:     SystemUsername,
		Code:     appCode,
		Name:     appName,
		IsPublic: &isPublic,
		Admins:   SystemUsername, // 管理员为 system
	}

	_, err = appService.CreateApp(ctx, createReq)
	if err != nil {
		return fmt.Errorf("创建应用 %s/%s 失败: %w", SystemUsername, appCode, err)
	}

	// 创建后更新应用类型为系统空间
	// 注意：CreateApp 返回的应用可能还没有 Type 字段，需要再次查询并更新
	createdApp, err := appRepo.GetAppByUserName(SystemUsername, appCode)
	if err != nil {
		return fmt.Errorf("查询刚创建的应用失败: %w", err)
	}
	if createdApp.Type != appmodel.AppTypeSystem {
		createdApp.Type = appmodel.AppTypeSystem
		if err := appRepo.UpdateApp(createdApp); err != nil {
			return fmt.Errorf("更新应用类型为系统空间失败: %w", err)
		}
	}

	logger.Infof(ctx, "[SystemWorkspace] 已创建应用: %s/%s", SystemUsername, appCode)
	return nil
}
