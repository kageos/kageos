package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

type operateLogInput struct {
	TenantUser   string
	CompanyCode  string
	App          string
	ActorUser    string
	Action       string
	ResourceType string
	ResourcePath string
	ResourceName string
	TargetUser   string
	TargetID     string
	Summary      string
	Details      interface{}
	OldValues    interface{}
	NewValues    interface{}
	Status       string
}

func (s *TeamAccessService) writeOperateLog(ctx context.Context, input operateLogInput) {
	if s.operateLogRepo == nil {
		return
	}
	if input.ActorUser == "" {
		input.ActorUser = contextx.GetRequestUser(ctx)
	}
	if input.Status == "" {
		input.Status = "success"
	}
	auditMeta := buildOperateLogAuditMetadata(ctx, "")
	log := &model.OperateLog{
		TenantUser:   input.TenantUser,
		CompanyCode:  firstNonEmpty(input.CompanyCode, contextx.GetRequestCompanyCode(ctx)),
		App:          input.App,
		ActorUser:    input.ActorUser,
		Action:       input.Action,
		ResourceType: input.ResourceType,
		ResourcePath: access.NormalizeResourcePath(input.ResourcePath),
		ResourceName: input.ResourceName,
		TargetUser:   input.TargetUser,
		TargetID:     input.TargetID,
		Summary:      input.Summary,
		Status:       input.Status,
		TraceID:      contextx.GetTraceId(ctx),
	}
	applyOperateLogAuditMetadata(log, auditMeta)
	log.DetailsJSON = mustMarshalRaw(input.Details)
	log.OldValuesJSON = mustMarshalRaw(input.OldValues)
	log.NewValuesJSON = mustMarshalRaw(input.NewValues)

	writeCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context) {
		if err := s.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
			logger.Warnf(ctx, "[TeamAccess] write operate log failed: action=%s err=%v", input.Action, err)
		}
	}(writeCtx)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mustMarshalRaw(v interface{}) json.RawMessage {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(redactOperateLogValue(v))
	if err != nil {
		return nil
	}
	return data
}
