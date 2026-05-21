package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/dto"
)

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
