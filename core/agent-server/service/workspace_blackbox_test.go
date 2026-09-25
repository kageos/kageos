package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestBuildResultHidesRuntimeDetails(t *testing.T) {
	result := buildWorkspaceSuccessResult("/system/test11", &dto.UpdateAppResp{
		NewVersion: "v2", GitCommitHash: "private-commit",
		Warnings: []string{"open /srv/namespace/system/test11/code/cmd/app: denied"},
		BuildTrace: &dto.BuildTrace{StoragePath: "/srv/trace.json", Spans: []dto.BuildTraceSpan{
			{Name: "runtime.builder_build_app", Attributes: map[string]string{"source_dir": "/srv/namespace/system/test11/code/cmd/app"}},
		}},
	})
	out := toolResultWithStructuredData(result, false).Content
	for _, forbidden := range []string{"code/cmd", "namespace", "/srv", "build_trace", "private-commit", "source_dir"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("model result leaked %q: %s", forbidden, out)
		}
	}
	if !strings.Contains(out, "v2") || result.NextRole != WorkspaceRoleQAEngineer {
		t.Fatalf("lost business result: %s", out)
	}
	schema := formatStructuredToolData((&BuildWorkspaceTool{}).Definition().OutputSchema)
	for _, forbidden := range []string{"build_trace", "git_commit_hash"} {
		if strings.Contains(schema, forbidden) {
			t.Fatalf("schema exposes %s", forbidden)
		}
	}
}

func TestBuildFailureSeparatesSourceAndPlatformErrors(t *testing.T) {
	for _, raw := range []string{
		"open /srv/namespace/system/test11/code/cmd/app: denied",
		"go.mod not found for source directory /srv/private",
		"code/cmd/app/main.go:3:2: broken platform entrypoint",
		"/srv/namespace/other/app/code/api/order.go:8:2: foreign app",
	} {
		result, content := workspaceBuildPublicFailure("/system/test11", raw)
		if result.AutoContinue || result.NextRole != "" || !strings.Contains(content, "platform_build_failed") {
			t.Fatalf("platform error must not trigger source repair: %#v %s", result, content)
		}
		if strings.Contains(content, raw) {
			t.Fatalf("leaked raw error: %s", content)
		}
	}
	raw := "build failed:\n/srv/namespace/system/test11/code/api/order/list.go:18:3: undefined: req.Status\nopen /srv/private: denied"
	result, content := workspaceBuildPublicFailure("/system/test11", raw)
	if result.NextRole != WorkspaceRoleBuildEngineer || !strings.Contains(content, "/system/test11/order/list.go:18:3") || !strings.Contains(content, "req.Status") {
		t.Fatalf("business diagnostic lost: %#v %s", result, content)
	}
	for _, forbidden := range []string{"/srv", "namespace/", "code/api", "private"} {
		if strings.Contains(content+formatStructuredToolData(result), forbidden) {
			t.Fatalf("leaked %s", forbidden)
		}
	}
}

func TestSearchFailureIsNotAnEmptySuccessfulSearch(t *testing.T) {
	for _, kind := range []string{"function", "directory"} {
		t.Run(kind, func(t *testing.T) {
			backend := searchBackend{
				functions: func(context.Context, *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
					return nil, errors.New("nats /srv/private unavailable")
				},
				resources: func(context.Context, *dto.SearchResourcesReq) (*dto.SearchResourcesResp, error) {
					return nil, errors.New("sql /srv/private failed")
				},
			}
			result := runSearchToolWithBackend(context.Background(), nil, searchArgs{FullCodePath: "/system/test11", ResourceType: kind}, backend)
			if !result.IsError || !strings.Contains(result.Content, "resource_query_failed") {
				t.Fatalf("failure hidden: %#v", result)
			}
			for _, forbidden := range []string{"未匹配到", "/srv", "nats", "sql"} {
				if strings.Contains(result.Content, forbidden) {
					t.Fatalf("unexpected %s: %s", forbidden, result.Content)
				}
			}
		})
	}
	backend := searchBackend{functions: func(context.Context, *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
		return &dto.SearchFunctionsResp{}, nil
	}}
	result := runSearchToolWithBackend(context.Background(), nil, searchArgs{ResourceType: "function", FullCodePath: "/system/test11"}, backend)
	if result.IsError || !strings.Contains(result.Content, "未匹配到") {
		t.Fatalf("real empty result changed: %#v", result)
	}
}

func TestBusinessToolDescriptionsUsePublicContracts(t *testing.T) {
	for _, tool := range []Tool{&RunTableSearchTool{}, &RunTableCreateTool{}, &RunTableUpdateTool{}, &RunTableDeleteTool{}, &RunFormSubmitTool{}, &RunChartQueryTool{}, &RunOnSelectFuzzyTool{}} {
		def := tool.Definition()
		for _, forbidden := range []string{"read_file", "init()", "OnTable", "app-server", "json 标签", "handler"} {
			if strings.Contains(def.Description, forbidden) {
				t.Fatalf("%s exposes %s", def.Name, forbidden)
			}
		}
	}
}

func TestBuildFailureKeepsSDKDiagnosticsWithoutStartupDetails(t *testing.T) {
	raw := "container /srv/private startup failed: router /orders/list.table schema decode failed: field Status invalid widget"
	result, content := workspaceBuildPublicFailure("/system/test11", raw)
	if result.NextRole != WorkspaceRoleBuildEngineer || !strings.Contains(content, "field Status") || !strings.Contains(content, "/orders/list.table") {
		t.Fatalf("lost SDK diagnostic: %#v %s", result, content)
	}
	if strings.Contains(content, "container") || strings.Contains(content, "/srv") {
		t.Fatalf("leaked runtime wrapper: %s", content)
	}
}

func TestSearchPartialFailureDoesNotReportCompleteResults(t *testing.T) {
	backend := searchBackend{
		functions: func(context.Context, *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
			return &dto.SearchFunctionsResp{Functions: []*dto.FunctionSearchResult{{Name: "Orders", FullCodePath: "/system/test11/orders.table"}}}, nil
		},
		resources: func(context.Context, *dto.SearchResourcesReq) (*dto.SearchResourcesResp, error) {
			return nil, errors.New("backend down")
		},
	}
	result := runSearchToolWithBackend(context.Background(), nil, searchArgs{ResourceType: "all", FullCodePath: "/system/test11"}, backend)
	if !result.IsError || !strings.Contains(result.Content, "resource_query_failed") {
		t.Fatalf("partial failure hidden: %#v", result)
	}
}
