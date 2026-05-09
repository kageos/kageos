package service

import "testing"

func TestBuildPreviewFileKey(t *testing.T) {
	service := &StorageService{}

	got, err := service.BuildPreviewFileKey(
		"liubeiluo/ccc/f/ticket/ticket_list.table/2026/05/09/90e2befc-08a7-49b2-9b92-f33148e52ce9.png",
		"ticket.preview.webp",
	)
	if err != nil {
		t.Fatalf("BuildPreviewFileKey returned error: %v", err)
	}

	want := "liubeiluo/ccc/f/ticket/ticket_list.table/2026/05/09/90e2befc-08a7-49b2-9b92-f33148e52ce9.png.thumb.webp"
	if got != want {
		t.Fatalf("BuildPreviewFileKey = %q, want %q", got, want)
	}
}

func TestBuildPreviewFileKeyRejectsUnsafeKey(t *testing.T) {
	service := &StorageService{}

	if _, err := service.BuildPreviewFileKey("../outside.png", "outside.preview.webp"); err == nil {
		t.Fatal("BuildPreviewFileKey expected error for unsafe key")
	}
}
