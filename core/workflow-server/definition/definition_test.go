package definition

import (
	"encoding/json"
	"testing"
)

func TestValidateGraphWithStartAndOutput(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"graph",
		"nodes":[
			{"id":"start","name":"开始","type":"workflow.start","schema":{"version":1,"type":"form","form":{"request":[{"code":"file","name":"文件","data":{"type":"string"},"validation":"required"}]}}},
			{"id":"extract","name":"extract","type":"form.submit","ref":"/system/a.form","input":{"file":{"$ref":"input.file"}}},
			{"id":"summary","name":"summary","type":"form.submit","ref":"/system/b.form","input":{"text":{"$ref":"steps.extract.output.text"}}},
			{"id":"output","name":"输出","type":"workflow.output","schema":{"version":1,"type":"form","form":{"response":[{"code":"summary","name":"摘要","data":{"type":"string"}}]}},"input":{"summary":{"$ref":"steps.summary.output.summary"}}}
		],
		"edges":[{"from":"start","to":"extract"},{"from":"extract","to":"summary"},{"from":"summary","to":"output"}]
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
	if len(order) != 4 || order[0].ID != "start" || order[3].ID != "output" {
		t.Fatalf("unexpected order: %#v", order)
	}
}

func TestValidateRejectsMissingOutputMapping(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"graph",
		"nodes":[
			{"id":"start","type":"workflow.start","schema":{"version":1,"type":"form","form":{"request":[]}}},
			{"id":"a","type":"form.submit","ref":"/a.form"},
			{"id":"output","type":"workflow.output","schema":{"version":1,"type":"form","form":{"response":[{"code":"result","name":"结果"}]}},"input":{}}
		],
		"edges":[{"from":"start","to":"a"},{"from":"a","to":"output"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := def.Validate(ValidateOptions{SupportedNodeTypes: SupportedMVPNodeTypes()}); err == nil {
		t.Fatal("expected output mapping validation error")
	}
}

func TestValidateRejectsCycle(t *testing.T) {
	t.Parallel()

	def, err := Parse(json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"graph",
		"nodes":[
			{"id":"start","type":"workflow.start","schema":{"version":1,"type":"form","form":{"request":[]}}},
			{"id":"a","type":"form.submit","ref":"/a.form"},
			{"id":"b","type":"form.submit","ref":"/b.form"},
			{"id":"output","type":"workflow.output","schema":{"version":1,"type":"form","form":{"response":[]}}}
		],
		"edges":[{"from":"start","to":"a"},{"from":"a","to":"b"},{"from":"b","to":"a"},{"from":"b","to":"output"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := def.Validate(ValidateOptions{SupportedNodeTypes: SupportedMVPNodeTypes()}); err == nil {
		t.Fatal("expected cycle validation error")
	}
}
