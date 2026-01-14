package permission

import (
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/service"
)

func init() {
	// 注册权限服务实现（企业版）
	// 只有在企业版 License 激活时才会调用
	enterprise.RegisterPermissionService(&service.PermissionServiceImpl{})
}

