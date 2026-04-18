package service

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestBoardServiceUpdatePostRequiresID(t *testing.T) {
	svc := &BoardService{}

	_, err := svc.UpdatePost(context.Background(), &dto.UpdatePostReq{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "帖子ID不能为空" {
		t.Fatalf("err = %q, want %q", err.Error(), "帖子ID不能为空")
	}
}

func TestServiceTreeMutationUpdateBoardRequiresID(t *testing.T) {
	svc := &serviceTreeMutationService{}

	err := svc.UpdateBoard(context.Background(), &dto.UpdateBoardReq{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "版块ID不能为空" {
		t.Fatalf("err = %q, want %q", err.Error(), "版块ID不能为空")
	}
}

func TestServiceTreeMutationUpdateMetadataRequiresID(t *testing.T) {
	svc := &serviceTreeMutationService{}

	err := svc.UpdateServiceTreeMetadata(context.Background(), &dto.UpdateServiceTreeMetadataReq{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "服务目录ID不能为空" {
		t.Fatalf("err = %q, want %q", err.Error(), "服务目录ID不能为空")
	}
}
