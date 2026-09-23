package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RecordTaskAudit(ctx context.Context, db *gorm.DB, event dto.TaskAuditEvent) error {
	switch event.Action {
	case "timer.task.created", "timer.task.updated", "timer.task.paused", "timer.task.resumed", "timer.task.cancelled", "timer.task.deleted", "timer.task.run_now":
	default:
		return fmt.Errorf("invalid task audit action")
	}
	if event.EventID == "" || len(event.EventID) > 160 || event.OccurredAt.IsZero() || event.Actor == "" || (event.Status != "success" && event.Status != "failed") {
		return fmt.Errorf("invalid task audit event")
	}
	tenant, app, err := access.ParseUserApp(event.ResourcePath)
	if err != nil {
		return err
	}
	details := map[string]any{"task_id": event.TaskID, "execution_id": event.ExecutionID, "stage": "completed"}
	if event.Status == "failed" {
		details["error"] = safeAuditError(fmt.Errorf("%s", event.Error))
		details["stage"] = "failed"
	}
	log := &model.OperateLog{EventID: &event.EventID, TenantUser: tenant, App: app, ActorUser: event.Actor,
		Action: event.Action, ResourceType: "scheduled_task", ResourcePath: access.NormalizeResourcePath(event.ResourcePath),
		ResourceName: event.ResourceName, TargetID: strconv.FormatInt(event.TaskID, 10), Status: event.Status,
		Summary: event.Action + " " + event.ResourceName, DetailsJSON: mustMarshalRaw(details),
		OldValuesJSON: mustMarshalRaw(event.Before), NewValuesJSON: mustMarshalRaw(event.After),
		Source: event.Source, SourceType: event.SourceType, SourceRef: event.SourceRef, InitiatorUser: event.Initiator,
		WorkspaceSessionID: event.SessionID, ToolCallID: event.ToolCallID, TraceID: event.TraceID,
		ExecutorType: inferOperateLogExecutorType(event.Source, event.SessionID)}
	log.CreatedAt = models.Time(event.OccurredAt)
	return db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_id"}}, DoNothing: true}).Create(log).Error
}
