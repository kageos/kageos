package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	workflowdto "github.com/ai-agent-os/ai-agent-os/core/workflow-server/dto"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/executor"
	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeFormExecutor struct{}

func (f *fakeFormExecutor) Type() string {
	return definition.NodeTypeForm
}

func (f *fakeFormExecutor) Validate(ctx context.Context, node definition.Node, def *definition.Definition) error {
	return nil
}

func (f *fakeFormExecutor) Execute(ctx context.Context, input executor.NodeInput, runtime executor.RuntimeContext) (map[string]interface{}, error) {
	switch input.Node.ID {
	case "extract":
		return map[string]interface{}{"text": fmt.Sprintf("%v text", input.Input["file"])}, nil
	case "summary":
		return map[string]interface{}{"summary": fmt.Sprintf("%v summary", input.Input["text"])}, nil
	default:
		return nil, fmt.Errorf("unexpected node %s", input.Node.ID)
	}
}

func TestWorkflowServiceRunSequentialForms(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db, executor.NewRegistry(&fakeFormExecutor{}))
	ctx := context.WithValue(context.Background(), contextx.RequestUserHeader, "alice")
	ctx = context.WithValue(ctx, contextx.TraceIdHeader, "trace-1")

	definitionJSON := json.RawMessage(`{
		"schema_version":"workflow.v1",
		"mode":"graph",
		"nodes":[
			{"id":"start","name":"开始","type":"workflow.start","schema":{"version":1,"type":"form","form":{"request":[{"code":"file","name":"文件","data":{"type":"string"},"validation":"required"}]}}},
			{"id":"extract","name":"提取文本","type":"form.submit","ref":"/system/extract.form","input":{"file":{"$ref":"input.file"}}},
			{"id":"summary","name":"生成摘要","type":"form.submit","ref":"/system/summary.form","input":{"text":{"$ref":"steps.extract.output.text"}}},
			{"id":"output","name":"输出","type":"workflow.output","schema":{"version":1,"type":"form","form":{"response":[{"code":"summary","name":"摘要","data":{"type":"string"}}]}},"input":{"summary":{"$ref":"steps.summary.output.summary"}}}
		],
		"edges":[{"from":"start","to":"extract"},{"from":"extract","to":"summary"},{"from":"summary","to":"output"}]
	}`)

	workflow, err := svc.CreateWorkflow(ctx, workflowdto.CreateWorkflowRequest{
		Name:       "文档摘要",
		Definition: definitionJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	version, err := svc.PublishWorkflow(ctx, workflow.ID, workflowdto.PublishWorkflowRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 1 {
		t.Fatalf("version = %d, want 1", version.Version)
	}

	detail, err := svc.RunWorkflow(ctx, workflow.ID, workflowdto.RunWorkflowRequest{
		Input: map[string]interface{}{"file": "report.pdf"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.Status != model.RunStatusSuccess {
		t.Fatalf("run status = %s, want success", detail.Run.Status)
	}
	if len(detail.Steps) != 4 {
		t.Fatalf("steps len = %d, want 4", len(detail.Steps))
	}
	var output map[string]interface{}
	if err := json.Unmarshal(detail.Run.OutputJSON, &output); err != nil {
		t.Fatal(err)
	}
	if output["summary"] != "report.pdf text summary" {
		t.Fatalf("summary = %#v", output["summary"])
	}
}
