package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestValidateDirectoryScaffoldItemsRejectsInvalidCode(t *testing.T) {
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
		t.Fatal("expected invalid directory code to be rejected")
	}
	if !strings.Contains(err.Error(), "不支持的英文标识") {
		t.Fatalf("unexpected error: %v", err)
	}
}
