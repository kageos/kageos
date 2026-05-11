package executor

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
)

type RuntimeContext struct {
	Run     *model.WorkflowRun
	Version int
}

type NodeInput struct {
	Node  definition.Node
	Input map[string]interface{}
}

type NodeExecutor interface {
	Type() string
	Validate(ctx context.Context, node definition.Node, def *definition.Definition) error
	Execute(ctx context.Context, input NodeInput, runtime RuntimeContext) (map[string]interface{}, error)
}

type Registry struct {
	items map[string]NodeExecutor
}

func NewRegistry(executors ...NodeExecutor) *Registry {
	r := &Registry{items: make(map[string]NodeExecutor, len(executors))}
	for _, item := range executors {
		r.Register(item)
	}
	return r
}

func (r *Registry) Register(item NodeExecutor) {
	if r == nil || item == nil {
		return
	}
	t := strings.TrimSpace(item.Type())
	if t == "" {
		return
	}
	r.items[t] = item
}

func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	if r == nil {
		return nil, fmt.Errorf("workflow executor registry is nil")
	}
	item := r.items[strings.TrimSpace(nodeType)]
	if item == nil {
		return nil, fmt.Errorf("unsupported node type: %s", nodeType)
	}
	return item, nil
}

func (r *Registry) SupportedTypes() map[string]bool {
	out := make(map[string]bool, len(r.items))
	for key := range r.items {
		out[key] = true
	}
	return out
}
