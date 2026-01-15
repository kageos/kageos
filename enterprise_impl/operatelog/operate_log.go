package operatelog

import (
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/enterprise_impl/operatelog/service"
)

func init() {
	// 注册操作日志服务实现（企业版）
	// 只有在企业版 License 激活时才会调用 Init() 方法初始化数据库连接
	enterprise.RegisterOperateLogger(&service.OperateLogService{})
}
