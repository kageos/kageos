package definition

import (
	"encoding/json"
	"testing"
)

func TestValidateSequence(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"sequence",
		"nodes":[
			{"id":"extract","name":"extract","type":"form.submit","ref":"/system/a.form","input":{"file":{"$ref":"input.file"}}},
			{"id":"summary","name":"summary","type":"form.submit","ref":"/system/b.form","input":{"text":{"$ref":"steps.extract.output.text"}}}
		],
		"edges":[{"from":"extract","to":"summary"}],
		"outputs":{"summary":{"$ref":"steps.summary.output.summary"}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := def.Validate(ValidateOptions{SupportedNodeTypes: SupportedMVPNodeTypes()}); err != nil {
		t.Fatal(err)
	}
	order, err := def.TopologicalOrder()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0].ID != "extract" || order[1].ID != "summary" {
		t.Fatalf("unexpected order: %#v", order)
	}
}

func TestValidateRejectsBranchInSequence(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"sequence",
		"nodes":[
			{"id":"a","type":"form.submit","ref":"/a.form"},
			{"id":"b","type":"form.submit","ref":"/b.form"},
			{"id":"c","type":"form.submit","ref":"/c.form"}
		],
		"edges":[{"from":"a","to":"b"},{"from":"a","to":"c"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := def.Validate(ValidateOptions{SupportedNodeTypes: SupportedMVPNodeTypes()}); err == nil {
		t.Fatal("expected sequence branch validation error")
	}
}

func TestValidateRejectsCycle(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"sequence",
		"nodes":[
			{"id":"a","type":"form.submit","ref":"/a.form"},
			{"id":"b","type":"form.submit","ref":"/b.form"}
		],
		"edges":[{"from":"a","to":"b"},{"from":"b","to":"a"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := def.Validate(ValidateOptions{SupportedNodeTypes: SupportedMVPNodeTypes()}); err == nil {
		t.Fatal("expected cycle validation error")
	}
}
