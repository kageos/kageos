package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestWorkspaceFileProfileReadsCSVSample(t *testing.T) {
	workspaceFileProfileCache = syncMapForWorkspaceFileProfileTest()
	oldResolve := workspaceFileProfileResolveFileRefs
	oldClient := workspaceFileProfileHTTPClient
	defer func() {
		workspaceFileProfileResolveFileRefs = oldResolve
		workspaceFileProfileHTTPClient = oldClient
		workspaceFileProfileCache = syncMapForWorkspaceFileProfileTest()
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte("线索来源,客户名称,线索状态,预计成交金额\n线上推广,阿里巴巴集团,初步接触,500000\n展会获取,腾讯科技,需求确认,1000000\n"))
	}))
	defer server.Close()
	workspaceFileProfileHTTPClient = server.Client()
	workspaceFileProfileResolveFileRefs = func(ctx context.Context, req *dto.ResolveFileRefsReq) (*dto.ResolveFileRefsResp, error) {
		return &dto.ResolveFileRefsResp{Files: []dto.ResolvedFile{{
			Ref:               "kageos/workspace/leads.csv",
			Name:              "leads.csv",
			ContentType:       "text/csv",
			Size:              128,
			ServerDownloadURL: server.URL,
		}}}, nil
	}

	block := workspaceFileProfileBlockForRefs(context.Background(), "kageos/workspace/leads.csv")

	for _, want := range []string{
		"<file_profile>",
		`"kind": "csv"`,
		`"线索来源"`,
		`"客户名称"`,
		`"阿里巴巴集团"`,
		fileProfileInstruction,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("profile block missing %q:\n%s", want, block)
		}
	}
}

func TestUserContentForLLMCanIncludeFileProfileBlock(t *testing.T) {
	refs := "kageos/workspace/leads.csv"
	profile := "<file_profile>\n{\"protocol_version\":\"workspace_file_profile.v1\"}\n</file_profile>"

	content := userContentForLLMWithFileProfileBlock("这个能搞成系统吗？", &refs, profile)

	if !strings.Contains(content, "<files>") || !strings.Contains(content, refs) {
		t.Fatalf("content missing files block:\n%s", content)
	}
	if !strings.Contains(content, "<file_profile>") {
		t.Fatalf("content missing profile block:\n%s", content)
	}
	if !strings.Contains(content, "优先使用 file_profile") {
		t.Fatalf("content missing profile instruction:\n%s", content)
	}
	if !strings.HasSuffix(content, "这个能搞成系统吗？") {
		t.Fatalf("content should end with user demand:\n%s", content)
	}
}

func syncMapForWorkspaceFileProfileTest() sync.Map {
	return sync.Map{}
}
