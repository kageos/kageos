package server

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	// ⭐ 导入企业版实现，触发 init() 函数注册
	_ "github.com/ai-agent-os/ai-agent-os/enterprise_impl/operatelog"
	_ "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission"
)

func (s *Server) logEnterpriseLicenseStatus() *license.Manager {
	ctx := s.ctx
	licenseMgr := license.GetManager()
	lic := licenseMgr.GetLicense()

	if lic == nil || !lic.IsValid() {
		logger.Infof(ctx, "[Enterprise] No valid license detected, initializing enterprise implementations in disabled mode")
	} else {
		logger.Infof(ctx, "[Enterprise] License detected: Edition=%s, Customer=%s",
			lic.Edition, lic.Customer)
	}

	return licenseMgr
}

// initEnterprise 初始化企业功能
// 说明：
//   - 企业实现统一在启动时完成初始化，避免运行时 License 热更新后出现“功能已开启但实现未初始化”的状态
//   - License 功能位仅用于控制“能否访问/能否使用”，不再决定是否初始化底层实现
func (s *Server) initEnterprise() error {
	ctx := s.ctx
	licenseMgr := s.logEnterpriseLicenseStatus()

	// 统一初始化操作日志实现；是否可用由 Feature 开关控制
	logger.Infof(ctx, "[Enterprise] Initializing operate log implementation...")
	if err := enterprise.InitOperateLogger(&enterprise.InitOptions{DB: s.db}); err != nil {
		return err
	}
	s.operateLogger = enterprise.GetOperateLogger()
	if licenseMgr.HasFeature(enterprise.FeatureOperateLog) {
		logger.Infof(ctx, "[Enterprise] Operate log feature initialized and enabled")
	} else {
		logger.Infof(ctx, "[Enterprise] Operate log implementation initialized, feature currently disabled by license")
	}

	// 统一初始化权限实现；是否可用由 Feature 开关控制
	logger.Infof(ctx, "[Enterprise] Initializing permission implementation...")
	// 企业版 ApplyPermissionByResourcePath 需要 AppIDResolver，由 app-server 注入（依赖 appRepo）
	if s.appRepo == nil {
		s.appRepo = repository.NewAppRepository(s.db)
	}
	if err := enterprise.InitPermissionService(&enterprise.InitOptions{
		DB:            s.db,
		AppIDResolver: service.NewAppIDResolver(s.appRepo),
	}); err != nil {
		return err
	}
	if licenseMgr.HasFeature(enterprise.FeaturePermission) {
		logger.Infof(ctx, "[Enterprise] Permission feature initialized and enabled")
	} else {
		logger.Infof(ctx, "[Enterprise] Permission implementation initialized, feature currently disabled by license")
	}

	// 后续可以添加更多功能的初始化，例如：
	// if licenseMgr.HasFeature(enterprise.FeatureWorkflow) {
	//     // 初始化工作流功能
	// }

	return nil
}

// initSchedulerEnterprise 初始化调度器需要的企业功能。
// scheduler 仅负责执行与投递，不需要权限服务与 HTTP 查询能力。
func (s *Server) initSchedulerEnterprise() error {
	ctx := s.ctx
	licenseMgr := s.logEnterpriseLicenseStatus()

	logger.Infof(ctx, "[Enterprise] Initializing scheduler operate log implementation...")
	if err := enterprise.InitOperateLogger(&enterprise.InitOptions{DB: s.db}); err != nil {
		return err
	}
	s.operateLogger = enterprise.GetOperateLogger()
	if licenseMgr.HasFeature(enterprise.FeatureOperateLog) {
		logger.Infof(ctx, "[Enterprise] Scheduler operate log feature initialized and enabled")
	} else {
		logger.Infof(ctx, "[Enterprise] Scheduler operate log implementation initialized, feature currently disabled by license")
	}

	return nil
}
