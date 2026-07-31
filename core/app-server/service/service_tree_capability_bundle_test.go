package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestDownloadCapabilityBundleUsesInstallKeyHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Install-Key"); got != "install-key-123" {
			t.Fatalf("X-Install-Key = %q, want install-key-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"schema_version":"capability.bundle.v1",
			"name":"remote bundle",
			"packages":[],
			"files":[]
		}`))
	}))
	defer server.Close()

	bundle, err := downloadCapabilityBundle(context.Background(), server.URL+"/bundle", "install-key-123")
	if err != nil {
		t.Fatalf("downloadCapabilityBundle() error = %v", err)
	}
	if bundle.Name != "remote bundle" || bundle.SchemaVersion != dto.CapabilityBundleSchemaVersion {
		t.Fatalf("unexpected bundle: %#v", bundle)
	}
}

func TestCapabilityBundleMetadataFromTreePreservesDirectoryCode(t *testing.T) {
	t.Parallel()

	metadata := capabilityBundleMetadataFromTree(&model.ServiceTree{
		Code:        "gold_watch",
		Name:        "黄金盯盘助手",
		Description: "监控黄金价格并生成提醒。",
		Tags:        "黄金价格，行情监控, 价格提醒；黄金价格",
		Type:        model.ServiceTreeTypePackage,
		Version:     "v12",
	})

	if metadata == nil || metadata.Directory == nil {
		t.Fatal("expected directory metadata")
	}
	directory := metadata.Directory
	if directory.Code != "gold_watch" {
		t.Fatalf("directory code = %q, want gold_watch", directory.Code)
	}
	if directory.ReleaseVersion != "" || directory.SourceRevision != "v12" {
		t.Fatalf("unexpected versions: %#v", directory)
	}
	wantTags := []string{"黄金价格", "行情监控", "价格提醒"}
	if !reflect.DeepEqual(directory.Tags, wantTags) {
		t.Fatalf("tags = %#v, want %#v", directory.Tags, wantTags)
	}
}

func TestCapabilityBundleMetadataUsesSemverReleaseVersion(t *testing.T) {
	t.Parallel()

	metadata := capabilityBundleMetadataFromTree(&model.ServiceTree{
		Code:    "gold_watch",
		Name:    "黄金盯盘助手",
		Type:    model.ServiceTreeTypePackage,
		Version: "0.2.0",
	})
	if got := metadata.Directory.ReleaseVersion; got != "0.2.0" {
		t.Fatalf("release version = %q, want 0.2.0", got)
	}
	if err := validateCapabilityBundleMetadata(metadata); err != nil {
		t.Fatalf("validateCapabilityBundleMetadata() error = %v", err)
	}
}

func TestValidateCapabilityBundleMetadataRejectsMarketplaceSlug(t *testing.T) {
	t.Parallel()

	err := validateCapabilityBundleMetadata(&dto.CapabilityBundleMetadata{
		Directory: &dto.CapabilityBundleDirectoryMetadata{
			Code: "gold-watch",
		},
	})
	if err == nil {
		t.Fatal("expected invalid directory code")
	}
}

func TestValidateCapabilityBundleMetadataRejectsNestedDirectoryCode(t *testing.T) {
	t.Parallel()

	err := validateCapabilityBundleMetadata(&dto.CapabilityBundleMetadata{
		Directory: &dto.CapabilityBundleDirectoryMetadata{
			Code: "tools/gold_watch",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "单个 package 标识") {
		t.Fatalf("expected single package identifier rejection, got %v", err)
	}
}

func TestDownloadCapabilityBundleExtractsInstallKeyFromURL(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if got := r.Header.Get("X-Install-Key"); got != "url-key-123" {
			t.Fatalf("X-Install-Key = %q, want url-key-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"schema_version":"capability.bundle.v1",
			"name":"url key bundle",
			"packages":[],
			"files":[]
		}`))
	}))
	defer server.Close()

	bundle, err := downloadCapabilityBundle(context.Background(), server.URL+"/api/v1/products/demo/bundle/url-key-123", "")
	if err != nil {
		t.Fatalf("downloadCapabilityBundle() error = %v", err)
	}
	if gotPath != "/api/v1/products/demo/bundle" {
		t.Fatalf("request path = %q, want /api/v1/products/demo/bundle", gotPath)
	}
	if bundle.Name != "url key bundle" {
		t.Fatalf("bundle name = %q, want url key bundle", bundle.Name)
	}
}

func TestDownloadCapabilityBundleRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := downloadCapabilityBundle(context.Background(), "file:///tmp/bundle.json", "")
	if err == nil || !strings.Contains(err.Error(), "http/https") {
		t.Fatalf("expected scheme rejection, got %v", err)
	}
}

func TestValidateCapabilityBundleRejectsWorkspaceBoundPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		bundle *dto.CapabilityBundle
		want   string
	}{
		{
			name: "absolute file path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Files: []*dto.CapabilityBundleFile{
					{Path: "/system/openapi/message/send.go", Content: "package message"},
				},
			},
			want: "目录内直接文件名",
		},
		{
			name: "namespace package path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "namespace/system/openapi/message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "namespace/system/openapi/message", Path: "send.go", Content: "package message"},
				},
			},
			want: "namespace",
		},
		{
			name: "code api package path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "system/openapi/code/api/message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "system/openapi/code/api/message", Path: "send.go", Content: "package message"},
				},
			},
			want: "code/api",
		},
		{
			name: "parent traversal",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "../message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "../message", Path: "send.go", Content: "package message"},
				},
			},
			want: "非法路径片段",
		},
		{
			name: "generated init",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Files: []*dto.CapabilityBundleFile{
					{Path: "init_.go", Content: "package message"},
				},
			},
			want: "init_.go",
		},
		{
			name: "internal manifest seed",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Files: []*dto.CapabilityBundleFile{
					{Path: "kageos_manifest.go", Content: "package message"},
				},
			},
			want: "本地目录种子声明",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCapabilityBundle(tc.bundle)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error:\nwant contains %q\ngot %v", tc.want, err)
			}
		})
	}
}

func TestAppendCapabilityBundleAgentTasksExportsRelativeTasks(t *testing.T) {
	payload, err := json.Marshal(map[string]interface{}{
		"full_code_path":       "/system/demo/customer_follow",
		"message":              "每天整理客户跟进清单。",
		"display_content":      "每天整理客户跟进清单。",
		"mode_code":            "dev",
		"max_duration_seconds": 900,
	})
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeAppScheduleClient{
		listResp: &scheduledsdk.ListTasksResponse{List: []*scheduledsdk.Task{
			{
				ID:              7,
				Title:           "客户跟进每日简报",
				Description:     "整理到期客户、高意向商机和建议动作。",
				ExecutorKey:     ScheduledAgentSessionExecutorKey,
				ExecutorPayload: payload,
				Metadata: map[string]string{
					"managed_by":    "app_manifest",
					"schedule_code": "daily_follow_brief",
					"mode_code":     "dev",
				},
				Status:        scheduledsdk.TaskStatusPending,
				Schedule:      scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "5 9 * * *", Timezone: "Asia/Shanghai"},
				ResourceScope: "workspace_directory",
				ResourceKey:   "/system/demo/customer_follow",
			},
		}},
	}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	bundle := &dto.CapabilityBundle{}
	svc := &serviceTreeCapabilityBundleService{}
	err = svc.appendCapabilityBundleAgentTasks(context.Background(), bundle,
		&model.ServiceTree{Code: "customer_follow", FullCodePath: "/system/demo/customer_follow"},
		&model.ServiceTree{Code: "customer_follow", FullCodePath: "/system/demo/customer_follow"},
		true,
		map[string]struct{}{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.AgentTasks) != 1 {
		t.Fatalf("agent_tasks len = %d, want 1", len(bundle.AgentTasks))
	}
	task := bundle.AgentTasks[0]
	if task.RelativePath != "customer_follow" || task.Code != "daily_follow_brief" || !task.Enabled {
		t.Fatalf("unexpected exported task identity: %#v", task)
	}
	if task.Message != "每天整理客户跟进清单。" || task.MaxDurationSeconds != 900 || task.Schedule.CronExpr != "5 9 * * *" {
		t.Fatalf("unexpected exported task content: %#v", task)
	}
	if task.ManagedBy != scheduledTaskSourceManifest || task.Origin != scheduledTaskOriginManifest {
		t.Fatalf("unexpected exported task ownership: %#v", task)
	}
}

func TestCapabilityBundleAgentTaskSerializesPausedState(t *testing.T) {
	raw, err := json.Marshal(&dto.CapabilityBundleAgentTask{
		RelativePath: "customer_follow",
		Code:         "daily_follow_brief",
		Message:      "每天整理客户跟进清单。",
		Enabled:      false,
		Schedule: scheduledsdk.Schedule{
			Type:     scheduledsdk.ScheduleCron,
			CronExpr: "5 9 * * *",
			Timezone: "Asia/Shanghai",
		},
		ManagedBy: scheduledTaskSourceManifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"enabled":false`) {
		t.Fatalf("paused task must serialize an explicit enabled=false, got %s", raw)
	}
	if !strings.Contains(string(raw), `"managed_by":"app_manifest"`) {
		t.Fatalf("manifest ownership must be portable, got %s", raw)
	}
}

func TestCapabilityBundleScheduledFunctionPreservesPortableDefinition(t *testing.T) {
	payload := mustRawJSON(map[string]interface{}{
		"full_code_path": "/system/demo/customer_follow/daily_report.form",
		"template_type":  "form",
		"action":         "execute",
		"method":         "POST",
		"payload": map[string]interface{}{
			"scope": "last_30_days",
		},
	})
	item, err := capabilityBundleScheduledFunctionFromTask(
		&model.ServiceTree{Code: "customer_follow", FullCodePath: "/system/demo/customer_follow", Type: model.ServiceTreeTypePackage},
		&scheduledsdk.Task{
			ID:              8,
			Title:           "客户跟进月报",
			ExecutorKey:     ScheduledFunctionExecutorKey,
			ExecutorPayload: payload,
			Metadata: map[string]string{
				"managed_by":      "app_manifest",
				"schedule_code":   "monthly_report",
				"default_enabled": "true",
			},
			Status:      scheduledsdk.TaskStatusPaused,
			Schedule:    scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "0 9 1 * *"},
			ResourceKey: "/system/demo/customer_follow/daily_report.form",
		},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Fatal("expected scheduled function")
	}
	if item.RelativePath != "customer_follow/daily_report.form" || item.Code != "monthly_report" {
		t.Fatalf("unexpected identity: %#v", item)
	}
	if !item.DefaultEnabled || item.ManagedBy != "app_manifest" || item.Origin != scheduledTaskOriginManifest {
		t.Fatalf("unexpected definition metadata: %#v", item)
	}
	if got := string(item.Body); !strings.Contains(got, `"scope":"last_30_days"`) {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestBuildCapabilityBundleAgentTaskRequestRebasesTargetPath(t *testing.T) {
	req, err := buildCapabilityBundleAgentTaskRequest(context.Background(), "/alice/app/customers", &dto.CapabilityBundleAgentTask{
		RelativePath:       "customers",
		Code:               "daily_follow_brief",
		Title:              "客户跟进每日简报",
		Description:        "整理到期客户、高意向商机和建议动作。",
		Message:            "每天整理客户跟进清单。",
		Schedule:           scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "5 9 * * *", Timezone: "Asia/Shanghai"},
		MaxDurationSeconds: 900,
	})
	if err != nil {
		t.Fatal(err)
	}
	if req.ExecutorKey != ScheduledAgentSessionExecutorKey || req.Status != scheduledsdk.TaskStatusPaused {
		t.Fatalf("unexpected request identity: %#v", req)
	}
	if req.ResourceKey != "/alice/app/customers" || req.SourceRef != "/alice/app/customers" {
		t.Fatalf("unexpected resource path: %#v", req)
	}
	if req.Metadata["managed_by"] != "capability_bundle" || req.Metadata["bundle_task_code"] != "daily_follow_brief" {
		t.Fatalf("unexpected metadata: %#v", req.Metadata)
	}
	if req.IdempotencyKey != packageAgentTaskIdempotencyKey("/alice/app/customers", "daily_follow_brief") {
		t.Fatalf("bundle and manifest tasks must share one logical idempotency key, got %q", req.IdempotencyKey)
	}
	var payload scheduledAgentSessionPayload
	if err := json.Unmarshal(req.ExecutorPayload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.FullCodePath != "/alice/app/customers" || payload.Message != "每天整理客户跟进清单。" || payload.MaxDurationSeconds != 900 {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestInstallCapabilityBundleAgentTasksOverwriteRefreshesExistingTask(t *testing.T) {
	fake := &fakeAppScheduleClient{}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	svc := &serviceTreeCapabilityBundleService{}
	refs, err := svc.installCapabilityBundleAgentTasks(context.Background(), "/alice/app", []*dto.CapabilityBundleAgentTask{
		{
			RelativePath: "customers",
			Code:         "daily_brief",
			Title:        "Daily brief",
			Message:      "Summarize customer updates",
			Enabled:      true,
			Schedule:     scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "0 9 * * *"},
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0] != "alice/app/customers:daily_brief" {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	if len(fake.created) != 1 {
		t.Fatalf("first import must create exactly one task, created=%d", len(fake.created))
	}
	if len(fake.updated) != 1 {
		t.Fatalf("updated tasks = %d, want 1", len(fake.updated))
	}
	if len(fake.resumed) != 0 || len(fake.paused) != 0 {
		t.Fatalf("overwrite must preserve runtime status, resumed=%#v paused=%#v", fake.resumed, fake.paused)
	}
}

func TestInstallCapabilityBundleAgentTasksSkipsManifestDefinition(t *testing.T) {
	fake := &fakeAppScheduleClient{}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	svc := &serviceTreeCapabilityBundleService{}
	refs, err := svc.installCapabilityBundleAgentTasks(context.Background(), "/system/democase", []*dto.CapabilityBundleAgentTask{
		{
			RelativePath: "gold_watch",
			Code:         "gold_watch_daily_report",
			Title:        "黄金盯盘日报",
			Message:      "生成黄金盯盘日报。",
			Schedule:     scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "0 8 * * *"},
			ManagedBy:    scheduledTaskSourceManifest,
		},
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 || len(fake.created) != 0 || len(fake.updated) != 0 {
		t.Fatalf("manifest definition must be left to source reconciliation, refs=%#v created=%d updated=%d", refs, len(fake.created), len(fake.updated))
	}
}

func TestInstallCapabilityBundleAgentTasksLegacyBundleReusesManifestTask(t *testing.T) {
	fake := &fakeAppScheduleClient{
		listResp: &scheduledsdk.ListTasksResponse{List: []*scheduledsdk.Task{
			{
				ID:          25,
				ExecutorKey: ScheduledAgentSessionExecutorKey,
				ResourceKey: "/system/democase/gold_watch",
				Metadata: map[string]string{
					"managed_by":    scheduledTaskSourceManifest,
					"schedule_code": "gold_watch_daily_report",
				},
			},
		}},
	}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	svc := &serviceTreeCapabilityBundleService{}
	refs, err := svc.installCapabilityBundleAgentTasks(context.Background(), "/system/democase", []*dto.CapabilityBundleAgentTask{
		{
			RelativePath: "gold_watch",
			Code:         "gold_watch_daily_report",
			Title:        "黄金盯盘日报",
			Message:      "生成黄金盯盘日报。",
			Schedule:     scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "0 8 * * *"},
			// 旧 capability.bundle.v1 没有 managed_by。
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 0 || len(fake.created) != 0 || len(fake.updated) != 0 {
		t.Fatalf("legacy bundle must reuse the manifest task, refs=%#v created=%d updated=%d", refs, len(fake.created), len(fake.updated))
	}
}

func TestInstallCapabilityBundleAgentTasksRepeatedImportReusesBundleTask(t *testing.T) {
	fake := &fakeAppScheduleClient{
		listResp: &scheduledsdk.ListTasksResponse{List: []*scheduledsdk.Task{
			{
				ID:          26,
				ExecutorKey: ScheduledAgentSessionExecutorKey,
				ResourceKey: "/alice/app/customers",
				Metadata: map[string]string{
					"managed_by":       scheduledTaskSourceBundle,
					"bundle_task_code": "daily_brief",
					"schedule_code":    "daily_brief",
				},
			},
		}},
	}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	svc := &serviceTreeCapabilityBundleService{}
	refs, err := svc.installCapabilityBundleAgentTasks(context.Background(), "/alice/app", []*dto.CapabilityBundleAgentTask{
		{
			RelativePath: "customers",
			Code:         "daily_brief",
			Title:        "Daily brief",
			Message:      "Summarize customer updates",
			Schedule:     scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "0 9 * * *"},
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(refs) != 1 || refs[0] != "alice/app/customers:daily_brief" {
		t.Fatalf("unexpected refs: %#v", refs)
	}
	if len(fake.created) != 0 || len(fake.updated) != 1 {
		t.Fatalf("repeated import must update one existing task, created=%d updated=%d", len(fake.created), len(fake.updated))
	}
}

func TestCleanupReplacedAgentTasksDeletesOnlyTasksMissingFromSource(t *testing.T) {
	fake := &fakeAppScheduleClient{
		listResp: &scheduledsdk.ListTasksResponse{List: []*scheduledsdk.Task{
			{
				ID:          10,
				ExecutorKey: ScheduledAgentSessionExecutorKey,
				ResourceKey: "/bob/app/tools",
				Metadata:    map[string]string{"bundle_task_code": "keep", "managed_by": "capability_bundle"},
			},
			{
				ID:          11,
				ExecutorKey: ScheduledAgentSessionExecutorKey,
				ResourceKey: "/bob/app/tools",
				Metadata:    map[string]string{"bundle_task_code": "remove", "managed_by": "capability_bundle"},
			},
			{
				ID:          12,
				ExecutorKey: ScheduledAgentSessionExecutorKey,
				ResourceKey: "/bob/app/tools",
				Metadata:    map[string]string{"origin": "user", "bundle_task_code": "custom_copy"},
			},
		}},
	}
	old := newAppScheduleClient
	newAppScheduleClient = func() appScheduleClient { return fake }
	defer func() { newAppScheduleClient = old }()

	err := cleanupReplacedAgentTasks(context.Background(), &copyDirectoryPlan{
		targetRootPath:       "/bob/app/tools",
		runtimeAssetBasePath: "/bob/app",
		agentTasks: []*dto.CapabilityBundleAgentTask{
			{RelativePath: "tools", Code: "keep"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(fake.deleted, []int64{11}) {
		t.Fatalf("deleted task ids = %#v, want [11]", fake.deleted)
	}
}

func TestBuildCapabilityBundleInstallPlanMountsRelativePackages(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "开放能力",
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "invoice", Name: "发票能力"},
			{Path: "report", Name: "报表能力"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "invoice", Path: "create.go", Content: "package invoice"},
			{PackagePath: "report", Path: "query.go", Content: "package report"},
		},
	}
	if err := validateCapabilityBundle(bundle); err != nil {
		t.Fatalf("expected valid bundle: %v", err)
	}

	plan, err := buildCapabilityBundleInstallPlan("/system/tools/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}

	gotDirs := make([]string, 0, len(plan.directoryItems))
	for _, item := range plan.directoryItems {
		gotDirs = append(gotDirs, item.FullCodePath+"::"+item.Name)
	}
	wantDirs := []string{
		"/system/tools/openapi/invoice::发票能力",
		"/system/tools/openapi/report::报表能力",
	}
	if !reflect.DeepEqual(gotDirs, wantDirs) {
		t.Fatalf("unexpected directories:\nwant=%#v\ngot=%#v", wantDirs, gotDirs)
	}

	gotFiles := make([]string, 0, len(plan.fileItems))
	for _, item := range plan.fileItems {
		gotFiles = append(gotFiles, item.FullCodePath+"::"+item.FileName+"."+item.FileType)
	}
	wantFiles := []string{
		"/system/tools/openapi/invoice::create.go",
		"/system/tools/openapi/report::query.go",
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("unexpected files:\nwant=%#v\ngot=%#v", wantFiles, gotFiles)
	}
}

func TestBuildCapabilityBundleInstallPlanIncludesDocs(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "CRM",
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm", Name: "CRM"},
			{RelativePath: "crm/readme.docs", ParentPath: "crm", Type: model.ServiceTreeTypeDocs, Code: "readme.docs", Name: "使用说明", Tags: []string{"docs", "crm"}},
		},
		Docs: []*dto.CapabilityBundleDoc{
			{RelativePath: "crm/readme.docs", Name: "使用说明", Content: "# 使用说明\n", Format: "markdown"},
		},
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm", Name: "CRM"},
		},
		Files: []*dto.CapabilityBundleFile{},
	}

	plan, err := buildCapabilityBundleInstallPlan("/alice/app/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.directoryItems) != 1 || plan.directoryItems[0].FullCodePath != "/alice/app/openapi/crm" {
		t.Fatalf("unexpected directories: %#v", plan.directoryItems)
	}
	if len(plan.docItems) != 1 {
		t.Fatalf("unexpected doc count: %d", len(plan.docItems))
	}
	got := plan.docItems[0]
	if got.FullCodePath != "/alice/app/openapi/crm/readme.docs" || got.ParentFullCodePath != "/alice/app/openapi/crm" {
		t.Fatalf("unexpected doc path: %#v", got)
	}
	if got.Name != "使用说明" || got.Content != "# 使用说明\n" || got.Tags != "docs,crm" {
		t.Fatalf("unexpected doc metadata: %#v", got)
	}
}

func TestBuildCapabilityBundleInstallPlanSupportsRootPackageFiles(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "消息能力",
		Files: []*dto.CapabilityBundleFile{
			{Path: "send.go", Content: "package openapi"},
		},
	}

	plan, err := buildCapabilityBundleInstallPlan("/customer/crm/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.directoryItems) != 0 {
		t.Fatalf("unexpected directories: %#v", plan.directoryItems)
	}
	if len(plan.fileItems) != 1 || plan.fileItems[0].FullCodePath != "/customer/crm/openapi" {
		t.Fatalf("unexpected files: %#v", plan.fileItems)
	}
}

func TestValidateCapabilityBundleTreeNodes(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "CRM",
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm", Name: "CRM"},
			{RelativePath: "crm/customers", ParentPath: "crm", Type: model.ServiceTreeTypePackage, Code: "customers", Name: "Customers"},
			{RelativePath: "crm/customers/list.table", ParentPath: "crm/customers", Type: model.ServiceTreeTypeFunction, Code: "list.table", Name: "Customer List", TemplateType: "table"},
		},
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm", Name: "CRM"},
			{Path: "crm/customers", Name: "Customers"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "crm/customers", Path: "list.go", Content: "package customers"},
		},
	}

	if err := validateCapabilityBundle(bundle); err != nil {
		t.Fatalf("expected valid bundle with nested tree nodes: %v", err)
	}
}

func TestFilterCapabilityBundleBySubpathRebasesSelectedDirectory(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "CRM Suite",
		Extensions: map[string]interface{}{
			"install": map[string]interface{}{"recommended_subpath": "crm/customers"},
		},
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm", Name: "CRM"},
			{RelativePath: "crm/customers", ParentPath: "crm", Type: model.ServiceTreeTypePackage, Code: "customers", Name: "Customers"},
			{RelativePath: "crm/customers/list.table", ParentPath: "crm/customers", Type: model.ServiceTreeTypeFunction, Code: "list.table", Name: "Customer List", TemplateType: "table"},
			{RelativePath: "crm/customers/readme.docs", ParentPath: "crm/customers", Type: model.ServiceTreeTypeDocs, Code: "readme.docs", Name: "使用说明"},
			{RelativePath: "crm/orders", ParentPath: "crm", Type: model.ServiceTreeTypePackage, Code: "orders", Name: "Orders"},
		},
		Docs: []*dto.CapabilityBundleDoc{
			{RelativePath: "crm/customers/readme.docs", Name: "使用说明", Content: "# Customers\n", Format: "markdown"},
		},
		AgentTasks: []*dto.CapabilityBundleAgentTask{
			{
				RelativePath: "crm/customers",
				Code:         "daily_follow_brief",
				Title:        "客户跟进每日简报",
				Message:      "每天整理客户跟进清单。",
				Schedule:     scheduledsdk.Schedule{Type: scheduledsdk.ScheduleCron, CronExpr: "5 9 * * *"},
			},
		},
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm", Name: "CRM"},
			{Path: "crm/customers", Name: "客户目录"},
			{Path: "crm/orders", Name: "订单目录"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "crm/customers", Path: "list.go", Content: "package customers"},
			{PackagePath: "crm/orders", Path: "list.go", Content: "package orders"},
		},
	}

	filtered, err := filterCapabilityBundleBySubpath(bundle, "crm/customers")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Name != "客户目录" {
		t.Fatalf("filtered name = %q, want 客户目录", filtered.Name)
	}
	if filtered.Extensions["install"] == nil {
		t.Fatalf("expected extensions to be preserved: %#v", filtered.Extensions)
	}

	gotPackages := make([]string, 0, len(filtered.Packages))
	for _, pkg := range filtered.Packages {
		gotPackages = append(gotPackages, pkg.Path+"::"+pkg.Name)
	}
	wantPackages := []string{"customers::客户目录"}
	if !reflect.DeepEqual(gotPackages, wantPackages) {
		t.Fatalf("unexpected packages:\nwant=%#v\ngot=%#v", wantPackages, gotPackages)
	}

	gotTreeNodes := make([]string, 0, len(filtered.TreeNodes))
	for _, node := range filtered.TreeNodes {
		gotTreeNodes = append(gotTreeNodes, node.RelativePath+"<-"+node.ParentPath)
	}
	wantTreeNodes := []string{
		"customers<-",
		"customers/list.table<-customers",
		"customers/readme.docs<-customers",
	}
	if !reflect.DeepEqual(gotTreeNodes, wantTreeNodes) {
		t.Fatalf("unexpected tree nodes:\nwant=%#v\ngot=%#v", wantTreeNodes, gotTreeNodes)
	}

	if len(filtered.Files) != 1 || filtered.Files[0].PackagePath != "customers" || filtered.Files[0].Path != "list.go" {
		t.Fatalf("unexpected files: %#v", filtered.Files)
	}
	if len(filtered.Docs) != 1 || filtered.Docs[0].RelativePath != "customers/readme.docs" {
		t.Fatalf("unexpected docs: %#v", filtered.Docs)
	}
	if len(filtered.AgentTasks) != 1 || filtered.AgentTasks[0].RelativePath != "customers" || filtered.AgentTasks[0].Code != "daily_follow_brief" {
		t.Fatalf("unexpected agent tasks: %#v", filtered.AgentTasks)
	}

	plan, err := buildCapabilityBundleInstallPlan("/alice/app/openapi", filtered)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.directoryItems) != 1 || plan.directoryItems[0].FullCodePath != "/alice/app/openapi/customers" {
		t.Fatalf("unexpected directory plan: %#v", plan.directoryItems)
	}
	if len(plan.fileItems) != 1 || plan.fileItems[0].FullCodePath != "/alice/app/openapi/customers" {
		t.Fatalf("unexpected file plan: %#v", plan.fileItems)
	}
	if len(plan.docItems) != 1 || plan.docItems[0].FullCodePath != "/alice/app/openapi/customers/readme.docs" {
		t.Fatalf("unexpected doc plan: %#v", plan.docItems)
	}
}

func TestFilterCapabilityBundleBySubpathRejectsMissingDirectory(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm/customers", Name: "Customers"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "crm/customers", Path: "list.go", Content: "package customers"},
		},
	}

	_, err := filterCapabilityBundleBySubpath(bundle, "crm/orders")
	if err == nil {
		t.Fatal("expected missing subpath error")
	}
	if !strings.Contains(err.Error(), "未匹配到可安装目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCapabilityBundleTreeNodesRejectsMissingParent(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm/customers/list.table", ParentPath: "crm/customers", Type: model.ServiceTreeTypeFunction, Code: "list.table"},
		},
		Files: []*dto.CapabilityBundleFile{
			{Path: "list.go", Content: "package crm"},
		},
	}

	err := validateCapabilityBundle(bundle)
	if err == nil {
		t.Fatal("expected missing parent validation error")
	}
	if !strings.Contains(err.Error(), "parent_path 未在 tree_nodes 中声明") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCapabilityBundleDocsRejectsNonDocsNode(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm"},
		},
		Docs: []*dto.CapabilityBundleDoc{
			{RelativePath: "crm", Content: "# CRM\n"},
		},
	}

	err := validateCapabilityBundle(bundle)
	if err == nil {
		t.Fatal("expected docs validation error")
	}
	if !strings.Contains(err.Error(), "不是 docs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCapabilityRelativeTreeNodePathSupportsNestedFunction(t *testing.T) {
	t.Parallel()

	root := &model.ServiceTree{Code: "crm", Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/app/crm"}
	node := &model.ServiceTree{Code: "list.table", Type: model.ServiceTreeTypeFunction, FullCodePath: "/alice/app/crm/customers/list.table"}

	got, err := capabilityRelativeTreeNodePath(root, node, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "crm/customers/list.table" {
		t.Fatalf("relative path = %q", got)
	}
}

func TestCapabilityRelativePackagePathCanExcludeRootCode(t *testing.T) {
	t.Parallel()

	root := &model.ServiceTree{
		Code:         "openapi",
		FullCodePath: "/system/openapi",
	}
	nested := &model.ServiceTree{
		Code:         "message",
		FullCodePath: "/system/openapi/platform/message",
	}

	got, err := capabilityRelativePackagePath(root, nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "platform/message" {
		t.Fatalf("unexpected relative path: %s", got)
	}

	got, err = capabilityRelativePackagePath(root, nested, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "openapi/platform/message" {
		t.Fatalf("unexpected include-root relative path: %s", got)
	}
}
