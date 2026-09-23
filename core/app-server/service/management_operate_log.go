package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

// A durable start record precedes external side effects. A crash leaves pending,
// not a fabricated success. Completion updates the same row synchronously.
type managementAudit struct {
	db      *gorm.DB
	log     *model.OperateLog
	details map[string]any
	started time.Time
	secrets []string
}

func beginManagementAudit(ctx context.Context, db *gorm.DB, action, resourceType, resourcePath, name, targetID string, before any, details map[string]any) (*managementAudit, error) {
	if db == nil {
		return nil, fmt.Errorf("operation audit storage unavailable")
	}
	resourcePath = access.NormalizeResourcePath(resourcePath)
	tenant, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return nil, err
	}
	if details == nil {
		details = map[string]any{}
	}
	details["stage"] = "started"
	eventID := uuid.NewString()
	log := &model.OperateLog{EventID: &eventID, TenantUser: tenant, App: app,
		ActorUser: contextx.GetRequestUser(ctx), Action: action, ResourceType: resourceType,
		ResourcePath: resourcePath, ResourceName: name, TargetID: targetID, Status: "pending",
		Summary: action + " " + resourcePath, OldValuesJSON: mustMarshalRaw(before), DetailsJSON: mustMarshalRaw(details), TraceID: contextx.GetTraceId(ctx)}
	applyOperateLogAuditMetadata(log, buildOperateLogAuditMetadata(ctx, ""))
	if err := db.WithContext(ctx).Create(log).Error; err != nil {
		return nil, fmt.Errorf("record operation start: %w", err)
	}
	return &managementAudit{db: db, log: log, details: details, started: time.Now()}, nil
}

func (a *managementAudit) stage(ctx context.Context, stage string) error {
	a.details["stage"] = stage
	return a.db.WithContext(ctx).Model(&model.OperateLog{}).Where("id = ?", a.log.ID).Update("details_json", mustMarshalRaw(a.details)).Error
}

// Register after the normal completion defer so it executes first during panic
// unwinding. Preserve the panic while preventing a nil named error from logging
// the interrupted operation as successful.
func (a *managementAudit) capturePanic(operationErr *error) {
	if cause := recover(); cause != nil {
		*operationErr = fmt.Errorf("operation interrupted before completion")
		panic(cause)
	}
}

func (a *managementAudit) finish(ctx context.Context, after any, operationErr *error) {
	status := "success"
	if *operationErr != nil {
		status = "failed"
		a.details["error"] = safeAuditError(*operationErr)
		for _, secret := range a.secrets {
			if secret != "" {
				a.details["error"] = strings.ReplaceAll(a.details["error"].(string), secret, "[redacted]")
			}
		}
	}
	a.details["duration_millis"] = time.Since(a.started).Milliseconds()
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	err := a.db.WithContext(writeCtx).Model(&model.OperateLog{}).Where("id = ?", a.log.ID).Updates(map[string]any{
		"status": status, "details_json": mustMarshalRaw(a.details), "new_values_json": mustMarshalRaw(after),
		"summary": fmt.Sprintf("%s %s (%s)", a.log.Action, a.log.ResourcePath, status),
	}).Error
	if err != nil {
		*operationErr = errors.Join(*operationErr, fmt.Errorf("operation result could not be recorded; audit %d remains pending: %w", a.log.ID, err))
	}
}

var auditURLPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

func safeAuditError(err error) string {
	if err == nil {
		return ""
	}
	// Network errors may echo signed download URLs or embedded basic auth.
	return auditURLPattern.ReplaceAllStringFunc(err.Error(), func(raw string) string {
		u, parseErr := url.Parse(raw)
		if parseErr != nil {
			return "[redacted URL]"
		}
		u.User, u.RawQuery, u.Fragment = nil, "", ""
		return u.String()
	})
}
