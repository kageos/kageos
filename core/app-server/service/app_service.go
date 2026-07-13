package service

import (
	"context"

	"github.com/kageos/kageos/pkg/appcall"

	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
)

type AppService struct {
	appCall         appRuntimeClient
	appRepo         *repository.AppRepository
	functionRepo    *repository.FunctionRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	operateLogRepo  *repository.OperateLogRepository
	docService      *DocService
	teamAccess      *TeamAccessService
	sensitiveFields *FunctionSensitiveFieldService
}

type appRuntimeClient interface {
	CreateApp(ctx context.Context, hostID int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error)
	UpdateApp(ctx context.Context, hostID int64, req *dto.UpdateAppRuntimeReq) (*dto.UpdateAppResp, error)
	RequestApp(ctx context.Context, hostID int64, req *dto.RequestAppReq) (*dto.RequestAppResp, error)
	DeleteApp(ctx context.Context, hostID int64, req *dto.DeleteAppRuntimeReq) (*dto.DeleteAppResp, error)
}

var _ appRuntimeClient = (*appcall.Client)(nil)

type AppServiceDependencies struct {
	RuntimeClient   appRuntimeClient
	AppRepository   *repository.AppRepository
	FunctionRepo    *repository.FunctionRepository
	ServiceTreeRepo *repository.ServiceTreeRepository
	OperateLogRepo  *repository.OperateLogRepository
	DocService      *DocService
	TeamAccess      *TeamAccessService
	SensitiveFields *FunctionSensitiveFieldService
}

// NewAppService 创建一个装配完成即可使用的 AppService。
func NewAppService(deps AppServiceDependencies) *AppService {
	return &AppService{
		appCall:         deps.RuntimeClient,
		appRepo:         deps.AppRepository,
		functionRepo:    deps.FunctionRepo,
		serviceTreeRepo: deps.ServiceTreeRepo,
		operateLogRepo:  deps.OperateLogRepo,
		docService:      deps.DocService,
		teamAccess:      deps.TeamAccess,
		sensitiveFields: deps.SensitiveFields,
	}
}
