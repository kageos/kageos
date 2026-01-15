package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	operatelogrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/operatelog/repository"
)

type OperateLogService struct {
	operateLogRepo *operatelogrepo.OperateLogRepository
	appRepo        *repository.AppRepository
}

func NewOperateLogService(options *enterprise.InitOptions) (*OperateLogService, error) {
	srv := &OperateLogService{}
	err := srv.Init(options)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (ops *OperateLogService) Init(options *enterprise.InitOptions) error {
	ops.operateLogRepo = operatelogrepo.NewOperateLogRepository(options.DB)
	ops.appRepo = repository.NewAppRepository(options.DB)
	return nil
}

func (ops *OperateLogService) CreateOperateLogger(req *dto.CreateOperateLoggerReq) (*dto.CreateOperateLoggerResp, error) {
	ctx := context.Background()

	// 根据 Resource 类型决定记录到哪个表
	switch req.Resource {
	case "app":
		// request_app 操作记录到 FormOperateLog 表
		if req.Action == "request_app" {
			return ops.createFormOperateLog(ctx, req)
		}
		// 其他 app 操作暂时不记录（如 create_app, update_app 等）
		return &dto.CreateOperateLoggerResp{}, nil
	case "table":
		// Table 操作记录到 TableOperateLog 表
		return ops.createTableOperateLog(ctx, req)
	case "form":
		// Form 提交操作记录到 FormOperateLog 表
		if req.Action == "form_submit" {
			return ops.createFormOperateLog(ctx, req)
		}
		// 其他 form 操作暂时不记录
		return &dto.CreateOperateLoggerResp{}, nil
	default:
		// 其他资源类型暂时不记录
		return &dto.CreateOperateLoggerResp{}, nil
	}
}

// createFormOperateLog 创建 Form 操作日志
func (ops *OperateLogService) createFormOperateLog(ctx context.Context, req *dto.CreateOperateLoggerReq) (*dto.CreateOperateLoggerResp, error) {
	// 解析 ResourceID（格式：user/app 或 user/app/router）
	parts := strings.Split(req.ResourceID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 ResourceID 格式: %s，应为 user/app 或 user/app/router", req.ResourceID)
	}
	tenantUser := parts[0]
	app := parts[1]

	// 获取 router（优先使用 ResourceID 中的，否则使用请求中的）
	var router string
	if len(parts) > 2 {
		// ResourceID 包含 router（格式：user/app/router）
		router = strings.Join(parts[2:], "/")
	} else {
		router = req.Router
	}

	// 获取 method 和 version
	method := req.Method
	version := req.Version

	// 如果 version 为空，查询 app 信息获取版本号
	if version == "" {
		appModel, err := ops.appRepo.GetAppByUserName(tenantUser, app)
		if err == nil && appModel != nil {
			version = appModel.Version
		}
	}

	// 构建 full_code_path（格式：/user/app/router）
	fullCodePath := fmt.Sprintf("/%s/%s", tenantUser, app)
	if router != "" {
		fullCodePath = fmt.Sprintf("/%s/%s/%s", tenantUser, app, strings.TrimPrefix(router, "/"))
	}

	// 使用请求中的 RequestBody 和 ResponseBody
	requestBody := req.RequestBody
	responseBody := req.ResponseBody

	// 创建 FormOperateLog 记录
	log := &model.FormOperateLog{
		TenantUser:      tenantUser,
		RequestUser:     req.User,
		Action:          req.Action,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		App:             app,
		FullCodePath:    fullCodePath,
		FunctionMethod:  method,
		RequestBody:     requestBody,
		ResponseBody:    responseBody,
		Code:            0, // 默认成功
		Msg:             "",
		Version:         version,
	}

	// 保存到数据库
	if err := ops.operateLogRepo.CreateFormOperateLog(log); err != nil {
		return nil, fmt.Errorf("创建 Form 操作日志失败: %w", err)
	}

	return &dto.CreateOperateLoggerResp{
		ID: log.ID,
	}, nil
}

// createTableOperateLog 创建 Table 操作日志
func (ops *OperateLogService) createTableOperateLog(ctx context.Context, req *dto.CreateOperateLoggerReq) (*dto.CreateOperateLoggerResp, error) {
	// 解析 ResourceID（格式：user/app/router）
	parts := strings.Split(req.ResourceID, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 ResourceID 格式: %s，应为 user/app 或 user/app/router", req.ResourceID)
	}
	tenantUser := parts[0]
	app := parts[1]
	
	// 构建 full_code_path
	fullCodePath := fmt.Sprintf("/%s/%s", tenantUser, app)
	if len(parts) > 2 {
		router := strings.Join(parts[2:], "/")
		fullCodePath = fmt.Sprintf("/%s/%s/%s", tenantUser, app, router)
	}

	// 获取 version（如果为空则查询）
	version := req.Version
	if version == "" {
		appModel, err := ops.appRepo.GetAppByUserName(tenantUser, app)
		if err == nil && appModel != nil {
			version = appModel.Version
		}
	}

	// 创建 TableOperateLog 记录
	log := &model.TableOperateLog{
		TenantUser:   tenantUser,
		RequestUser:  req.User,
		Action:       req.Action,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		App:          app,
		FullCodePath: fullCodePath,
		RowID:        req.RowID,
		Updates:      req.Updates,
		OldValues:    req.OldValues,
		TraceID:      req.TraceID,
		Version:      version,
	}

	// 保存到数据库
	if err := ops.operateLogRepo.CreateTableOperateLog(log); err != nil {
		return nil, fmt.Errorf("创建 Table 操作日志失败: %w", err)
	}

	return &dto.CreateOperateLoggerResp{
		ID: log.ID,
	}, nil
}

// GetTableOperateLogs 查询 Table 操作日志（企业版实现）
func (ops *OperateLogService) GetTableOperateLogs(ctx context.Context, req *dto.GetTableOperateLogsReq) (*dto.GetTableOperateLogsResp, error) {
	// 调用 repository 层查询
	logs, total, err := ops.operateLogRepo.GetTableOperateLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// 转换为 interface{} 列表
	logsInterface := make([]interface{}, len(logs))
	for i, log := range logs {
		logsInterface[i] = log
	}

	return &dto.GetTableOperateLogsResp{
		Logs:     logsInterface,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetFormOperateLogs 查询 Form 操作日志（企业版实现）
func (ops *OperateLogService) GetFormOperateLogs(ctx context.Context, req *dto.GetFormOperateLogsReq) (*dto.GetFormOperateLogsResp, error) {
	// 调用 repository 层查询
	logs, total, err := ops.operateLogRepo.GetFormOperateLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// 转换为 interface{} 列表
	logsInterface := make([]interface{}, len(logs))
	for i, log := range logs {
		logsInterface[i] = log
	}

	return &dto.GetFormOperateLogsResp{
		Logs:     logsInterface,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
