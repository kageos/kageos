package service

import (
	"strings"
	"testing"
)

func TestCheckWorkspaceGoSourcesDetectsCommonSDKIssues(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_record_list.table.go",
			Content: `package nps

import "time"

type NpsRecord struct {
	Status string ` + "`json:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:success,#F56C6C\"`" + `
	Attachment string ` + "`json:\"attachment\" widget:\"name:附件;type:file;readonly:true\"`" + `
}
`,
		},
	})
	if result.Passed {
		t.Fatal("expected issues")
	}
	for _, want := range []string{"file_name", "go_import", "options_colors", "widget_type", "widget_tag"} {
		if !hasCheckIssueCategory(result.Issues, want) {
			t.Fatalf("expected category %s in %#v", want, result.Issues)
		}
	}
}

func TestCheckWorkspaceGoSourcesPassesCleanFile(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_record_list.go",
			Content: `package nps

type NpsRecord struct {
	Status string ` + "`json:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:67C23A,F56C6C\"`" + `
	Score int ` + "`json:\"score\" widget:\"name:评分;type:integer\"`" + `
}
`,
		},
	})
	if !result.Passed {
		t.Fatalf("expected clean result, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesDetectsRequestModelDuplicate(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_record_list.go",
			Content: `package nps

type NpsRecord struct {
	Status string ` + "`json:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:67C23A,F56C6C\"`" + `
}

func (NpsRecord) TableName() string { return "nps_record" }

type NpsRecordListReq struct {
	Status string ` + "`json:\"status\" form:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:67C23A,F56C6C\"`" + `
}
`,
		},
	})
	if !hasCheckIssueCategory(result.Issues, "table_request_duplicate") {
		t.Fatalf("expected table_request_duplicate issue, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesAllowsPageSortReqRequestModelOverlap(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_record_list.go",
			Content: `package nps

import "github.com/kageos/kageos/pkg/gormx/query"

type NpsRecord struct {
	Status string ` + "`json:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:67C23A,F56C6C\"`" + `
}

func (NpsRecord) TableName() string { return "nps_record" }

type NpsRecordListReq struct {
	Status string ` + "`json:\"status\" form:\"status\" widget:\"name:状态;type:select;options:开放,关闭;options_colors:67C23A,F56C6C\"`" + `
	query.PageSortReq ` + "`widget:\"-\"`" + `
}
`,
		},
	})
	if hasCheckIssueCategory(result.Issues, "table_request_duplicate") {
		t.Fatalf("did not expect table_request_duplicate issue, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesDetectsOnSelectFuzzyMismatch(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/auction", []goSourceFileForCheck{
		{
			Name: "auction_bid_list.go",
			Content: `package auction

import "github.com/kageos/kageos/sdk/agent-app/app"

type AuctionBid struct {
	ItemID int ` + "`json:\"item_id\" widget:\"name:拍品ID;type:ID\" callback:\"OnSelectFuzzy\"`" + `
}

var T = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"item_id": nil,
		},
	},
}
`,
		},
	})
	if !hasCheckIssueCategory(result.Issues, "onselect_fuzzy") {
		t.Fatalf("expected onselect_fuzzy issue, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesDetectsMissingWidgetType(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps.go",
			Content: `package nps

type NpsRecord struct {
	Title string ` + "`json:\"title\" widget:\"name:标题\"`" + `
}
`,
		},
	})
	if !hasCheckIssueCategory(result.Issues, "widget_type") {
		t.Fatalf("expected widget_type issue, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesDetectsWidgetGoTypeMismatch(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/files", []goSourceFileForCheck{
		{
			Name: "files.go",
			Content: `package files

type ConvertReq struct {
	InputFiles []string ` + "`json:\"input_files\" widget:\"name:输入文件;type:files;max_count:3\"`" + `
	Enabled string ` + "`json:\"enabled\" widget:\"name:启用;type:switch\"`" + `
	Score float64 ` + "`json:\"score\" widget:\"name:评分;type:integer\"`" + `
	Amount int ` + "`json:\"amount\" widget:\"name:金额;type:float\"`" + `
	Owner int ` + "`json:\"owner\" widget:\"name:负责人;type:user\"`" + `
}
`,
		},
	})
	if !hasCheckIssueCategory(result.Issues, "widget_go_type") {
		t.Fatalf("expected widget_go_type issue, got %#v", result.Issues)
	}
	for _, want := range []string{"type:files", "type:switch", "type:integer", "type:float", "type:user"} {
		if !hasCheckIssueMessage(result.Issues, want) {
			t.Fatalf("expected issue message containing %q, got %#v", want, result.Issues)
		}
	}
}

func TestCheckWorkspaceGoSourcesDetectsUnknownSDKSelector(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_statistics.go",
			Content: `package nps

import "github.com/kageos/kageos/sdk/agent-app/app"

func bad(req *app.OnSelectFuzzyReq) {}
`,
		},
	})
	if !hasCheckIssueCategory(result.Issues, "sdk_selector") {
		t.Fatalf("expected sdk_selector issue, got %#v", result.Issues)
	}
}

func TestCheckWorkspaceGoSourcesAllowsKnownSDKSelector(t *testing.T) {
	result := checkWorkspaceGoSources("/u/app/nps", []goSourceFileForCheck{
		{
			Name: "nps_statistics.go",
			Content: `package nps

import "github.com/kageos/kageos/sdk/agent-app/callback"

func ok(req *callback.OnSelectFuzzyReq) {}
`,
		},
	})
	if hasCheckIssueCategory(result.Issues, "sdk_selector") {
		t.Fatalf("did not expect sdk_selector issue, got %#v", result.Issues)
	}
}

func hasCheckIssueCategory(issues []checkWorkspaceCodeIssue, category string) bool {
	for _, issue := range issues {
		if issue.Category == category {
			return true
		}
	}
	return false
}

func hasCheckIssueMessage(issues []checkWorkspaceCodeIssue, text string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.Message, text) {
			return true
		}
	}
	return false
}
