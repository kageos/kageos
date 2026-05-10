package repository

import (
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

func TestBuildTreeFromNodesDedupesFullCodePath(t *testing.T) {
	t.Parallel()

	repo := &ServiceTreeRepository{}
	root := &model.ServiceTree{
		Base:         models.Base{ID: 1},
		Name:         "工具",
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/system/tools",
		AppID:        1,
	}
	firstMessage := &model.ServiceTree{
		Base:         models.Base{ID: 2},
		Name:         "消息",
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/system/tools/message",
		AppID:        1,
	}
	duplicateMessage := &model.ServiceTree{
		Base:         models.Base{ID: 3},
		Name:         "重复消息",
		Type:         model.ServiceTreeTypePackage,
		FullCodePath: "/system/tools/message",
		AppID:        1,
	}
	send := &model.ServiceTree{
		Base:         models.Base{ID: 4},
		Name:         "发送",
		Type:         model.ServiceTreeTypeFunction,
		FullCodePath: "/system/tools/message/send",
		AppID:        1,
	}

	roots := repo.buildTreeFromNodes([]*model.ServiceTree{root, firstMessage, duplicateMessage, send})

	if len(roots) != 1 {
		t.Fatalf("expected one root, got %d", len(roots))
	}
	if len(roots[0].Children) != 1 {
		t.Fatalf("expected duplicate child path to be shown once, got %d", len(roots[0].Children))
	}
	if roots[0].Children[0].ID != firstMessage.ID {
		t.Fatalf("expected first node to be kept, got id %d", roots[0].Children[0].ID)
	}
	if len(roots[0].Children[0].Children) != 1 {
		t.Fatalf("expected child function to remain attached, got %d", len(roots[0].Children[0].Children))
	}
	if roots[0].Children[0].Children[0].ID != send.ID {
		t.Fatalf("expected send function to remain attached, got id %d", roots[0].Children[0].Children[0].ID)
	}
}
