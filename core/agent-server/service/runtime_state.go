package service

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

const (
	RuntimeStateKindWorkspaceSession = "workspace_session"

	RuntimeStateStatusThinking        = "thinking"
	RuntimeStateStatusRunning         = "running"
	RuntimeStateStatusToolRunning     = "tool_running"
	RuntimeStateStatusWaitingApproval = "waiting_approval"
	RuntimeStateStatusFailed          = "failed"
	RuntimeStateStatusCancelled       = "cancelled"
)

type RuntimeStateFilter struct {
	RootFullCodePath string
	Kind             string
	Status           string
}

type RuntimeStateStore interface {
	Upsert(ctx context.Context, item dto.RuntimeStateItem) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, filter RuntimeStateFilter) ([]dto.RuntimeStateItem, error)
	Summary(ctx context.Context, filter RuntimeStateFilter) (map[string]dto.RuntimeStateSummary, error)
}

type InMemoryRuntimeStateStore struct {
	mu    sync.RWMutex
	items map[string]dto.RuntimeStateItem
}

func NewInMemoryRuntimeStateStore() *InMemoryRuntimeStateStore {
	return &InMemoryRuntimeStateStore{
		items: make(map[string]dto.RuntimeStateItem),
	}
}

func (s *InMemoryRuntimeStateStore) Upsert(_ context.Context, item dto.RuntimeStateItem) error {
	key := strings.TrimSpace(item.Key)
	if key == "" {
		return nil
	}
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)

	if existing, ok := s.items[key]; ok {
		if item.StartedAt.IsZero() {
			item.StartedAt = existing.StartedAt
		}
		if item.FullCodePath == "" {
			item.FullCodePath = existing.FullCodePath
		}
		if item.Kind == "" {
			item.Kind = existing.Kind
		}
		if item.SessionID == "" {
			item.SessionID = existing.SessionID
		}
		if item.SourceType == "" {
			item.SourceType = existing.SourceType
		}
		if item.SourceRef == "" {
			item.SourceRef = existing.SourceRef
		}
		if item.User == "" {
			item.User = existing.User
		}
		if item.Title == "" {
			item.Title = existing.Title
		}
		if item.ModeCode == "" {
			item.ModeCode = existing.ModeCode
		}
		if item.Metadata == nil {
			item.Metadata = existing.Metadata
		}
	}
	if item.StartedAt.IsZero() {
		item.StartedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = now
	}
	item.Key = key
	item.FullCodePath = normalizeRuntimeFullCodePath(item.FullCodePath)
	s.items[key] = item
	return nil
}

func (s *InMemoryRuntimeStateStore) Delete(_ context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

func (s *InMemoryRuntimeStateStore) List(_ context.Context, filter RuntimeStateFilter) ([]dto.RuntimeStateItem, error) {
	now := time.Now()
	root := normalizeRuntimeFullCodePath(filter.RootFullCodePath)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)

	out := make([]dto.RuntimeStateItem, 0, len(s.items))
	for _, item := range s.items {
		if filter.Kind != "" && item.Kind != filter.Kind {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if root != "" && !runtimePathHasPrefix(item.FullCodePath, root) {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *InMemoryRuntimeStateStore) Summary(ctx context.Context, filter RuntimeStateFilter) (map[string]dto.RuntimeStateSummary, error) {
	items, err := s.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	root := normalizeRuntimeFullCodePath(filter.RootFullCodePath)
	summaries := make(map[string]dto.RuntimeStateSummary)
	for _, item := range items {
		for _, aggregatePath := range runtimeAggregatePaths(root, item.FullCodePath) {
			summary := summaries[aggregatePath]
			applyRuntimeStateToSummary(&summary, item)
			summaries[aggregatePath] = summary
		}
	}
	return summaries, nil
}

func (s *InMemoryRuntimeStateStore) cleanupExpiredLocked(now time.Time) {
	for key, item := range s.items {
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			delete(s.items, key)
		}
	}
}

func applyRuntimeStateToSummary(summary *dto.RuntimeStateSummary, item dto.RuntimeStateItem) {
	switch item.Status {
	case RuntimeStateStatusFailed:
		summary.FailedRecentCount++
	case RuntimeStateStatusCancelled:
		// cancelled 不计入运行中，仅作为短暂状态保留给明细查询。
	default:
		summary.RunningCount++
		summary.ManualRunningCount++
	}

	switch item.Status {
	case RuntimeStateStatusThinking:
		summary.ThinkingCount++
	case RuntimeStateStatusToolRunning:
		summary.ToolRunningCount++
	case RuntimeStateStatusWaitingApproval:
		summary.WaitingApprovalCount++
	}

	if item.UpdatedAt.After(summary.LastActivityAt) {
		summary.LastActivityAt = item.UpdatedAt
	}
	refreshRuntimeSummaryDisplay(summary)
}

func refreshRuntimeSummaryDisplay(summary *dto.RuntimeStateSummary) {
	switch {
	case summary.WaitingApprovalCount > 0:
		summary.DominantStatus = RuntimeStateStatusWaitingApproval
		summary.BadgeTone = "approval"
	case summary.ToolRunningCount > 0:
		summary.DominantStatus = RuntimeStateStatusToolRunning
		summary.BadgeTone = "tool"
	case summary.ThinkingCount > 0:
		summary.DominantStatus = RuntimeStateStatusThinking
		summary.BadgeTone = "thinking"
	case summary.RunningCount > 0:
		summary.DominantStatus = RuntimeStateStatusRunning
		summary.BadgeTone = "running"
	case summary.FailedRecentCount > 0:
		summary.DominantStatus = RuntimeStateStatusFailed
		summary.BadgeTone = "failed"
	default:
		summary.DominantStatus = ""
		summary.BadgeTone = ""
	}

	if summary.RunningCount > 0 {
		summary.BadgeText = strconv.Itoa(summary.RunningCount)
	} else if summary.FailedRecentCount > 0 {
		summary.BadgeText = "!"
	} else {
		summary.BadgeText = ""
	}

	parts := make([]string, 0, 5)
	if summary.RunningCount > 0 {
		parts = append(parts, fmt.Sprintf("运行中 %d", summary.RunningCount))
	}
	if summary.ThinkingCount > 0 {
		parts = append(parts, fmt.Sprintf("思考中 %d", summary.ThinkingCount))
	}
	if summary.ToolRunningCount > 0 {
		parts = append(parts, fmt.Sprintf("工具执行 %d", summary.ToolRunningCount))
	}
	if summary.FailedRecentCount > 0 {
		parts = append(parts, fmt.Sprintf("最近失败 %d", summary.FailedRecentCount))
	}
	summary.Tooltip = strings.Join(parts, "，")
}

func normalizeRuntimeFullCodePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	cleaned := path.Clean(raw)
	if cleaned == "." || cleaned == "/" {
		return ""
	}
	return cleaned
}

func runtimePathHasPrefix(fullCodePath, root string) bool {
	fullCodePath = normalizeRuntimeFullCodePath(fullCodePath)
	root = normalizeRuntimeFullCodePath(root)
	if root == "" {
		return fullCodePath != ""
	}
	return fullCodePath == root || strings.HasPrefix(fullCodePath, root+"/")
}

func runtimeAggregatePaths(root, fullCodePath string) []string {
	fullCodePath = normalizeRuntimeFullCodePath(fullCodePath)
	root = normalizeRuntimeFullCodePath(root)
	if fullCodePath == "" || (root != "" && !runtimePathHasPrefix(fullCodePath, root)) {
		return nil
	}
	if root == "" {
		return runtimePathAncestors(fullCodePath)
	}
	out := []string{root}
	if fullCodePath == root {
		return out
	}
	relative := strings.TrimPrefix(fullCodePath, root+"/")
	current := root
	for _, segment := range strings.Split(relative, "/") {
		if segment == "" {
			continue
		}
		current += "/" + segment
		out = append(out, current)
	}
	return out
}

func runtimePathAncestors(fullCodePath string) []string {
	fullCodePath = normalizeRuntimeFullCodePath(fullCodePath)
	if fullCodePath == "" {
		return nil
	}
	segments := strings.Split(strings.TrimPrefix(fullCodePath, "/"), "/")
	out := make([]string, 0, len(segments))
	current := ""
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		current += "/" + segment
		out = append(out, current)
	}
	return out
}
