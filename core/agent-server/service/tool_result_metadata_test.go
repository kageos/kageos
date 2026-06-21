package service

import (
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestMetadataForDisplayFileFields(t *testing.T) {
	metadata := metadataForDisplayFileFields(" output_files ", "", "preview_files", "output_files")
	if metadata == nil {
		t.Fatal("expected metadata")
	}
	got := metadata.DisplayFileFields
	if len(got) != 2 || got[0] != "output_files" || got[1] != "preview_files" {
		t.Fatalf("unexpected display file fields: %#v", got)
	}
}

func TestCollectDisplayFileFields(t *testing.T) {
	outputFiles := &widget.Field{Code: "output_files"}
	outputFiles.Widget.Type = widget.TypeFiles
	title := &widget.Field{Code: "title"}
	title.Widget.Type = widget.TypeInput

	got := collectDisplayFileFields([]*widget.Field{nil, outputFiles, title})
	if len(got) != 1 || got[0] != "output_files" {
		t.Fatalf("unexpected display file fields: %#v", got)
	}
}
