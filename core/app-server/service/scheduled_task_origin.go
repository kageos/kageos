package service

import (
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/scheduledsdk"
)

const (
	scheduledTaskOriginManifest = "manifest"
	scheduledTaskOriginUser     = "user"
	scheduledTaskSourceManifest = "app_manifest"
	scheduledTaskSourceBundle   = "capability_bundle"
)

func scheduledTaskOrigin(task *scheduledsdk.Task) string {
	if task == nil {
		return scheduledTaskOriginUser
	}
	metadata := task.Metadata
	if origin := strings.TrimSpace(metadata["origin"]); origin != "" {
		return origin
	}
	switch strings.TrimSpace(metadata["managed_by"]) {
	case "app_manifest", "capability_bundle":
		return scheduledTaskOriginManifest
	default:
		return scheduledTaskOriginUser
	}
}

func isBuiltinScheduledTask(task *scheduledsdk.Task) bool {
	return scheduledTaskOrigin(task) == scheduledTaskOriginManifest
}

func scheduledTaskDefaultEnabled(task *scheduledsdk.Task) bool {
	if task == nil {
		return false
	}
	if raw := strings.TrimSpace(task.Metadata["default_enabled"]); raw != "" {
		if enabled, err := strconv.ParseBool(raw); err == nil {
			return enabled
		}
	}
	return task.Status == scheduledsdk.TaskStatusPending
}

func shouldExportScheduledTask(task *scheduledsdk.Task, selectedUserTaskIDs map[int64]struct{}) bool {
	if task == nil {
		return false
	}
	if isBuiltinScheduledTask(task) {
		return true
	}
	_, selected := selectedUserTaskIDs[task.ID]
	return selected
}
