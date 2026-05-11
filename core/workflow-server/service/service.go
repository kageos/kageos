package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	workflowdto "github.com/ai-agent-os/ai-agent-os/core/workflow-server/dto"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/executor"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/gorm"
)

const defaultListPageSize = 20

type Service struct {
	db       *gorm.DB
	defRepo  *repository.WorkflowDefinitionRepository
	verRepo  *repository.WorkflowVersionRepository
	runRepo  *repository.WorkflowRunRepository
	stepRepo *repository.WorkflowStepRunRepository
	runner   *Runner
}

func NewService(db *gorm.DB, registry *executor.Registry) *Service {
	defRepo := repository.NewWorkflowDefinitionRepository(db)
	verRepo := repository.NewWorkflowVersionRepository(db)
	runRepo := repository.NewWorkflowRunRepository(db)
	stepRepo := repository.NewWorkflowStepRunRepository(db)
	return &Service{
		db:       db,
		defRepo:  defRepo,
		verRepo:  verRepo,
		runRepo:  runRepo,
		stepRepo: stepRepo,
		runner: NewRunner(RunnerDeps{
			RunRepo:  runRepo,
			StepRepo: stepRepo,
			Registry: registry,
			Now:      time.Now,
		}),
	}
}

func (s *Service) CreateWorkflow(ctx context.Context, req workflowdto.CreateWorkflowRequest) (*workflowdto.WorkflowItem, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	definitionJSON := normalizeRawJSON(req.Definition)
	if len(definitionJSON) > 0 {
		if err := s.validateDefinition(definitionJSON); err != nil {
			return nil, err
		}
	}
	user := contextx.GetRequestUser(ctx)
	item := &model.WorkflowDefinition{
		Name:                name,
		Description:         strings.TrimSpace(req.Description),
		AppID:               req.AppID,
		FullCodePath:        strings.TrimSpace(req.FullCodePath),
		Status:              model.WorkflowStatusDraft,
		CreatedBy:           user,
		UpdatedBy:           user,
		DraftDefinitionJSON: definitionJSON,
	}
	if err := s.defRepo.Create(item); err != nil {
		return nil, err
	}
	return workflowdto.ToWorkflowItem(item), nil
}

func (s *Service) UpdateWorkflow(ctx context.Context, id int64, req workflowdto.UpdateWorkflowRequest) (*workflowdto.WorkflowItem, error) {
	item, err := s.defRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, fmt.Errorf("name cannot be empty")
		}
		item.Name = name
	}
	if req.Description != nil {
		item.Description = strings.TrimSpace(*req.Description)
	}
	if req.AppID != nil {
		item.AppID = *req.AppID
	}
	if req.FullCodePath != nil {
		item.FullCodePath = strings.TrimSpace(*req.FullCodePath)
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !isWorkflowStatus(status) {
			return nil, fmt.Errorf("invalid workflow status: %s", status)
		}
		item.Status = status
	}
	if req.Definition != nil {
		raw := normalizeRawJSON(*req.Definition)
		if len(raw) > 0 {
			if err := s.validateDefinition(raw); err != nil {
				return nil, err
			}
		}
		item.DraftDefinitionJSON = raw
	}
	item.UpdatedBy = contextx.GetRequestUser(ctx)
	if err := s.defRepo.Update(item); err != nil {
		return nil, err
	}
	return workflowdto.ToWorkflowItem(item), nil
}

func (s *Service) ListWorkflows(ctx context.Context, status string, appID int64, page, pageSize int) (*workflowdto.WorkflowListResponse, error) {
	page, pageSize = normalizePage(page, pageSize)
	list, total, err := s.defRepo.List(repository.ListWorkflowDefinitionsFilter{
		Status: strings.TrimSpace(status),
		AppID:  appID,
		Offset: (page - 1) * pageSize,
		Limit:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*workflowdto.WorkflowItem, 0, len(list))
	for _, item := range list {
		out = append(out, workflowdto.ToWorkflowItem(item))
	}
	return &workflowdto.WorkflowListResponse{List: out, Total: total}, nil
}

func (s *Service) GetWorkflow(ctx context.Context, id int64) (*workflowdto.WorkflowItem, error) {
	item, err := s.defRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return workflowdto.ToWorkflowItem(item), nil
}

func (s *Service) PublishWorkflow(ctx context.Context, id int64, req workflowdto.PublishWorkflowRequest) (*workflowdto.WorkflowVersionItem, error) {
	var version *model.WorkflowDefinitionVersion
	err := s.db.Transaction(func(tx *gorm.DB) error {
		defRepo := s.defRepo.WithDB(tx)
		verRepo := s.verRepo.WithDB(tx)
		workflow, err := defRepo.GetByID(id)
		if err != nil {
			return err
		}
		raw := normalizeRawJSON(req.Definition)
		if len(raw) == 0 {
			raw = normalizeRawJSON(workflow.DraftDefinitionJSON)
		}
		if len(raw) == 0 {
			return fmt.Errorf("workflow definition is empty")
		}
		parsed, err := definition.Parse(raw)
		if err != nil {
			return err
		}
		if err := parsed.Validate(definition.ValidateOptions{SupportedNodeTypes: definition.SupportedMVPNodeTypes()}); err != nil {
			return err
		}
		next, err := verRepo.NextVersion(id)
		if err != nil {
			return err
		}
		inputSchema, _ := json.Marshal(parsed.Inputs)
		outputSchema, _ := json.Marshal(parsed.Outputs)
		version = &model.WorkflowDefinitionVersion{
			WorkflowID:       id,
			Version:          next,
			DefinitionJSON:   raw,
			InputSchemaJSON:  inputSchema,
			OutputSchemaJSON: outputSchema,
			Status:           model.VersionStatusPublished,
			CreatedBy:        contextx.GetRequestUser(ctx),
		}
		if err := verRepo.Create(version); err != nil {
			return err
		}
		workflow.DraftDefinitionJSON = raw
		workflow.UpdatedBy = contextx.GetRequestUser(ctx)
		if err := defRepo.Update(workflow); err != nil {
			return err
		}
		return defRepo.SetLatestVersion(id, version.ID, model.WorkflowStatusEnabled)
	})
	if err != nil {
		return nil, err
	}
	return workflowdto.ToWorkflowVersionItem(version), nil
}

func (s *Service) RunWorkflow(ctx context.Context, workflowID int64, req workflowdto.RunWorkflowRequest) (*workflowdto.WorkflowRunDetail, error) {
	workflow, err := s.defRepo.GetByID(workflowID)
	if err != nil {
		return nil, err
	}
	if workflow.Status != model.WorkflowStatusEnabled {
		return nil, fmt.Errorf("workflow is not enabled")
	}
	var version *model.WorkflowDefinitionVersion
	if req.VersionID > 0 {
		version, err = s.verRepo.GetByID(req.VersionID)
	} else {
		version, err = s.verRepo.GetLatestPublished(workflowID)
	}
	if err != nil {
		return nil, err
	}
	if version.WorkflowID != workflowID {
		return nil, fmt.Errorf("version does not belong to workflow")
	}
	input := req.Input
	if input == nil {
		input = map[string]interface{}{}
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow input: %w", err)
	}
	now := time.Now()
	run := &model.WorkflowRun{
		WorkflowID:      workflowID,
		VersionID:       version.ID,
		Status:          model.RunStatusRunning,
		InputJSON:       inputJSON,
		RequestUser:     contextx.GetRequestUser(ctx),
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		TraceID:         contextx.GetTraceId(ctx),
		StartedAt:       &now,
		DurationMillis:  0,
	}
	if err := s.runRepo.Create(run); err != nil {
		return nil, err
	}
	if err := s.runner.Execute(ctx, run, version, input); err != nil {
		detail, detailErr := s.GetRunDetail(ctx, run.ID)
		if detailErr != nil {
			return nil, err
		}
		return detail, nil
	}
	return s.GetRunDetail(ctx, run.ID)
}

func (s *Service) GetRunDetail(ctx context.Context, runID int64) (*workflowdto.WorkflowRunDetail, error) {
	run, err := s.runRepo.GetByID(runID)
	if err != nil {
		return nil, err
	}
	steps, err := s.stepRepo.ListByRunID(runID)
	if err != nil {
		return nil, err
	}
	out := make([]*workflowdto.WorkflowStepRunItem, 0, len(steps))
	for _, step := range steps {
		out = append(out, workflowdto.ToWorkflowStepRunItem(step))
	}
	return &workflowdto.WorkflowRunDetail{
		Run:   workflowdto.ToWorkflowRunItem(run),
		Steps: out,
	}, nil
}

func (s *Service) ListRunSteps(ctx context.Context, runID int64) ([]*workflowdto.WorkflowStepRunItem, error) {
	steps, err := s.stepRepo.ListByRunID(runID)
	if err != nil {
		return nil, err
	}
	out := make([]*workflowdto.WorkflowStepRunItem, 0, len(steps))
	for _, step := range steps {
		out = append(out, workflowdto.ToWorkflowStepRunItem(step))
	}
	return out, nil
}

func (s *Service) CancelRun(ctx context.Context, runID int64) error {
	return s.runRepo.Cancel(runID)
}

func (s *Service) validateDefinition(raw json.RawMessage) error {
	parsed, err := definition.Parse(raw)
	if err != nil {
		return err
	}
	return parsed.Validate(definition.ValidateOptions{SupportedNodeTypes: definition.SupportedMVPNodeTypes()})
}

func normalizeRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func isWorkflowStatus(status string) bool {
	switch status {
	case model.WorkflowStatusDraft, model.WorkflowStatusEnabled, model.WorkflowStatusDisabled:
		return true
	default:
		return false
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
