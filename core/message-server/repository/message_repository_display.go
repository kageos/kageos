package repository

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/dto"
)

func hydrateMessageSourceDisplays(items []dto.MessageInboxItem) {
	for i := range items {
		hydrateMessageSourceDisplay(&items[i])
	}
}

func hydrateMessageSourceDisplay(item *dto.MessageInboxItem) {
	if item == nil {
		return
	}
	item.ScheduledTaskID, item.ScheduledExecutionID = parseScheduledSourceRef(item.SourceRef)
	sourcePath := strings.TrimSpace(item.SourcePath)
	if sourcePath == "" {
		sourcePath = strings.TrimSpace(item.FullCodePath)
	}
	name := strings.TrimSpace(item.SourceTitle)
	if name == "" {
		name = pathBaseName(sourcePath)
	}
	if sourcePath == "" && name == "" {
		return
	}
	item.SourceDisplay = &dto.MessageSourceDisplay{
		Name:               name,
		Type:               strings.TrimSpace(item.SourceType),
		TemplateType:       strings.TrimSpace(item.SourceTemplateType),
		FullCodePath:       sourcePath,
		ParentName:         strings.TrimSpace(item.SourceParentTitle),
		ParentFullCodePath: strings.TrimSpace(item.SourceParentPath),
		ThreadKey:          strings.TrimSpace(item.ThreadKey),
	}
}

func threadTitle(item dto.MessageInboxItem) string {
	display := item.SourceDisplay
	var displayParentName, displayName string
	if display != nil {
		displayParentName = display.ParentName
		displayName = display.Name
	}
	for _, value := range []string{
		displayParentName,
		item.SourceParentTitle,
		displayName,
		item.SourceTitle,
		item.From,
		"system",
	} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "system"
}

func threadSubtitle(item dto.MessageInboxItem, count int64) string {
	sourceName := sourceSecondaryText(item)
	if count > 1 {
		return fmt.Sprintf("%s · %d 条消息", sourceName, count)
	}
	return sourceName
}

func sourceSecondaryText(item dto.MessageInboxItem) string {
	var displayName, displayParentName string
	if item.SourceDisplay != nil {
		displayName = item.SourceDisplay.Name
		displayParentName = item.SourceDisplay.ParentName
	}
	functionName := strings.TrimSpace(displayName)
	if functionName == "" {
		functionName = strings.TrimSpace(item.SourceTitle)
	}
	parentName := strings.TrimSpace(displayParentName)
	if parentName == "" {
		parentName = strings.TrimSpace(item.SourceParentTitle)
	}
	if functionName != "" && functionName != parentName {
		return functionName
	}
	if strings.TrimSpace(item.WorkspaceSessionTitle) != "" {
		return strings.TrimSpace(item.WorkspaceSessionTitle)
	}
	for _, value := range []string{item.SourcePath, item.FullCodePath, item.From, "-"} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "-"
}

func threadPath(item dto.MessageInboxItem) string {
	if item.SourceDisplay != nil {
		if path := strings.TrimSpace(item.SourceDisplay.ParentFullCodePath); path != "" {
			return path
		}
		if path := strings.TrimSpace(item.SourceDisplay.FullCodePath); path != "" {
			return path
		}
	}
	if path := strings.TrimSpace(item.SourceParentPath); path != "" {
		return path
	}
	if path := strings.TrimSpace(item.SourcePath); path != "" {
		return path
	}
	return strings.TrimSpace(item.FullCodePath)
}

func threadKind(item dto.MessageInboxItem) string {
	displayParentPath := ""
	if item.SourceDisplay != nil {
		displayParentPath = item.SourceDisplay.ParentFullCodePath
	}
	if strings.TrimSpace(item.SourceParentPath) != "" || strings.TrimSpace(displayParentPath) != "" {
		return "directory"
	}
	if strings.TrimSpace(item.WorkspaceSessionID) != "" {
		return "session"
	}
	if strings.TrimSpace(item.SourcePath) != "" || strings.TrimSpace(item.FullCodePath) != "" {
		return "function"
	}
	return "sender"
}

func parseScheduledSourceRef(sourceRef string) (int64, int64) {
	parts := strings.Split(strings.TrimSpace(sourceRef), ":")
	var taskID int64
	var executionID int64
	for i := 0; i < len(parts)-1; i++ {
		switch parts[i] {
		case "timer_task":
			taskID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		case "execution", "timer_execution":
			executionID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
	}
	return taskID, executionID
}

func pathBaseName(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}
