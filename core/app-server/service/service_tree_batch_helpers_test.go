package service

import (
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestValidateDirectoryScaffoldItemsForGoPackagesRejectsInvalidCode(t *testing.T) {
	t.Parallel()

	req := &dto.BatchCreateDirectoryTreeReq{
		User: "luobei",
		App:  "demo",
		Items: []*dto.DirectoryScaffoldItem{
			{FullCodePath: "/luobei/demo/ticket-system", Name: "工单"},
		},
	}

	err := validateDirectoryScaffoldItemsForGoPackages(req)
	if err == nil {
		t.Fatal("expected invalid package code to be rejected")
	}
	if !strings.Contains(err.Error(), "非法 Go package 名称") {
		t.Fatalf("unexpected error: %v", err)
	}
}
