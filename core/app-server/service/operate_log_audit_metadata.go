package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/contextx"
)

const (
	operateLogExecutorUser              = "user"
	operateLogExecutorAgent             = "agent"
	operateLogExecutorOpenAPI           = "openapi"
	operateLogExecutorPublicShare       = "public_share"
	operateLogExecutorScheduledFunction = "scheduled_function"
	operateLogExecutorUnknown           = "unknown"
)

type operateLogAuditMetadata struct {
	Source                string
	SourceType            string
	SourceRef             string
	ExecutorType          string
	WorkspaceSessionID    string
	WorkspaceSessionTitle string
	WorkspaceRole         string
}

func buildOperateLogAuditMetadata(ctx context.Context, sourceOverride string) operateLogAuditMetadata {
	source := strings.TrimSpace(sourceOverride)
	if source == "" {
		source = contextx.GetAuditClientSource(ctx)
	}
	workspaceSessionID := strings.TrimSpace(contextx.GetWorkspaceSessionID(ctx))
	meta := operateLogAuditMetadata{
		Source:                source,
		SourceType:            strings.TrimSpace(contextx.GetSourceType(ctx)),
		SourceRef:             strings.TrimSpace(contextx.GetSourceRef(ctx)),
		WorkspaceSessionID:    workspaceSessionID,
		WorkspaceSessionTitle: strings.TrimSpace(contextx.GetWorkspaceSessionTitle(ctx)),
		WorkspaceRole:         strings.TrimSpace(contextx.GetWorkspaceRole(ctx)),
	}
	meta.ExecutorType = inferOperateLogExecutorType(source, workspaceSessionID)
	return meta
}

func inferOperateLogExecutorType(source, workspaceSessionID string) string {
	if strings.TrimSpace(workspaceSessionID) != "" {
		return operateLogExecutorAgent
	}
	switch strings.TrimSpace(source) {
	case contextx.ClientSourceAgent:
		return operateLogExecutorAgent
	case contextx.ClientSourceOpenAPI, "api":
		return operateLogExecutorOpenAPI
	case contextx.ClientSourcePublicShare:
		return operateLogExecutorPublicShare
	case contextx.ClientSourceScheduledTask:
		return operateLogExecutorScheduledFunction
	case contextx.ClientSourceBrowser:
		return operateLogExecutorUser
	case "", contextx.ClientSourceUnknown:
		return operateLogExecutorUnknown
	default:
		return source
	}
}

func applyOperateLogAuditMetadata(log *model.OperateLog, meta operateLogAuditMetadata) {
	if log == nil {
		return
	}
	log.Source = meta.Source
	log.SourceType = meta.SourceType
	log.SourceRef = meta.SourceRef
	log.ExecutorType = meta.ExecutorType
	log.WorkspaceSessionID = meta.WorkspaceSessionID
	log.WorkspaceSessionTitle = meta.WorkspaceSessionTitle
	log.WorkspaceRole = meta.WorkspaceRole
}
