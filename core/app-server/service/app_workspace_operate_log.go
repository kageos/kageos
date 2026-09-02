package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

const (
	workspaceUpdatedAction         = "workspace.updated"
	workspaceSettingsUpdatedAction = "workspace.settings.updated"
)

type workspaceOperateLogValues struct {
	Name                  string `json:"name,omitempty"`
	Admins                string `json:"admins,omitempty"`
	IsPublic              bool   `json:"is_public"`
	HideUnauthorizedNodes bool   `json:"hide_unauthorized_nodes"`
	Version               string `json:"version,omitempty"`
}

type workspaceUpdateOperateLogDetails struct {
	DurationMillis    int64  `json:"duration_millis"`
	SourceFileCount   int    `json:"source_file_count,omitempty"`
	WriteOnly         bool   `json:"write_only,omitempty"`
	ForceDiff         bool   `json:"force_diff,omitempty"`
	GitCommitHash     string `json:"git_commit_hash,omitempty"`
	BuildTraceID      string `json:"build_trace_id,omitempty"`
	Requirement       string `json:"requirement,omitempty"`
	ChangeDescription string `json:"change_description,omitempty"`
	Error             string `json:"error,omitempty"`
}

func workspaceOperateLogSnapshot(app *model.App) *workspaceOperateLogValues {
	if app == nil {
		return nil
	}
	return &workspaceOperateLogValues{
		Name:                  app.Name,
		Admins:                app.Admins,
		IsPublic:              app.IsPublic,
		HideUnauthorizedNodes: app.HideUnauthorizedNodes,
		Version:               app.Version,
	}
}

func workspaceUpdateResponseSnapshot(app *model.App, resp *dto.UpdateAppResp) *workspaceOperateLogValues {
	values := workspaceOperateLogSnapshot(app)
	if values == nil {
		values = &workspaceOperateLogValues{}
	}
	if resp != nil && resp.NewVersion != "" {
		values.Version = resp.NewVersion
	}
	return values
}

func (a *AppService) writeWorkspaceOperateLog(
	ctx context.Context,
	app *model.App,
	action string,
	status string,
	summary string,
	details interface{},
	oldValues interface{},
	newValues interface{},
) {
	if a == nil || app == nil {
		return
	}
	writer := a.permission
	if writer == nil || writer.operateLogRepo == nil {
		writer = nil
	}
	if writer == nil && a.operateLogRepo != nil {
		writer = &PermissionService{operateLogRepo: a.operateLogRepo}
	}
	if writer == nil {
		return
	}
	writer.writeOperateLog(ctx, operateLogInput{
		TenantUser:   app.User,
		App:          app.Code,
		ActorUser:    contextx.GetRequestUser(ctx),
		Action:       action,
		ResourceType: "workspace",
		ResourcePath: fmt.Sprintf("/%s/%s", app.User, app.Code),
		ResourceName: app.Name,
		TargetID:     fmt.Sprintf("%d", app.ID),
		Summary:      summary,
		Details:      details,
		OldValues:    oldValues,
		NewValues:    newValues,
		Status:       status,
	})
}

func workspaceUpdateOperateLogDetailsFromRequest(req *dto.UpdateAppReq, resp *dto.UpdateAppResp, startedAt time.Time, updateErr error) workspaceUpdateOperateLogDetails {
	details := workspaceUpdateOperateLogDetails{
		DurationMillis: time.Since(startedAt).Milliseconds(),
	}
	if req != nil {
		details.SourceFileCount = len(req.SourceFiles)
		details.WriteOnly = req.WriteOnly
		details.ForceDiff = req.ForceDiff
		details.Requirement = req.Requirement
		details.ChangeDescription = req.ChangeDescription
	}
	if resp != nil {
		details.GitCommitHash = resp.GitCommitHash
		details.BuildTraceID = updateAppTraceID(resp)
	}
	if updateErr != nil {
		details.Error = updateErr.Error()
	}
	return details
}
