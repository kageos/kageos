package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type AppService struct {
	appRuntime *AppRuntime
	userRepo   *repository.UserRepository
	appRepo    *repository.AppRepository
}

// NewAppService 创建 AppService（依赖注入）
func NewAppService(appRuntime *AppRuntime, userRepo *repository.UserRepository, appRepo *repository.AppRepository) *AppService {
	return &AppService{
		appRuntime: appRuntime,
		userRepo:   userRepo,
		appRepo:    appRepo,
	}
}

// CreateApp 创建应用
func (a *AppService) CreateApp(ctx context.Context, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	// 从请求体中获取租户用户信息（应用所有者）
	tenantUser := req.User
	if tenantUser == "" {
		return nil, fmt.Errorf("租户用户信息不能为空")
	}

	// 从 context 中获取请求用户信息（实际发起请求的用户）
	requestUser := contextx.GetRequestUser(ctx)
	if requestUser == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 根据租户用户获取主机和 NATS 信息
	user, err := a.userRepo.GetUserByUsernameWithHostAndNats(tenantUser)
	if err != nil {
		return nil, fmt.Errorf("获取租户用户 %s 的主机信息失败: %w", tenantUser, err)
	}

	// 创建包含用户信息的请求对象（内部使用）
	createReqWithUser := struct {
		User string `json:"user"`
		App  string `json:"app"`
	}{
		User: tenantUser, // 明确传递租户用户
		App:  req.App,
	}

	resp, err := a.appRuntime.CreateApp(ctx, user.HostID, createReqWithUser)
	if err != nil {
		return nil, err
	}

	// 写入数据库记录
	app := model.App{
		Base: models.Base{
			CreatedBy: requestUser, // 记录实际请求用户（谁发起的请求）
		},
		Version: "v1",
		Name:    req.App,
		User:    tenantUser, // 记录租户用户（应用所有者）
		NatsID:  user.Host.NatsID,
		HostID:  user.Host.ID,
		Status:  "已启用",
	}
	err = a.appRepo.CreateApp(&app)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// UpdateApp 更新应用
func (a *AppService) UpdateApp(ctx context.Context, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	// 根据应用信息获取 NATS 连接，而不是根据当前用户
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 更新应用，使用应用所属的 HostID
	resp, err := a.appRuntime.UpdateApp(ctx, app.HostID, req)
	if err != nil {
		return nil, err
	}

	// 更新数据库中的版本信息
	app.Version = resp.NewVersion
	err = a.appRepo.UpdateApp(app)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// RequestApp 请求应用
func (a *AppService) RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}
	req.Version = app.Version
	resp, err := a.appRuntime.RequestApp(ctx, app.NatsID, req)
	if err != nil {
		return nil, err
	}
	resp.Version = req.Version
	return resp, nil
}

// DeleteApp 删除应用
func (a *AppService) DeleteApp(ctx context.Context, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	// 根据应用信息获取 NATS 连接
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 删除应用
	resp, err := a.appRuntime.DeleteApp(ctx, app.HostID, req)
	if err != nil {
		return nil, err
	}

	// 删除数据库记录
	err = a.appRepo.DeleteAppAndVersions(req.User, req.App)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
