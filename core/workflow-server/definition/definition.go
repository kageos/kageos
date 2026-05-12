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
	ModeGraph       = "graph"
	NodeTypeStart   = "workflow.start"
	NodeTypeOutput  = "workflow.output"
	NodeTypeForm    = "form.submit"
)

var nodeIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

type Definition struct {
	SchemaVersion string `json:"schema_version"`
	Mode          string `json:"mode"`
	Nodes         []Node `json:"nodes"`
	Edges         []Edge `json:"edges"`
}

type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"`
}

type Node struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Type      string                     `json:"type"`
	Ref       string                     `json:"ref,omitempty"`
	Schema    json.RawMessage            `json:"schema,omitempty"`
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

type FormSchema struct {
	Version int            `json:"version"`
	Type    string         `json:"type"`
	Form    FormSchemaBody `json:"form"`
}

type FormSchemaBody struct {
	Request  []FormField `json:"request,omitempty"`
	Response []FormField `json:"response,omitempty"`
}

type FormField struct {
	Code       string                 `json:"code"`
	Name       string                 `json:"name,omitempty"`
	FieldName  string                 `json:"field_name,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Widget     map[string]interface{} `json:"widget,omitempty"`
	Validation string                 `json:"validation,omitempty"`
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
		mode = ModeGraph
	}
	if mode != ModeGraph {
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
	if err := d.validateGraphBoundaries(nodeByID); err != nil {
		return err
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
	if len(opts.SupportedNodeTypes) > 0 && !opts.SupportedNodeTypes[node.Type] && !IsBuiltinNodeType(node.Type) {
		return fmt.Errorf("unsupported node type: %s", node.Type)
	}
	switch node.Type {
	case NodeTypeStart:
		if strings.TrimSpace(node.Ref) != "" {
			return fmt.Errorf("ref is not allowed for %s", NodeTypeStart)
		}
		if len(node.Input) > 0 {
			return fmt.Errorf("input is not allowed for %s", NodeTypeStart)
		}
		if _, err := validateFormSchema(node.Schema, "request"); err != nil {
			return err
		}
	case NodeTypeOutput:
		if strings.TrimSpace(node.Ref) != "" {
			return fmt.Errorf("ref is not allowed for %s", NodeTypeOutput)
		}
		fields, err := validateFormSchema(node.Schema, "response")
		if err != nil {
			return err
		}
		for _, field := range fields {
			if _, ok := node.Input[field.Code]; !ok {
				return fmt.Errorf("input.%s is required for output field", field.Code)
			}
		}
	case NodeTypeForm:
		if strings.TrimSpace(node.Ref) == "" {
			return fmt.Errorf("ref is required for %s", NodeTypeForm)
		}
	default:
		if len(opts.SupportedNodeTypes) == 0 {
			return fmt.Errorf("unsupported node type: %s", node.Type)
		}
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

func validateFormSchema(raw json.RawMessage, section string) ([]FormField, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("schema is required")
	}
	var schema FormSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	if schema.Type != "form" {
		return nil, fmt.Errorf("schema.type must be form")
	}
	if schema.Version <= 0 {
		return nil, fmt.Errorf("schema.version is required")
	}
	var fields []FormField
	switch section {
	case "request":
		fields = schema.Form.Request
	case "response":
		fields = schema.Form.Response
	default:
		return nil, fmt.Errorf("unsupported schema section: %s", section)
	}
	seen := make(map[string]bool, len(fields))
	for i := range fields {
		fields[i].Code = strings.TrimSpace(fields[i].Code)
		if fields[i].Code == "" {
			return nil, fmt.Errorf("schema.form.%s[%d].code is required", section, i)
		}
		if strings.Contains(fields[i].Code, ".") {
			return nil, fmt.Errorf("schema.form.%s[%d].code must not contain dot", section, i)
		}
		if seen[fields[i].Code] {
			return nil, fmt.Errorf("duplicate schema.form.%s code: %s", section, fields[i].Code)
		}
		seen[fields[i].Code] = true
	}
	return fields, nil
}

func validateEdges(edges []Edge, nodeByID map[string]Node) error {
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

func (d *Definition) validateGraphBoundaries(nodeByID map[string]Node) error {
	inDegree := make(map[string]int, len(d.Nodes))
	outDegree := make(map[string]int, len(d.Nodes))
	outgoing := make(map[string][]string, len(d.Nodes))
	incoming := make(map[string][]string, len(d.Nodes))
	for _, node := range d.Nodes {
		inDegree[node.ID] = 0
		outDegree[node.ID] = 0
	}
	for _, edge := range d.Edges {
		inDegree[edge.To]++
		outDegree[edge.From]++
		outgoing[edge.From] = append(outgoing[edge.From], edge.To)
		incoming[edge.To] = append(incoming[edge.To], edge.From)
	}

	startID := ""
	outputID := ""
	for id, node := range nodeByID {
		switch node.Type {
		case NodeTypeStart:
			if startID != "" {
				return fmt.Errorf("workflow graph requires exactly one %s node", NodeTypeStart)
			}
			startID = id
		case NodeTypeOutput:
			if outputID != "" {
				return fmt.Errorf("workflow graph requires exactly one %s node", NodeTypeOutput)
			}
			outputID = id
		}
	}
	if startID == "" {
		return fmt.Errorf("workflow graph requires one %s node", NodeTypeStart)
	}
	if outputID == "" {
		return fmt.Errorf("workflow graph requires one %s node", NodeTypeOutput)
	}
	if inDegree[startID] != 0 {
		return fmt.Errorf("%s node must not have incoming edges", NodeTypeStart)
	}
	if outDegree[startID] == 0 {
		return fmt.Errorf("%s node must have outgoing edges", NodeTypeStart)
	}
	if outDegree[outputID] != 0 {
		return fmt.Errorf("%s node must not have outgoing edges", NodeTypeOutput)
	}
	if inDegree[outputID] == 0 {
		return fmt.Errorf("%s node must have incoming edges", NodeTypeOutput)
	}

	reachable := traverseGraph(startID, outgoing)
	canReachOutput := traverseGraph(outputID, incoming)
	for id := range nodeByID {
		if !reachable[id] {
			return fmt.Errorf("node %s is not reachable from %s", id, NodeTypeStart)
		}
		if !canReachOutput[id] {
			return fmt.Errorf("node %s cannot reach %s", id, NodeTypeOutput)
		}
	}
	return nil
}

func traverseGraph(start string, adjacency map[string][]string) map[string]bool {
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[id] {
			if seen[next] {
				continue
			}
			seen[next] = true
			queue = append(queue, next)
		}
	}
	return seen
}

func (d *Definition) StartNode() (*Node, bool) {
	if d == nil {
		return nil, false
	}
	for i := range d.Nodes {
		if d.Nodes[i].Type == NodeTypeStart {
			return &d.Nodes[i], true
		}
	}
	return nil, false
}

func (d *Definition) OutputNode() (*Node, bool) {
	if d == nil {
		return nil, false
	}
	for i := range d.Nodes {
		if d.Nodes[i].Type == NodeTypeOutput {
			return &d.Nodes[i], true
		}
	}
	return nil, false
}

func (d *Definition) StartSchemaJSON() json.RawMessage {
	node, ok := d.StartNode()
	if !ok || len(node.Schema) == 0 {
		return nil
	}
	return cloneRaw(node.Schema)
}

func (d *Definition) OutputSchemaJSON() json.RawMessage {
	node, ok := d.OutputNode()
	if !ok || len(node.Schema) == 0 {
		return nil
	}
	return cloneRaw(node.Schema)
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}

func IsBuiltinNodeType(nodeType string) bool {
	switch strings.TrimSpace(nodeType) {
	case NodeTypeStart, NodeTypeOutput:
		return true
	default:
		return false
	}
}

func SupportedMVPNodeTypes() map[string]bool {
	return map[string]bool{
		NodeTypeStart:  true,
		NodeTypeOutput: true,
		NodeTypeForm:   true,
	}
}
