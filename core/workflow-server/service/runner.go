package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/executor"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/workflowexpr"
)

type Runner struct {
	runRepo  *repository.WorkflowRunRepository
	stepRepo *repository.WorkflowStepRunRepository
	registry *executor.Registry
	now      func() time.Time
}

type RunnerDeps struct {
	RunRepo  *repository.WorkflowRunRepository
	StepRepo *repository.WorkflowStepRunRepository
	Registry *executor.Registry
	Now      func() time.Time
}

func NewRunner(deps RunnerDeps) *Runner {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &Runner{
		runRepo:  deps.RunRepo,
		stepRepo: deps.StepRepo,
		registry: deps.Registry,
		now:      now,
	}
}

func (r *Runner) Execute(ctx context.Context, run *model.WorkflowRun, version *model.WorkflowDefinitionVersion, input map[string]interface{}) error {
	if run == nil {
		return fmt.Errorf("workflow run is nil")
	}
	start := r.runStart(run)
	def, err := definition.Parse(version.DefinitionJSON)
	if err != nil {
		r.finishRunFailed(run, start, err)
		return err
	}
	if err := def.Validate(definition.ValidateOptions{SupportedNodeTypes: r.registry.SupportedTypes()}); err != nil {
		r.finishRunFailed(run, start, err)
		return err
	}
	order, err := def.TopologicalOrder()
	if err != nil {
		r.finishRunFailed(run, start, err)
		return err
	}
	for _, node := range order {
		if definition.IsBuiltinNodeType(node.Type) {
			continue
		}
		exec, err := r.registry.Get(node.Type)
		if err != nil {
			r.finishRunFailed(run, start, err)
			return err
		}
		if err := exec.Validate(ctx, node, def); err != nil {
			r.finishRunFailed(run, start, err)
			return err
		}
	}

	stepsContext := map[string]interface{}{}
	exprCtx := workflowexpr.Context{
		"input": input,
		"steps": stepsContext,
	}
	var workflowOutput map[string]interface{}
	for _, node := range order {
		var nodeInput map[string]interface{}
		var output map[string]interface{}
		var execErr error
		switch node.Type {
		case definition.NodeTypeStart:
			nodeInput = cloneMap(input)
			output = cloneMap(input)
		case definition.NodeTypeOutput:
			nodeInput, execErr = resolveNodeInput(node, exprCtx)
			output = cloneMap(nodeInput)
			workflowOutput = output
		default:
			nodeInput, execErr = resolveNodeInput(node, exprCtx)
		}
		if execErr != nil {
			r.finishRunFailed(run, start, fmt.Errorf("node %s input: %w", node.ID, execErr))
			return execErr
		}
		step, err := r.createStepRun(run, node, nodeInput)
		if err != nil {
			r.finishRunFailed(run, start, err)
			return err
		}
		if !definition.IsBuiltinNodeType(node.Type) {
			output, execErr = r.executeNode(ctx, node, nodeInput, run, version)
			if execErr != nil {
				_ = r.finishStep(step, model.StepStatusFailed, nil, execErr.Error())
				r.finishRunFailed(run, start, fmt.Errorf("node %s failed: %w", node.ID, execErr))
				return execErr
			}
		}
		if err := r.finishStep(step, model.StepStatusSuccess, output, ""); err != nil {
			r.finishRunFailed(run, start, err)
			return err
		}
		stepsContext[node.ID] = map[string]interface{}{
			"input":  nodeInput,
			"output": output,
		}
	}

	if workflowOutput == nil {
		err := fmt.Errorf("workflow output node did not run")
		r.finishRunFailed(run, start, err)
		return err
	}
	outputJSON, err := json.Marshal(workflowOutput)
	if err != nil {
		r.finishRunFailed(run, start, err)
		return err
	}
	finishedAt := r.now()
	return r.runRepo.Finish(run.ID, model.RunStatusSuccess, outputJSON, "", finishedAt, finishedAt.Sub(start).Milliseconds())
}

func (r *Runner) runStart(run *model.WorkflowRun) time.Time {
	if run.StartedAt != nil {
		return *run.StartedAt
	}
	now := r.now()
	return now
}

func (r *Runner) finishRunFailed(run *model.WorkflowRun, startedAt time.Time, runErr error) {
	finishedAt := r.now()
	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
	}
	_ = r.runRepo.Finish(run.ID, model.RunStatusFailed, nil, errMsg, finishedAt, finishedAt.Sub(startedAt).Milliseconds())
}

func (r *Runner) createStepRun(run *model.WorkflowRun, node definition.Node, input map[string]interface{}) (*model.WorkflowStepRun, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	now := r.now()
	step := &model.WorkflowStepRun{
		RunID:     run.ID,
		StepID:    node.ID,
		StepName:  node.Name,
		NodeType:  node.Type,
		NodeRef:   node.Ref,
		Status:    model.StepStatusRunning,
		InputJSON: inputJSON,
		TraceID:   run.TraceID,
		Attempt:   1,
		StartedAt: &now,
	}
	if err := r.stepRepo.Create(step); err != nil {
		return nil, err
	}
	return step, nil
}

func (r *Runner) executeNode(ctx context.Context, node definition.Node, input map[string]interface{}, run *model.WorkflowRun, version *model.WorkflowDefinitionVersion) (map[string]interface{}, error) {
	exec, err := r.registry.Get(node.Type)
	if err != nil {
		return nil, err
	}
	return exec.Execute(ctx, executor.NodeInput{
		Node:  node,
		Input: input,
	}, executor.RuntimeContext{
		Run:     run,
		Version: version.Version,
	})
}

func (r *Runner) finishStep(step *model.WorkflowStepRun, status string, output map[string]interface{}, errMsg string) error {
	var outputJSON []byte
	var err error
	if output != nil {
		outputJSON, err = json.Marshal(output)
		if err != nil {
			return err
		}
	}
	finishedAt := r.now()
	start := finishedAt
	if step.StartedAt != nil {
		start = *step.StartedAt
	}
	return r.stepRepo.Finish(step.ID, status, outputJSON, errMsg, finishedAt, finishedAt.Sub(start).Milliseconds())
}

func resolveNodeInput(node definition.Node, exprCtx workflowexpr.Context) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(node.Input))
	for key, expr := range node.Input {
		value, err := workflowexpr.ResolveRaw(expr, exprCtx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
