package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	DurationMillis    int64                     `json:"duration_millis"`
	SourceFileCount   int                       `json:"source_file_count,omitempty"`
	WriteOnly         bool                      `json:"write_only,omitempty"`
	ForceDiff         bool                      `json:"force_diff,omitempty"`
	GitCommitHash     string                    `json:"git_commit_hash,omitempty"`
	BuildTraceID      string                    `json:"build_trace_id,omitempty"`
	Requirement       string                    `json:"requirement,omitempty"`
	ChangeDescription string                    `json:"change_description,omitempty"`
	Error             string                    `json:"error,omitempty"`
	Outcome           string                    `json:"outcome"`
	Warnings          []string                  `json:"warnings,omitempty"`
	Changes           *workspaceFunctionChanges `json:"changes,omitempty"`
}

// Keep historical names and paths, not live references or full schemas/source.
type workspaceFunctionChange struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

type workspaceFunctionChanges struct {
	Mode    string                    `json:"mode"`
	Added   []workspaceFunctionChange `json:"added"`
	Updated []workspaceFunctionChange `json:"updated"`
	Deleted []workspaceFunctionChange `json:"deleted"`
	Synced  []workspaceFunctionChange `json:"synced"`
}

func workspaceFunctionSnapshots(apis []*dto.ApiInfo) []workspaceFunctionChange {
	items := make([]workspaceFunctionChange, 0, len(apis))
	seen := make(map[string]bool)
	for _, api := range apis {
		if api == nil {
			continue
		}
		path := strings.TrimSpace(api.FullCodePath)
		if path == "" && api.User != "" && api.App != "" && api.Router != "" {
			path = api.BuildFullCodePath()
		}
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		items = append(items, workspaceFunctionChange{Name: api.Name, Path: path, Type: api.TemplateType})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items
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
	if update, ok := details.(workspaceUpdateOperateLogDetails); ok {
		if update.Outcome == "write_only" {
			summary = fmt.Sprintf("%s wrote workspace source files in /%s/%s without publishing", contextx.GetRequestUser(ctx), app.User, app.Code)
		}
		if changes := update.Changes; changes != nil {
			if changes.Mode == "resync" {
				summary += fmt.Sprintf("; full synchronization: %d functions reported (not an incremental diff)", len(changes.Synced))
			} else {
				summary += fmt.Sprintf("; reported function changes: %d added, %d updated, %d deleted", len(changes.Added), len(changes.Updated), len(changes.Deleted))
			}
		}
		if len(update.Warnings) > 0 {
			summary += "; published with warnings"
		}
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
		Outcome:        "runtime_failed",
	}
	if req != nil {
		details.SourceFileCount = len(req.SourceFiles)
		details.WriteOnly = req.WriteOnly
		details.ForceDiff = req.ForceDiff
		details.Requirement = req.Requirement
		details.ChangeDescription = req.ChangeDescription
	}
	if resp != nil {
		details.Outcome = "completed"
		details.GitCommitHash = resp.GitCommitHash
		details.BuildTraceID = updateAppTraceID(resp)
		details.Warnings = append([]string{}, resp.Warnings...)
		if resp.Error != "" {
			details.Warnings = append(details.Warnings, resp.Error)
		}
		if len(details.Warnings) > 0 {
			details.Outcome = "metadata_warning"
		}
		if resp.Diff != nil && !details.WriteOnly {
			diff := resp.Diff
			details.Changes = &workspaceFunctionChanges{
				Mode:    "diff",
				Added:   workspaceFunctionSnapshots(diff.Add),
				Updated: workspaceFunctionSnapshots(diff.Update),
				Deleted: workspaceFunctionSnapshots(diff.Delete),
				Synced:  []workspaceFunctionChange{},
			}
			// ForceDiff clears the SDK baseline: these are synchronization entries,
			// not evidence that the functions were newly created or modified.
			if details.ForceDiff {
				apis := append([]*dto.ApiInfo{}, diff.Add...)
				apis = append(apis, diff.Update...)
				details.Changes = &workspaceFunctionChanges{
					Mode: "resync", Synced: workspaceFunctionSnapshots(apis),
					Added: []workspaceFunctionChange{}, Updated: []workspaceFunctionChange{}, Deleted: []workspaceFunctionChange{},
				}
			}
		}
		if details.WriteOnly {
			details.Outcome = "write_only"
		}
	}
	if updateErr != nil {
		details.Error = updateErr.Error()
		if resp != nil {
			details.Outcome = "finalization_failed"
		}
	}
	return details
}
