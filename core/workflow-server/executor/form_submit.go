package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

type FormSubmitClient interface {
	Submit(ctx context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error)
}

type APICallFormSubmitClient struct{}

func (c *APICallFormSubmitClient) Submit(ctx context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error) {
	return apicall.FormSubmit(ctx, fullCodePath, body)
}

type FormSubmitExecutor struct {
	client FormSubmitClient
}

func NewFormSubmitExecutor(client FormSubmitClient) *FormSubmitExecutor {
	if client == nil {
		client = &APICallFormSubmitClient{}
	}
	return &FormSubmitExecutor{client: client}
}

func (e *FormSubmitExecutor) Type() string {
	return definition.NodeTypeForm
}

func (e *FormSubmitExecutor) Validate(ctx context.Context, node definition.Node, def *definition.Definition) error {
	if strings.TrimSpace(node.Ref) == "" {
		return fmt.Errorf("form.submit node %s requires ref", node.ID)
	}
	return nil
}

func (e *FormSubmitExecutor) Execute(ctx context.Context, input NodeInput, runtime RuntimeContext) (map[string]interface{}, error) {
	ref := strings.TrimSpace(input.Node.Ref)
	if ref == "" {
		return nil, fmt.Errorf("form.submit node %s requires ref", input.Node.ID)
	}
	callCtx := ctx
	if contextx.GetClientSource(callCtx) == "" {
		callCtx = contextx.WithClientSource(callCtx, "workflow")
	}
	output, err := e.client.Submit(callCtx, ref, input.Input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		output = map[string]interface{}{}
	}
	return output, nil
}
