package permission

import (
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/service"
)

func init() {
	// 注册权限服务实现（企业版）
	// 这里在进程启动时注册实现；是否可用由 License 功能位控制
	enterprise.RegisterPermissionService(&service.PermissionServiceImpl{})
}
