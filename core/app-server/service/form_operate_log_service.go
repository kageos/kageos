package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// RecordFormOperateLog 记录 Form 操作日志（form_submit/request_app）。
// 策略：社区版和企业版都走统一接口；社区版默认空实现，企业版会真正落库。
func (a *AppService) RecordFormOperateLog(ctx context.Context, req *dto.RecordFormOperateLogReq) error {
	if req == nil {
		return nil
	}
	if req.TenantUser == "" || req.App == "" {
		return fmt.Errorf("记录 Form 操作日志失败: tenant_user 和 app 不能为空")
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		app, err := a.appRepo.GetAppByUserName(req.TenantUser, req.App)
		if err != nil {
			return fmt.Errorf("获取应用信息失败: %w", err)
		}
		version = app.Version
	}

	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = "form_submit"
	}
	router := strings.TrimPrefix(strings.TrimSpace(req.Router), "/")
	resourceID := fmt.Sprintf("%s/%s", req.TenantUser, req.App)
	if router != "" {
		resourceID = fmt.Sprintf("%s/%s/%s", req.TenantUser, req.App, router)
	}

	operateLogReq := &dto.CreateOperateLoggerReq{
		User:         req.RequestUser,
		Action:       action,
		Resource:     functionschema.TypeForm,
		ResourceID:   resourceID,
		Source:       req.Source,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		TraceID:      req.TraceID,
		Router:       router,
		Method:       req.FunctionMethod,
		RequestBody:  req.RequestBody,
		ResponseBody: req.ResponseBody,
		Version:      version,
	}

	operateLogger := enterprise.GetOperateLogger()
	go func() {
		if _, err := operateLogger.CreateOperateLogger(operateLogReq); err != nil {
			logger.Warnf(ctx, "[RecordFormOperateLog] 记录 Form 操作日志失败: %v", err)
		}
	}()

	return nil
}
