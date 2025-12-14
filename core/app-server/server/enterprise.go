package server

import (
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// initEnterprise 初始化企业功能
// 说明：
//   - 社区版：使用空实现，不执行任何操作
//   - 企业版：根据 License 中的功能开关，初始化对应的企业功能
//   - 不同版本支持的功能不同，通过 HasFeature() 精确控制
func (s *Server) initEnterprise() error {
	ctx := s.ctx

	// 获取 License 管理器
	licenseMgr := license.GetManager()
	lic := licenseMgr.GetLicense()

	// 检查是否有有效的 License
	if lic == nil || !lic.IsValid() {
		logger.Infof(ctx, "[Enterprise] Community edition detected, using default implementations")
		// 社区版：使用空实现
		s.operateLogger = enterprise.GetOperateLogger()
		return nil
	}

	// 有有效的 License，根据功能开关初始化企业功能
	logger.Infof(ctx, "[Enterprise] License detected: Edition=%s, Customer=%s",
		lic.Edition, lic.Customer)

	// 初始化操作日志功能（如果 License 支持）
	if licenseMgr.HasOperateLogFeature() {
		logger.Infof(ctx, "[Enterprise] Initializing operate log feature...")
		err := enterprise.InitOperateLogger(&enterprise.InitOptions{DB: s.db})
		if err != nil {
			return err
		}
		s.operateLogger = enterprise.GetOperateLogger()
		logger.Infof(ctx, "[Enterprise] Operate log feature initialized")
	} else {
		logger.Infof(ctx, "[Enterprise] Operate log feature not available in license, using default implementation")
		s.operateLogger = enterprise.GetOperateLogger()
	}

	// 后续可以添加更多功能的初始化，例如：
	// if licenseMgr.HasFeature(enterprise.FeatureWorkflow) {
	//     // 初始化工作流功能
	// }

	return nil
}
