package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/appcall"
	"github.com/kageos/kageos/pkg/contextx"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

type AppService struct {
	appCall         appRuntimeClient
	appRepo         *repository.AppRepository
	functionRepo    *repository.FunctionRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	operateLogRepo  *repository.OperateLogRepository
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

// NewAppService 创建 AppService（依赖注入）
func NewAppService(appCall appRuntimeClient, appRepo *repository.AppRepository, functionRepo *repository.FunctionRepository, serviceTreeRepo *repository.ServiceTreeRepository, operateLogRepo *repository.OperateLogRepository) *AppService {
	return &AppService{
		appCall:         appCall,
		appRepo:         appRepo,
		functionRepo:    functionRepo,
		serviceTreeRepo: serviceTreeRepo,
		operateLogRepo:  operateLogRepo,
	}
}

func (a *AppService) SetTeamAccessService(teamAccess *TeamAccessService) {
	a.teamAccess = teamAccess
}

func (a *AppService) SetFunctionSensitiveFieldService(sensitiveFields *FunctionSensitiveFieldService) {
	a.sensitiveFields = sensitiveFields
}

// CreateApp 创建应用
func (a *AppService) CreateApp(ctx context.Context, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	return a.createAppFlow(ctx, req)
}

// UpdateApp 更新应用（更新应用代码并重新编译部署）
func (a *AppService) UpdateApp(ctx context.Context, req *dto.UpdateAppReq) (*dto.UpdateAppResp, error) {
	start := time.Now()
	user, appCode, app, err := a.resolveUpdateTargetApp(req)
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] resolve target failed: resource_path=%s, err=%v, elapsed=%s",
			req.ResourcePath, err, time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	resolveElapsed := time.Since(start)

	// 调用 app-runtime 更新应用，使用应用所属的 HostID
	runtimeStart := time.Now()
	resp, err := a.appCall.UpdateApp(ctx, app.HostID, &dto.UpdateAppRuntimeReq{
		User:              user,
		App:               appCode,
		SourceFiles:       req.SourceFiles,
		Requirement:       req.Requirement,
		ChangeDescription: req.ChangeDescription,
		WriteOnly:         req.WriteOnly,
		ForceDiff:         req.ForceDiff,
	})
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] runtime update failed: user=%s, app=%s, hostID=%d, err=%v, resolveElapsed=%s, runtimeElapsed=%s, totalElapsed=%s",
			user, appCode, app.HostID, err,
			resolveElapsed.Truncate(time.Millisecond),
			time.Since(runtimeStart).Truncate(time.Millisecond),
			time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	runtimeElapsed := time.Since(runtimeStart)

	finalizeStart := time.Now()
	warnings, err := a.finalizeReleasedAppMetadata(ctx, "AppService:UpdateApp", app, user, appCode, resp.NewVersion, resp.Diff)
	if err != nil {
		logger.Errorf(ctx, "[AppService:UpdateApp] finalize metadata failed: user=%s, app=%s, newVersion=%s, err=%v, finalizeElapsed=%s, totalElapsed=%s",
			user, appCode, resp.NewVersion, err,
			time.Since(finalizeStart).Truncate(time.Millisecond),
			time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	resp.Warnings = append(resp.Warnings, warnings...)
	logger.Infof(ctx, "[AppService:UpdateApp] completed: user=%s, app=%s, oldVersion=%s, newVersion=%s, trace_id=%s, resolveElapsed=%s, runtimeElapsed=%s, finalizeElapsed=%s, totalElapsed=%s",
		user, appCode, resp.OldVersion, resp.NewVersion, updateAppTraceID(resp),
		resolveElapsed.Truncate(time.Millisecond),
		runtimeElapsed.Truncate(time.Millisecond),
		time.Since(finalizeStart).Truncate(time.Millisecond),
		time.Since(start).Truncate(time.Millisecond))

	return resp, nil
}

func updateAppTraceID(resp *dto.UpdateAppResp) string {
	if resp == nil || resp.BuildTrace == nil {
		return ""
	}
	return resp.BuildTrace.TraceID
}

func (a *AppService) resolveUpdateTargetApp(req *dto.UpdateAppReq) (string, string, *model.App, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return "", "", nil, err
	}

	app, err := a.appRepo.GetAppByUserName(user, appCode)
	return user, appCode, app, err
}

func (a *AppService) persistReleasedAppVersion(user, appCode, newVersion string) error {
	if newVersion == "" {
		return nil
	}
	return a.appRepo.UpdateAppVersion(user, appCode, newVersion)
}

func (a *AppService) syncUpdatedAppMetadata(
	ctx context.Context,
	appID int64,
	diff *dto.DiffData,
) string {
	if diff == nil {
		return ""
	}

	if err := a.processAPIDiff(ctx, appID, diff); err != nil {
		return fmt.Sprintf("应用已发布，但函数元数据同步失败: %v", err)
	}

	return ""
}

func (a *AppService) finalizeReleasedAppMetadata(
	ctx context.Context,
	logPrefix string,
	app *model.App,
	user, appCode, newVersion string,
	diff *dto.DiffData,
) ([]string, error) {
	if app == nil {
		return nil, fmt.Errorf("应用不存在")
	}

	if err := a.persistReleasedAppVersion(user, appCode, newVersion); err != nil {
		return nil, err
	}

	warning := a.syncUpdatedAppMetadata(ctx, app.ID, diff)
	if warning == "" {
		return nil, nil
	}

	logger.Warnf(ctx, "[%s] %s user=%s app=%s newVersion=%s",
		logPrefix, warning, user, appCode, newVersion)
	return []string{warning}, nil
}

// RequestApp 请求应用
func (a *AppService) RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	start := time.Now()
	app, err := a.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		logger.Errorf(ctx, "[AppService:RequestApp] GetAppByUserName failed: user=%s, app=%s, traceId=%s, err=%v, elapsed=%s",
			req.User, req.App, req.TraceId, err, time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	dbElapsed := time.Since(start)
	req.Version = app.Version
	a.applyRequestSourceContext(ctx, req)
	logger.Debugf(ctx, "[AppService:RequestApp] start: traceId=%s, %s/%s/%s, method=%s, router=%s, natsId=%d, dbElapsed=%s",
		req.TraceId, req.User, req.App, req.Version, req.Method, req.Router, app.NatsID, dbElapsed.Truncate(time.Millisecond))

	if err := a.requireFunctionConnectors(ctx, req); err != nil {
		logger.Warnf(ctx, "[AppService:RequestApp] connector dependency not ready: traceId=%s, user=%s, app=%s, router=%s, err=%v",
			req.TraceId, req.User, req.App, req.Router, err)
		return nil, err
	}

	resp, err := a.appCall.RequestApp(ctx, app.NatsID, req)
	totalElapsed := time.Since(start)
	if err != nil {
		logger.Errorf(ctx, "[AppService:RequestApp] appCall failed: traceId=%s, %s/%s/%s, err=%v, totalElapsed=%s",
			req.TraceId, req.User, req.App, req.Version, err, totalElapsed.Truncate(time.Millisecond))
		return nil, err
	}
	logger.Debugf(ctx, "[AppService:RequestApp] done: traceId=%s, %s/%s/%s, hasError=%v, totalElapsed=%s",
		req.TraceId, req.User, req.App, req.Version, resp.Error != "", totalElapsed.Truncate(time.Millisecond))
	resp.Version = req.Version
	return resp, nil
}

func (a *AppService) applyRequestSourceContext(ctx context.Context, req *dto.RequestAppReq) {
	if a == nil || a.serviceTreeRepo == nil || req == nil {
		return
	}
	if req.SourcePath == "" {
		req.SourcePath = requestAppFullCodePath(req)
	}
	if req.SourcePath == "" {
		return
	}
	if req.SourceTitle != "" && req.SourceParentTitle != "" && req.SourceTemplateType != "" {
		return
	}

	sourcePath := access.NormalizeResourcePath(req.SourcePath)
	parentPath := strings.TrimSpace(req.SourceParentPath)
	if parentPath == "" {
		parentPath = parentFullCodePath(sourcePath)
	}
	paths := []string{sourcePath}
	if parentPath != "" {
		paths = append(paths, parentPath)
	}
	nodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(paths)
	if err != nil {
		logger.Warnf(ctx, "[AppService:RequestApp] resolve source display failed: source_path=%s err=%v", sourcePath, err)
		return
	}
	if source := nodes[sourcePath]; source != nil {
		if req.SourceTitle == "" {
			req.SourceTitle = source.Name
		}
		if req.SourceTemplateType == "" {
			req.SourceTemplateType = source.TemplateType
		}
	}
	if parentPath != "" {
		req.SourceParentPath = parentPath
		if parent := nodes[parentPath]; parent != nil {
			if req.SourceParentTitle == "" {
				req.SourceParentTitle = parent.Name
			}
		}
	}
}

func requestAppFullCodePath(req *dto.RequestAppReq) string {
	if req == nil {
		return ""
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" || strings.HasPrefix(router, "_") {
		return ""
	}
	return access.NormalizeResourcePath(fmt.Sprintf("/%s/%s/%s", strings.TrimSpace(req.User), strings.TrimSpace(req.App), router))
}

func parentFullCodePath(fullCodePath string) string {
	fullCodePath = strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if fullCodePath == "" {
		return ""
	}
	parts := strings.Split(fullCodePath, "/")
	if len(parts) <= 2 {
		return ""
	}
	return "/" + strings.Join(parts[:len(parts)-1], "/")
}

func (a *AppService) requireFunctionConnectors(ctx context.Context, req *dto.RequestAppReq) error {
	if a == nil || a.functionRepo == nil {
		return nil
	}
	fullCodePath := requestFunctionFullCodePath(req)
	if fullCodePath == "" {
		return nil
	}
	function, err := a.functionRepo.GetFunctionByFullCodePath(fullCodePath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("检查函数连接器依赖失败: %w", err)
	}
	connectors := splitConnectorCodes(function.Connectors)
	if len(connectors) == 0 {
		return nil
	}
	endpoints := splitConnectorEndpoints(function.ConnectorEndpoints)
	statuses := functionConnectorStatuses(ctx, fullCodePath, connectors, endpoints)
	missing := missingConnectorProviders(statuses)
	if len(missing) > 0 {
		if err := connectorDependencyError(statuses); err != nil {
			return err
		}
		return fmt.Errorf("函数依赖连接器 %s，请先完成连接或补充授权后再执行", strings.Join(missing, "、"))
	}
	return nil
}

// IncrementFunctionRunCount 将指定 full_code_path 的 function 运行次数 +1（成功执行 Form/Table/Chart 后调用，用于 search 按热度排序）
func (a *AppService) IncrementFunctionRunCount(ctx context.Context, fullCodePath string) {
	if fullCodePath == "" {
		return
	}
	if err := a.serviceTreeRepo.IncrementRunCountByFullCodePath(ctx, fullCodePath); err != nil {
		logger.Warnf(ctx, "[AppService] IncrementFunctionRunCount failed: full_code_path=%s, err=%v", fullCodePath, err)
	}
}

// RecordTableActionLog 记录 Table 操作日志（OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows）。
func (a *AppService) RecordTableActionLog(ctx context.Context, req *dto.RecordTableActionLogReq) error {
	if req == nil {
		return nil
	}
	if req.TenantUser == "" || req.App == "" {
		return fmt.Errorf("记录 Table 操作日志失败: tenant_user 和 app 不能为空")
	}

	// 获取应用信息（用于获取版本号）
	app, err := a.appRepo.GetAppByUserName(req.TenantUser, req.App)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = app.Version
	}

	resourceID := fmt.Sprintf("%s/%s/%s", req.TenantUser, req.App, strings.TrimPrefix(req.Router, "/"))

	// 根据操作类型处理不同的记录逻辑
	switch req.Action {
	case "OnTableAddRow":
		log := a.buildTableActionOperateLog(ctx, req, resourceID, req.RowID, req.Body, nil, version)
		go func(log *model.OperateLog) {
			if err := a.operateLogRepo.CreateOperateLog(context.Background(), log); err != nil {
				logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 新增操作日志失败: %v", err)
			}
		}(log)

	case "OnTableUpdateRow":
		// 更新操作：记录 updates 和 old_values
		log := a.buildTableActionOperateLog(ctx, req, resourceID, req.RowID, req.Updates, req.OldValues, version)
		go func(log *model.OperateLog) {
			if err := a.operateLogRepo.CreateOperateLog(context.Background(), log); err != nil {
				logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 更新操作日志失败: %v", err)
			}
		}(log)

	case "OnTableDeleteRows":
		// 删除操作：为每个删除的记录创建一条日志
		for _, rowID := range req.RowIDs {
			log := a.buildTableActionOperateLog(ctx, req, resourceID, rowID, nil, nil, version)
			go func(log *model.OperateLog) {
				if err := a.operateLogRepo.CreateOperateLog(context.Background(), log); err != nil {
					logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 删除操作日志失败: %v", err)
				}
			}(log)
		}
	}

	return nil
}

func (a *AppService) buildTableActionOperateLog(ctx context.Context, req *dto.RecordTableActionLogReq, resourceID string, rowID int64, updates, oldValues []byte, version string) *model.OperateLog {
	fullCodePath := buildTableActionLogFullCodePath(req, resourceID)
	tableFields := a.sensitiveFieldSet(fullCodePath, sensitiveFieldSectionRequest, sensitiveFieldSectionFields)
	responseFields := a.sensitiveFieldSet(fullCodePath, sensitiveFieldSectionResponse)
	auditMeta := buildOperateLogAuditMetadata(ctx, req.Source)
	details := dto.TableActionLogDetails{
		RowID:      rowID,
		Version:    version,
		SourceType: auditMeta.SourceType,
		SourceRef:  auditMeta.SourceRef,
	}
	if len(req.RowIDs) > 0 {
		details.RowIDs = req.RowIDs
	}
	if req.DurationMillis > 0 {
		details.DurationMillis = req.DurationMillis
	}
	if len(req.ResponseBody) > 0 {
		details.ResponseBody = rawMessageObjectWithFields(req.ResponseBody, responseFields)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "success"
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		summary = buildTableActionLogSummary(req.RequestUser, req.Action, fullCodePath, rowID, status)
	}
	log := &model.OperateLog{
		TenantUser:    req.TenantUser,
		CompanyCode:   contextx.GetRequestCompanyCode(ctx),
		App:           req.App,
		ActorUser:     req.RequestUser,
		Action:        req.Action,
		ResourceType:  "table",
		ResourcePath:  fullCodePath,
		ResourceName:  strings.TrimPrefix(req.Router, "/"),
		TargetID:      fmt.Sprintf("%d", rowID),
		Summary:       summary,
		DetailsJSON:   mustMarshalRaw(details),
		OldValuesJSON: sanitizeOperateLogRawMessageWithFields(oldValues, tableFields),
		NewValuesJSON: sanitizeOperateLogRawMessageWithFields(updates, tableFields),
		Status:        status,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		TraceID:       req.TraceID,
	}
	applyOperateLogAuditMetadata(log, auditMeta)
	return log
}

func (a *AppService) sensitiveFieldSet(fullCodePath string, sections ...string) map[string]struct{} {
	if a == nil || a.sensitiveFields == nil {
		return nil
	}
	return a.sensitiveFields.SensitiveFieldSet(fullCodePath, sections...)
}

func buildTableActionLogSummary(requestUser, action, fullCodePath string, rowID int64, status string) string {
	if status == "failed" {
		switch action {
		case "OnTableAddRow":
			return fmt.Sprintf("%s failed to create row on %s", requestUser, fullCodePath)
		case "OnTableUpdateRow":
			return fmt.Sprintf("%s failed to update row #%d on %s", requestUser, rowID, fullCodePath)
		case "OnTableDeleteRows":
			return fmt.Sprintf("%s failed to delete row #%d on %s", requestUser, rowID, fullCodePath)
		default:
			return fmt.Sprintf("%s failed to execute %s on %s", requestUser, action, fullCodePath)
		}
	}
	switch action {
	case "OnTableAddRow":
		return fmt.Sprintf("%s created row on %s", requestUser, fullCodePath)
	case "OnTableUpdateRow":
		return fmt.Sprintf("%s updated row #%d on %s", requestUser, rowID, fullCodePath)
	case "OnTableDeleteRows":
		return fmt.Sprintf("%s deleted row #%d on %s", requestUser, rowID, fullCodePath)
	default:
		return fmt.Sprintf("%s executed %s on %s", requestUser, action, fullCodePath)
	}
}

func buildTableActionLogFullCodePath(req *dto.RecordTableActionLogReq, resourceID string) string {
	parts := strings.Split(resourceID, "/")
	fullCodePath := fmt.Sprintf("/%s/%s", req.TenantUser, req.App)
	if len(parts) > 2 {
		fullCodePath = fmt.Sprintf("/%s/%s/%s", req.TenantUser, req.App, strings.Join(parts[2:], "/"))
	}
	return fullCodePath
}

// processAPIDiff 处理API差异，包括新增、更新、删除
func (a *AppService) processAPIDiff(ctx context.Context, appID int64, diffData *dto.DiffData) error {
	state, err := a.loadAppMetadataSyncState(ctx, appID)
	if err != nil {
		return err
	}

	// ⭐ 第一步：目录对账——SDK 返回全量 package 列表，先与数据库比对，缺失的统一创建
	if len(diffData.Packages) > 0 {
		if err := a.reconcilePackages(ctx, state, diffData.Packages); err != nil {
			return fmt.Errorf("目录对账失败: %w", err)
		}
	}

	if err := a.syncAddedAPIs(ctx, state, diffData.Add); err != nil {
		return err
	}
	if err := a.syncUpdatedAPIs(ctx, state, diffData.Update); err != nil {
		return err
	}
	scheduledAPIs := append([]*dto.ApiInfo{}, diffData.Add...)
	scheduledAPIs = append(scheduledAPIs, diffData.Update...)
	if err := a.reconcileFormSchedules(ctx, state, scheduledAPIs); err != nil {
		return fmt.Errorf("同步默认定时任务失败: %w", err)
	}
	if err := a.reconcilePackageAgentTasks(ctx, state, diffData.Packages); err != nil {
		return fmt.Errorf("同步默认 Agent 任务失败: %w", err)
	}

	// 处理删除的API
	if len(diffData.Delete) > 0 {
		err := a.deleteFunctionsForAPIs(ctx, state.app, diffData.Delete)
		if err != nil {
			return fmt.Errorf("删除function和service_tree记录失败: %w", err)
		}
	}

	return nil
}

type appMetadataSyncState struct {
	app               *model.App
	currentVersionNum int
	requestUser       string
}

func (a *AppService) loadAppMetadataSyncState(ctx context.Context, appID int64) (*appMetadataSyncState, error) {
	app, err := a.appRepo.GetAppByID(appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	return &appMetadataSyncState{
		app:               app,
		currentVersionNum: app.GetVersionNumber(),
		requestUser:       contextx.GetRequestUser(ctx),
	}, nil
}

func (a *AppService) syncAddedAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo) error {
	if len(apis) == 0 {
		return nil
	}

	functions, err := a.convertApiInfoToFunctions(state.app.ID, apis, state.requestUser)
	if err != nil {
		return fmt.Errorf("转换新增API失败: %w", err)
	}

	if err := a.functionRepo.CreateFunctions(functions); err != nil {
		return fmt.Errorf("创建function记录失败: %w", err)
	}

	if err := a.createServiceTreesForAPIs(state, apis, functions); err != nil {
		return fmt.Errorf("创建service_tree记录失败: %w", err)
	}
	if err := a.syncSensitiveFieldsForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("同步敏感字段失败: %w", err)
	}

	return nil
}

func (a *AppService) syncUpdatedAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo) error {
	if len(apis) == 0 {
		return nil
	}

	functions, err := a.convertApiInfoToFunctions(state.app.ID, apis, state.requestUser)
	if err != nil {
		return fmt.Errorf("转换更新API失败: %w", err)
	}

	if err := a.updateFunctionsForAPIs(state.app.ID, apis, functions); err != nil {
		return fmt.Errorf("更新function记录失败: %w", err)
	}

	if err := a.updateServiceTreesForAPIs(state, apis, functions); err != nil {
		return fmt.Errorf("更新service_tree记录失败: %w", err)
	}
	if err := a.syncSensitiveFieldsForAPIs(ctx, state, apis, functions); err != nil {
		return fmt.Errorf("同步敏感字段失败: %w", err)
	}

	return nil
}

func (a *AppService) syncSensitiveFieldsForAPIs(ctx context.Context, state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	if a.sensitiveFields == nil || len(apis) == 0 {
		return nil
	}
	for i, api := range apis {
		if api == nil || api.Schema == nil {
			continue
		}
		var functionID int64
		if i < len(functions) && functions[i] != nil {
			functionID = functions[i].ID
		}
		if err := a.sensitiveFields.SyncFunctionSchema(ctx, state.app.User, state.app.Code, api.BuildFullCodePath(), functionID, api.Schema); err != nil {
			return fmt.Errorf("同步 %s 敏感字段失败: %w", api.BuildFullCodePath(), err)
		}
	}
	return nil
}

// convertApiInfoToFunctions 将ApiInfo转换为Function模型
func (a *AppService) convertApiInfoToFunctions(appID int64, apis []*dto.ApiInfo, username string) ([]*model.Function, error) {
	functions := make([]*model.Function, len(apis))

	for i, api := range apis {
		schemaJSON, err := functionschema.Marshal(api.Schema)
		if err != nil {
			return nil, fmt.Errorf("序列化function schema失败: %w", err)
		}
		if api.TemplateType != "" && api.Schema != nil && api.TemplateType != api.Schema.Type {
			return nil, fmt.Errorf("template_type 与 schema.type 不一致: template_type=%s schema.type=%s", api.TemplateType, api.Schema.Type)
		}

		endpoints := normalizeConnectorEndpoints(api.ConnectorEndpoints)
		connectors := normalizeConnectorCodes(append(append([]string{}, api.Connectors...), connectorCodesFromEndpoints(endpoints)...))
		function := &model.Function{
			AppID:              appID,
			Method:             api.Method,
			Router:             api.BuildFullCodePath(),
			Schema:             schemaJSON,
			HasConfig:          false, // 预留字段，默认为false
			Connectors:         joinConnectorCodes(connectors),
			ConnectorEndpoints: joinConnectorEndpoints(endpoints),
			TemplateType:       api.TemplateType,
		}
		// 设置创建者用户名（通过嵌入的 Base 结构体）
		function.CreatedBy = username
		if api.CreateTables != nil {
			function.CreateTables = strings.Join(api.CreateTables, ",")
		}

		functions[i] = function
	}

	return functions, nil
}

// createServiceTreesForAPIs 为新增的API创建ServiceTree记录
func (a *AppService) createServiceTreesForAPIs(state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	parentNodes, err := a.loadParentPackageNodes(apis)
	if err != nil {
		return err
	}

	for i, api := range apis {
		var parentAdmins string
		parentPath := api.GetParentFullCodePath()

		if parentPath != "" {
			parent, exists := parentNodes[parentPath]
			if !exists {
				return fmt.Errorf("父级package节点不存在: %s（目录对账可能未覆盖此路径）", parentPath)
			}
			parentAdmins = parent.Admins
		}

		treeID, err := a.createFunctionNode(state, api, functions[i].ID, parentAdmins)
		if err != nil {
			return fmt.Errorf("创建function节点失败: %w", err)
		}
		api.TreeID = treeID
	}
	return nil
}

// reconcilePackages 目录对账：拿 SDK 返回的全量 package 列表与数据库比对，缺失的统一创建
// SDK 每次 update 都会返回当前应用注册的全量 package（已按深度从浅到深排序），
// 所以按顺序遍历即可保证父节点先于子节点被创建
func (a *AppService) reconcilePackages(ctx context.Context, state *appMetadataSyncState, packages []*dto.PackageInfo) error {
	// 1. 提取所有 fullPath，批量查库看哪些已存在
	allPaths := make([]string, 0, len(packages)+1)
	for _, pkg := range packages {
		allPaths = append(allPaths, pkg.FullPath)
	}

	rootPath := fmt.Sprintf("/%s/%s", state.app.User, state.app.Code)
	allPaths = append(allPaths, rootPath)

	existingNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(allPaths)
	if err != nil {
		return fmt.Errorf("批量查询package节点失败: %w", err)
	}

	// 2. 找出缺失的 package
	var missing []*dto.PackageInfo
	for _, pkg := range packages {
		if _, exists := existingNodes[pkg.FullPath]; !exists {
			missing = append(missing, pkg)
		}
	}

	if len(missing) == 0 {
		logger.Infof(ctx, "[reconcilePackages] 目录对账完成: %d 个 package 全部存在，无需创建", len(packages))
		return nil
	}

	logger.Infof(ctx, "[reconcilePackages] 目录对账: 共 %d 个 package，其中 %d 个缺失，开始创建", len(packages), len(missing))

	// 3. 按顺序创建缺失的 package（SDK 已按深度排序，父在前子在后）
	for _, pkg := range missing {
		parts := strings.Split(strings.Trim(pkg.FullPath, "/"), "/")
		// parts 至少 3 段：user/app/pkg，parentPath = /user/app（即根节点）
		if len(parts) < 3 {
			logger.Warnf(ctx, "[reconcilePackages] 跳过非法路径: %s (深度不足)", pkg.FullPath)
			continue
		}
		parentPath := "/" + strings.Join(parts[:len(parts)-1], "/")
		_, ok := existingNodes[parentPath]
		if !ok {
			return fmt.Errorf("[reconcilePackages] 父节点不存在: %s (package=%s)，请检查根节点或上级目录是否已创建", parentPath, pkg.FullPath)
		}

		node := &model.ServiceTree{
			AppID:            state.app.ID,
			Type:             model.ServiceTreeTypePackage,
			Code:             pkg.Code,
			Name:             pkg.Name,
			Description:      pkg.Desc,
			FullCodePath:     pkg.FullPath,
			AddVersionNum:    state.currentVersionNum,
			UpdateVersionNum: 0,
		}
		if state.requestUser != "" {
			node.CreatedBy = state.requestUser
			node.Admins = state.requestUser
		}

		if err := a.serviceTreeRepo.Create(node); err != nil {
			return fmt.Errorf("创建 package 节点失败 (%s): %w", pkg.FullPath, err)
		}

		existingNodes[pkg.FullPath] = node
		logger.Infof(ctx, "[reconcilePackages] 创建 package: %s (code=%s, name=%s, parentPath=%s)", pkg.FullPath, pkg.Code, pkg.Name, parentPath)
	}

	logger.Infof(ctx, "[reconcilePackages] 目录对账完成: 成功创建 %d 个缺失的 package 节点", len(missing))
	return nil
}

func (a *AppService) loadParentPackageNodes(apis []*dto.ApiInfo) (map[string]*model.ServiceTree, error) {
	parentPaths := make(map[string]bool)
	for _, api := range apis {
		parentPath := api.GetParentFullCodePath()
		if parentPath != "" {
			parentPaths[parentPath] = true
		}
	}

	parentPathList := make([]string, 0, len(parentPaths))
	for path := range parentPaths {
		parentPathList = append(parentPathList, path)
	}

	parentNodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(parentPathList)
	if err != nil {
		return nil, fmt.Errorf("批量查询父级package节点失败: %w", err)
	}

	for path, node := range parentNodes {
		if !node.IsPackage() {
			return nil, fmt.Errorf("路径 %s 已存在，但类型不是package，当前类型: %s", path, node.Type)
		}
	}

	return parentNodes, nil
}

// createFunctionNode 创建function节点，返回创建的TreeID
func (a *AppService) createFunctionNode(
	state *appMetadataSyncState,
	api *dto.ApiInfo,
	functionID int64,
	parentAdmins string,
) (int64, error) {
	// 检查是否已存在（full_name_path全局唯一）
	existingNode, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
	if err == nil {
		a.applyFunctionNodeMetadata(existingNode, api, functionID)

		// 如果节点是新增的（AddVersionNum为0），设置添加版本号
		if existingNode.AddVersionNum == 0 {
			existingNode.AddVersionNum = state.currentVersionNum
		} else {
			// 如果节点已存在，更新更新版本号
			existingNode.UpdateVersionNum = state.currentVersionNum
		}

		// 更新节点信息
		err = a.serviceTreeRepo.UpdateServiceTree(existingNode)
		if err != nil {
			return 0, err
		}
		// 返回已存在的节点ID
		return existingNode.ID, nil
	}
	// 如果是记录不存在的错误，这是正常的，继续创建
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, fmt.Errorf("查询路径失败: %w", err)
	}
	// err是gorm.ErrRecordNotFound，说明路径不存在，可以继续创建

	// 获取创建者用户名
	requestUser := state.requestUser

	// 确定 Admins 字段：优先使用父节点的 Admins，如果没有父节点或父节点没有 Admins，则使用当前用户
	admins := parentAdmins
	if admins == "" && requestUser != "" {
		admins = requestUser
	}

	// 创建新的function节点
	serviceTree := &model.ServiceTree{
		AppID:            state.app.ID,
		Type:             model.ServiceTreeTypeFunction,
		FullCodePath:     api.FullCodePath,        // 直接使用api.FullCodePath，不需要重新计算
		AddVersionNum:    state.currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,                       // 新增节点，更新版本号为0
		Admins:           admins,                  // 设置管理员列表（从父节点继承，或使用当前用户）
	}
	a.applyFunctionNodeMetadata(serviceTree, api, functionID)

	// 设置创建者
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	// 创建ServiceTree节点（GORM Create会自动填充ID）
	err = a.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, "")
	if err != nil {
		return 0, err
	}

	// 返回创建的节点ID
	return serviceTree.ID, nil
}

func (a *AppService) applyFunctionNodeMetadata(tree *model.ServiceTree, api *dto.ApiInfo, functionID int64) {
	tree.Code = api.Code
	tree.Name = api.Name
	tree.Description = api.Desc
	tree.TemplateType = api.TemplateType
	tree.RefID = functionID
	tree.Tags = strings.Join(api.Tags, ",")
	endpoints := normalizeConnectorEndpoints(api.ConnectorEndpoints)
	tree.Connectors = joinConnectorCodes(append(append([]string{}, api.Connectors...), connectorCodesFromEndpoints(endpoints)...))
	tree.ConnectorEndpoints = joinConnectorEndpoints(endpoints)
}

// updateFunctionsForAPIs 更新API对应的Function记录
func (a *AppService) updateFunctionsForAPIs(appID int64, apis []*dto.ApiInfo, functions []*model.Function) error {
	// 对于每个要更新的API，先查找现有的Function记录获取ID
	for i, api := range apis {
		router := api.BuildFullCodePath()
		existingFunction, err := a.functionRepo.GetFunctionByKey(appID, api.Method, router)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Function不存在，创建新的（这种情况不应该发生，但为了容错处理）
				newFunctions := []*model.Function{functions[i]}
				err = a.functionRepo.CreateFunctions(newFunctions)
				if err != nil {
					return fmt.Errorf("创建function记录失败: %w", err)
				}
				// 更新functions[i]的ID
				functions[i].ID = newFunctions[0].ID
				continue
			}
			return fmt.Errorf("查询function记录失败: %w", err)
		}
		// 保留现有的ID
		functions[i].ID = existingFunction.ID
	}

	// 批量更新Function记录
	return a.functionRepo.UpdateFunctions(functions)
}

// updateServiceTreesForAPIs 更新API对应的ServiceTree记录
func (a *AppService) updateServiceTreesForAPIs(state *appMetadataSyncState, apis []*dto.ApiInfo, functions []*model.Function) error {
	parentNodes, err := a.loadParentPackageNodes(apis)
	if err != nil {
		return err
	}

	// 更新function节点
	for i, api := range apis {
		// 根据FullCodePath查找现有的ServiceTree
		existingTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果不存在，创建新的节点（这种情况不应该发生，但为了容错处理）
				var parentAdmins string
				parentPath := api.GetParentFullCodePath()
				if parentPath != "" {
					parent, exists := parentNodes[parentPath]
					if exists {
						parentAdmins = parent.Admins
					}
				}
				treeID, err := a.createFunctionNode(state, api, functions[i].ID, parentAdmins)
				if err != nil {
					return fmt.Errorf("创建function节点失败: %w", err)
				}
				// 赋值TreeID，方便后续写快照时入库
				api.TreeID = treeID
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 更新节点信息并设置更新版本号
		a.applyFunctionNodeMetadata(existingTree, api, functions[i].ID)
		// 更新版本号：如果AddVersionNum为0，说明是新增的，设置为当前版本；否则更新UpdateVersionNum
		if existingTree.AddVersionNum == 0 {
			existingTree.AddVersionNum = state.currentVersionNum
		} else {
			existingTree.UpdateVersionNum = state.currentVersionNum
		}

		// 保存更新后的节点
		if err := a.serviceTreeRepo.UpdateServiceTree(existingTree); err != nil {
			return fmt.Errorf("更新service_tree节点失败: %w", err)
		}
		// 赋值TreeID，方便后续写快照时入库
		api.TreeID = existingTree.ID
	}
	return nil
}

// deleteFunctionsForAPIs 删除API对应的Function和ServiceTree记录
func (a *AppService) deleteFunctionsForAPIs(ctx context.Context, app *model.App, apis []*dto.ApiInfo) error {
	// 收集需要删除的router和method
	routers := make([]string, 0, len(apis))
	methods := make([]string, 0, len(apis))

	for _, api := range apis {
		// 根据FullCodePath查找ServiceTree
		serviceTree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(api.FullCodePath)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// ServiceTree不存在，跳过
				continue
			}
			return fmt.Errorf("查询service_tree失败: %w", err)
		}

		// 删除ServiceTree（会级联删除子节点）
		err = a.serviceTreeRepo.DeleteServiceTree(serviceTree.ID)
		if err != nil {
			return fmt.Errorf("删除service_tree失败: %w", err)
		}

		// 收集Function的router和method用于删除
		router := api.BuildFullCodePath()
		routers = append(routers, router)
		methods = append(methods, api.Method)
		if a.sensitiveFields != nil {
			if err := a.sensitiveFields.DeleteFunction(ctx, app.User, app.Code, router); err != nil {
				return fmt.Errorf("删除function敏感字段失败: %w", err)
			}
		}
	}

	// 批量删除Function记录
	if len(routers) > 0 {
		err := a.functionRepo.DeleteFunctions(app.ID, routers, methods)
		if err != nil {
			return fmt.Errorf("删除function记录失败: %w", err)
		}
	}

	return nil
}

// DeleteApp 删除应用
func (a *AppService) DeleteApp(ctx context.Context, req *dto.DeleteAppReq) (*dto.DeleteAppResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	// 根据应用信息获取 NATS 连接
	app, err := a.appRepo.GetAppByUserName(user, appCode)
	if err != nil {
		return nil, err
	}

	// 调用 app-runtime 删除应用
	resp, err := a.appCall.DeleteApp(ctx, app.HostID, &dto.DeleteAppRuntimeReq{
		User: user,
		App:  appCode,
	})
	if err != nil {
		return nil, err
	}

	// 删除数据库记录
	err = a.appRepo.DeleteAppAndVersions(user, appCode)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetApps 获取应用列表
func (a *AppService) GetApps(ctx context.Context, req *dto.GetAppsReq) (*dto.GetAppsResp, error) {
	// 设置分页默认值
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10 // 默认每页10条
	}

	// 从数据库获取应用列表（支持搜索和过滤）
	apps, _, err := a.appRepo.GetAppsWithPage(req.User, page, pageSize, req.Search, req.IncludeAll, req.Type)
	if err != nil {
		return nil, fmt.Errorf("获取应用列表失败: %w", err)
	}
	apps, err = a.mergeAccessibleApps(ctx, apps, req)
	if err != nil {
		return nil, fmt.Errorf("获取已授权应用失败: %w", err)
	}

	// 转换为 AppInfo 列表
	appInfos := make([]*dto.AppInfo, 0, len(apps))
	for _, app := range apps {
		if !a.canReadApp(ctx, app, req.User) {
			continue
		}
		appInfos = append(appInfos, &dto.AppInfo{
			ID:                    app.ID,
			User:                  app.User,
			Code:                  app.Code,
			Name:                  app.Name,
			Status:                app.Status,
			Version:               app.Version,
			NatsID:                app.NatsID,
			HostID:                app.HostID,
			IsPublic:              app.IsPublic,
			HideUnauthorizedNodes: app.HideUnauthorizedNodes,
			Admins:                app.Admins,
			Type:                  int(app.Type),
			CreatedAt:             time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt:             time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		})
	}

	return &dto.GetAppsResp{
		PageInfoResp: dto.PageInfoResp{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: len(appInfos),
			Items:      appInfos,
		},
	}, nil
}

func (a *AppService) mergeAccessibleApps(ctx context.Context, apps []*model.App, req *dto.GetAppsReq) ([]*model.App, error) {
	if a.teamAccess == nil || req == nil || req.User == "" || req.Type != nil {
		return apps, nil
	}
	grantedApps, err := a.teamAccess.ListAccessibleApps(ctx, req.User)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(apps)+len(grantedApps))
	merged := make([]*model.App, 0, len(apps)+len(grantedApps))
	matchesSearch := func(app *model.App) bool {
		if app == nil {
			return false
		}
		search := strings.ToLower(strings.TrimSpace(req.Search))
		if search == "" {
			return true
		}
		return strings.Contains(strings.ToLower(app.Name), search) || strings.Contains(strings.ToLower(app.Code), search)
	}
	for _, app := range apps {
		if app == nil {
			continue
		}
		key := app.User + "/" + app.Code
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, app)
	}
	for _, app := range grantedApps {
		if !matchesSearch(app) {
			continue
		}
		key := app.User + "/" + app.Code
		if seen[key] {
			continue
		}
		seen[key] = true
		merged = append(merged, app)
	}
	return merged, nil
}

func (a *AppService) canReadApp(ctx context.Context, app *model.App, currentUser string) bool {
	if app == nil {
		return false
	}
	if a.teamAccess == nil {
		return true
	}
	resourcePath := app.GetPrefix()
	ok, err := a.teamAccess.Can(ctx, app.User, app.Code, currentUser, resourcePath, access.ActionRead)
	if err != nil {
		return false
	}
	if ok {
		return true
	}
	hasAnyAccess, err := a.teamAccess.HasAnyWorkspaceAccess(ctx, app.User, app.Code, currentUser)
	return err == nil && hasAnyAccess
}

// GetAppDetail 获取应用详情
func (a *AppService) GetAppDetail(ctx context.Context, req *dto.GetAppDetailReq) (*dto.GetAppDetailResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	app, err := a.appRepo.GetAppByUserName(user, appCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("应用不存在: %s/%s", user, appCode)
		}
		return nil, fmt.Errorf("获取应用详情失败: %w", err)
	}

	return &dto.GetAppDetailResp{
		AppInfo: dto.AppInfo{
			ID:                    app.ID,
			User:                  app.User,
			Code:                  app.Code,
			Name:                  app.Name,
			Status:                app.Status,
			Version:               app.Version,
			NatsID:                app.NatsID,
			HostID:                app.HostID,
			IsPublic:              app.IsPublic,
			HideUnauthorizedNodes: app.HideUnauthorizedNodes,
			Admins:                app.Admins,
			Type:                  int(app.Type),
			CreatedAt:             time.Time(app.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt:             time.Time(app.UpdatedAt).Format("2006-01-02 15:04:05"),
		},
	}, nil
}

// GetAppByUserName 根据用户名和应用名获取应用信息
func (a *AppService) GetAppByUserName(ctx context.Context, user, app string) (*model.App, error) {
	return a.appRepo.GetAppByUserName(user, app)
}

// UpdateWorkspace 更新工作空间（只更新 MySQL 记录，不涉及容器更新）
func (a *AppService) UpdateWorkspace(ctx context.Context, req *dto.UpdateWorkspaceReq) (*dto.UpdateWorkspaceResp, error) {
	user, appCode, err := resolveUserAppFromRequiredResourcePath(req.ResourcePath)
	if err != nil {
		return nil, err
	}

	// 获取应用信息
	app, err := a.appRepo.GetAppByUserName(user, appCode)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	if req.Admins != nil {
		app.Admins = *req.Admins
	}
	if req.HideUnauthorizedNodes != nil {
		app.HideUnauthorizedNodes = *req.HideUnauthorizedNodes
	}
	if err := a.appRepo.UpdateApp(app); err != nil {
		return nil, fmt.Errorf("更新工作空间失败: %w", err)
	}

	logger.Infof(ctx, "[AppService] 更新工作空间成功: user=%s, app=%s, admins=%s, hide_unauthorized_nodes=%t",
		user, appCode, app.Admins, app.HideUnauthorizedNodes)

	return &dto.UpdateWorkspaceResp{
		User:                  user,
		App:                   appCode,
		Admins:                app.Admins,
		HideUnauthorizedNodes: app.HideUnauthorizedNodes,
	}, nil
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数信息
// fullCodePath 参数应该是完整的路径（如 /luobei/operations/tools/pdftools/to_images）
// full-code-path 是全局唯一的，不需要 method 参数
func (a *AppService) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error) {
	function, err := a.functionRepo.GetFunctionByFullCodePath(fullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取函数信息失败: %w", err)
	}

	return function, nil
}
