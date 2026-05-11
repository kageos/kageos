package dto

import (
	"encoding/json"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
)

type CreateWorkflowRequest struct {
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	AppID        int64           `json:"app_id"`
	FullCodePath string          `json:"full_code_path"`
	Definition   json.RawMessage `json:"definition"`
}

type UpdateWorkflowRequest struct {
	Name         *string          `json:"name,omitempty"`
	Description  *string          `json:"description,omitempty"`
	AppID        *int64           `json:"app_id,omitempty"`
	FullCodePath *string          `json:"full_code_path,omitempty"`
	Status       *string          `json:"status,omitempty"`
	Definition   *json.RawMessage `json:"definition,omitempty"`
}

type PublishWorkflowRequest struct {
	Definition json.RawMessage `json:"definition,omitempty"`
}

type RunWorkflowRequest struct {
	VersionID int64                  `json:"version_id,omitempty"`
	Input     map[string]interface{} `json:"input"`
}

type WorkflowItem struct {
	ID                  int64           `json:"id"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	AppID               int64           `json:"app_id"`
	FullCodePath        string          `json:"full_code_path"`
	Status              string          `json:"status"`
	LatestVersionID     int64           `json:"latest_version_id"`
	CreatedBy           string          `json:"created_by"`
	UpdatedBy           string          `json:"updated_by"`
	DraftDefinitionJSON json.RawMessage `json:"draft_definition_json,omitempty"`
}

type WorkflowListResponse struct {
	List  []*WorkflowItem `json:"list"`
	Total int64           `json:"total"`
}

type WorkflowVersionItem struct {
	ID               int64           `json:"id"`
	CreatedAt        time.Time       `json:"created_at"`
	WorkflowID       int64           `json:"workflow_id"`
	Version          int             `json:"version"`
	DefinitionJSON   json.RawMessage `json:"definition_json"`
	InputSchemaJSON  json.RawMessage `json:"input_schema_json,omitempty"`
	OutputSchemaJSON json.RawMessage `json:"output_schema_json,omitempty"`
	Status           string          `json:"status"`
	CreatedBy        string          `json:"created_by"`
}

type WorkflowRunItem struct {
	ID              int64           `json:"id"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	WorkflowID      int64           `json:"workflow_id"`
	VersionID       int64           `json:"version_id"`
	Status          string          `json:"status"`
	InputJSON       json.RawMessage `json:"input_json,omitempty"`
	OutputJSON      json.RawMessage `json:"output_json,omitempty"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	RequestUser     string          `json:"request_user"`
	RequestUserDept string          `json:"request_user_dept"`
	TraceID         string          `json:"trace_id"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	FinishedAt      *time.Time      `json:"finished_at,omitempty"`
	DurationMillis  int64           `json:"duration_millis"`
}

type WorkflowStepRunItem struct {
	ID             int64           `json:"id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	RunID          int64           `json:"run_id"`
	StepID         string          `json:"step_id"`
	StepName       string          `json:"step_name"`
	NodeType       string          `json:"node_type"`
	NodeRef        string          `json:"node_ref"`
	Status         string          `json:"status"`
	InputJSON      json.RawMessage `json:"input_json,omitempty"`
	OutputJSON     json.RawMessage `json:"output_json,omitempty"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	TraceID        string          `json:"trace_id"`
	Attempt        int             `json:"attempt"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	DurationMillis int64           `json:"duration_millis"`
}

type WorkflowRunDetail struct {
	Run   *WorkflowRunItem       `json:"run"`
	Steps []*WorkflowStepRunItem `json:"steps"`
}

func ToWorkflowItem(item *model.WorkflowDefinition) *WorkflowItem {
	if item == nil {
		return nil
	}
	return &WorkflowItem{
		ID:                  item.ID,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
		Name:                item.Name,
		Description:         item.Description,
		AppID:               item.AppID,
		FullCodePath:        item.FullCodePath,
		Status:              item.Status,
		LatestVersionID:     item.LatestVersionID,
		CreatedBy:           item.CreatedBy,
		UpdatedBy:           item.UpdatedBy,
		DraftDefinitionJSON: item.DraftDefinitionJSON,
	}
}

func ToWorkflowVersionItem(item *model.WorkflowDefinitionVersion) *WorkflowVersionItem {
	if item == nil {
		return nil
	}
	return &WorkflowVersionItem{
		ID:               item.ID,
		CreatedAt:        item.CreatedAt,
		WorkflowID:       item.WorkflowID,
		Version:          item.Version,
		DefinitionJSON:   item.DefinitionJSON,
		InputSchemaJSON:  item.InputSchemaJSON,
		OutputSchemaJSON: item.OutputSchemaJSON,
		Status:           item.Status,
		CreatedBy:        item.CreatedBy,
	}
}

func ToWorkflowRunItem(item *model.WorkflowRun) *WorkflowRunItem {
	if item == nil {
		return nil
	}
	return &WorkflowRunItem{
		ID:              item.ID,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		WorkflowID:      item.WorkflowID,
		VersionID:       item.VersionID,
		Status:          item.Status,
		InputJSON:       item.InputJSON,
		OutputJSON:      item.OutputJSON,
		ErrorMessage:    item.ErrorMessage,
		RequestUser:     item.RequestUser,
		RequestUserDept: item.RequestUserDept,
		TraceID:         item.TraceID,
		StartedAt:       item.StartedAt,
		FinishedAt:      item.FinishedAt,
		DurationMillis:  item.DurationMillis,
	}
}

func ToWorkflowStepRunItem(item *model.WorkflowStepRun) *WorkflowStepRunItem {
	if item == nil {
		return nil
	}
	return &WorkflowStepRunItem{
		ID:             item.ID,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
		RunID:          item.RunID,
		StepID:         item.StepID,
		StepName:       item.StepName,
		NodeType:       item.NodeType,
		NodeRef:        item.NodeRef,
		Status:         item.Status,
		InputJSON:      item.InputJSON,
		OutputJSON:     item.OutputJSON,
		ErrorMessage:   item.ErrorMessage,
		TraceID:        item.TraceID,
		Attempt:        item.Attempt,
		StartedAt:      item.StartedAt,
		FinishedAt:     item.FinishedAt,
		DurationMillis: item.DurationMillis,
	}
}
