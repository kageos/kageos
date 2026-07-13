package v1

import (
	"github.com/kageos/kageos/core/app-server/service"
)

type ServiceTree struct {
	serviceTreeService *service.ServiceTreeService
	teamAccessService  *service.TeamAccessService
}

// NewServiceTree 创建 ServiceTree 处理器（依赖注入）
func NewServiceTree(serviceTreeService *service.ServiceTreeService, teamAccessService *service.TeamAccessService) *ServiceTree {
	return &ServiceTree{
		serviceTreeService: serviceTreeService,
		teamAccessService:  teamAccessService,
	}
}
