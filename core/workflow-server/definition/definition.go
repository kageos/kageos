package definition

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/workflowexpr"
)

const (
	SchemaVersionV1 = "workflow.v1"
	ModeSequence    = "sequence"
	NodeTypeForm    = "form.submit"
)

var nodeIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

type Definition struct {
	SchemaVersion string                     `json:"schema_version"`
	Mode          string                     `json:"mode"`
	Inputs        map[string]InputField      `json:"inputs,omitempty"`
	Triggers      []Trigger                  `json:"triggers,omitempty"`
	Nodes         []Node                     `json:"nodes"`
	Edges         []Edge                     `json:"edges"`
	Outputs       map[string]json.RawMessage `json:"outputs,omitempty"`
}

type InputField struct {
	Type     string `json:"type,omitempty"`
	Required bool   `json:"required,omitempty"`
	Title    string `json:"title,omitempty"`
}

type Trigger struct {
	Type string `json:"type"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}

type Node struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Type      string                     `json:"type"`
	Ref       string                     `json:"ref,omitempty"`
	Input     map[string]json.RawMessage `json:"input,omitempty"`
	DependsOn []string                   `json:"depends_on,omitempty"`
	Retry     *RetryPolicy               `json:"retry,omitempty"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type ValidateOptions struct {
	SupportedNodeTypes map[string]bool
}

func Parse(raw json.RawMessage) (*Definition, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("definition is empty")
	}
	var def Definition
	if err := json.Unmarshal(raw, &def); err != nil {
		return nil, fmt.Errorf("parse definition: %w", err)
	}
	return &def, nil
}

func (d *Definition) Validate(opts ValidateOptions) error {
	if d == nil {
		return fmt.Errorf("definition is nil")
	}
	if strings.TrimSpace(d.SchemaVersion) != SchemaVersionV1 {
		return fmt.Errorf("unsupported schema_version: %s", d.SchemaVersion)
	}
	mode := strings.TrimSpace(d.Mode)
	if mode == "" {
		mode = ModeSequence
	}
	if mode != ModeSequence {
		return fmt.Errorf("unsupported workflow mode: %s", mode)
	}
	if len(d.Nodes) == 0 {
		return fmt.Errorf("nodes is required")
	}
	nodeByID := make(map[string]Node, len(d.Nodes))
	for i, node := range d.Nodes {
		if err := validateNode(node, opts); err != nil {
			return fmt.Errorf("nodes[%d]: %w", i, err)
		}
		if _, exists := nodeByID[node.ID]; exists {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeByID[node.ID] = node
	}
	if err := validateEdges(d.Edges, nodeByID); err != nil {
		return err
	}
	if _, err := d.TopologicalOrder(); err != nil {
		return err
	}
	if mode == ModeSequence {
		if err := d.validateSequence(nodeByID); err != nil {
			return err
		}
	}
	for name, expr := range d.Outputs {
		if err := workflowexpr.ValidateRaw(expr, workflowexpr.MVPOptions()); err != nil {
			return fmt.Errorf("outputs.%s: %w", name, err)
		}
	}
	return nil
}

func validateNode(node Node, opts ValidateOptions) error {
	node.ID = strings.TrimSpace(node.ID)
	if node.ID == "" {
		return fmt.Errorf("id is required")
	}
	if !nodeIDPattern.MatchString(node.ID) {
		return fmt.Errorf("id must match %s", nodeIDPattern.String())
	}
	node.Type = strings.TrimSpace(node.Type)
	if node.Type == "" {
		return fmt.Errorf("type is required")
	}
	if len(opts.SupportedNodeTypes) > 0 && !opts.SupportedNodeTypes[node.Type] {
		return fmt.Errorf("unsupported node type: %s", node.Type)
	}
	if node.Type == NodeTypeForm && strings.TrimSpace(node.Ref) == "" {
		return fmt.Errorf("ref is required for %s", NodeTypeForm)
	}
	for field, expr := range node.Input {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("input field name is empty")
		}
		if err := workflowexpr.ValidateRaw(expr, workflowexpr.MVPOptions()); err != nil {
			return fmt.Errorf("input.%s: %w", field, err)
		}
	}
	return nil
}

func validateEdges(edges []Edge, nodeByID map[string]Node) error {
	if len(nodeByID) == 1 && len(edges) == 0 {
		return nil
	}
	if len(edges) == 0 {
		return fmt.Errorf("edges is required")
	}
	for i, edge := range edges {
		if strings.TrimSpace(edge.From) == "" || strings.TrimSpace(edge.To) == "" {
			return fmt.Errorf("edges[%d]: from and to are required", i)
		}
		if edge.From == edge.To {
			return fmt.Errorf("edges[%d]: self edge is not allowed", i)
		}
		if _, ok := nodeByID[edge.From]; !ok {
			return fmt.Errorf("edges[%d]: unknown from node %s", i, edge.From)
		}
		if _, ok := nodeByID[edge.To]; !ok {
			return fmt.Errorf("edges[%d]: unknown to node %s", i, edge.To)
		}
	}
	return nil
}

func (d *Definition) TopologicalOrder() ([]Node, error) {
	nodeByID := make(map[string]Node, len(d.Nodes))
	inDegree := make(map[string]int, len(d.Nodes))
	out := make(map[string][]string, len(d.Nodes))
	for _, node := range d.Nodes {
		nodeByID[node.ID] = node
		inDegree[node.ID] = 0
	}
	for _, edge := range d.Edges {
		if _, ok := nodeByID[edge.From]; !ok {
			return nil, fmt.Errorf("unknown from node %s", edge.From)
		}
		if _, ok := nodeByID[edge.To]; !ok {
			return nil, fmt.Errorf("unknown to node %s", edge.To)
		}
		out[edge.From] = append(out[edge.From], edge.To)
		inDegree[edge.To]++
	}

	queue := make([]string, 0, len(d.Nodes))
	for _, node := range d.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}
	order := make([]Node, 0, len(d.Nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, nodeByID[id])
		for _, next := range out[id] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	if len(order) != len(d.Nodes) {
		return nil, fmt.Errorf("workflow graph contains a cycle")
	}
	return order, nil
}

func (d *Definition) validateSequence(nodeByID map[string]Node) error {
	if len(d.Nodes) == 1 {
		return nil
	}
	if len(d.Edges) != len(d.Nodes)-1 {
		return fmt.Errorf("sequence workflow requires edges count = nodes - 1")
	}
	inDegree := make(map[string]int, len(d.Nodes))
	outDegree := make(map[string]int, len(d.Nodes))
	for _, node := range d.Nodes {
		inDegree[node.ID] = 0
		outDegree[node.ID] = 0
	}
	for _, edge := range d.Edges {
		inDegree[edge.To]++
		outDegree[edge.From]++
	}
	starts := 0
	ends := 0
	for id := range nodeByID {
		if inDegree[id] > 1 || outDegree[id] > 1 {
			return fmt.Errorf("sequence workflow node %s must have at most one input and one output", id)
		}
		if inDegree[id] == 0 {
			starts++
		}
		if outDegree[id] == 0 {
			ends++
		}
	}
	if starts != 1 || ends != 1 {
		return fmt.Errorf("sequence workflow requires exactly one start node and one end node")
	}
	return nil
}

func SupportedMVPNodeTypes() map[string]bool {
	return map[string]bool{
		NodeTypeForm: true,
	}
}
