package dto

import "testing"

func TestUpdateAppReqBuildSummary(t *testing.T) {
	t.Run("prefers explicit summary", func(t *testing.T) {
		req := &UpdateAppReq{
			Summary:           "直接摘要",
			Requirement:       "旧需求",
			ChangeDescription: "旧描述",
		}

		if got := req.BuildSummary(); got != "直接摘要" {
			t.Fatalf("BuildSummary() = %q, want %q", got, "直接摘要")
		}
	})

	t.Run("combines requirement and change description", func(t *testing.T) {
		req := &UpdateAppReq{
			Requirement:       "补充上传能力",
			ChangeDescription: "新增 output_files 返回处理",
		}

		want := "需求：补充上传能力\n\n变更描述：新增 output_files 返回处理"
		if got := req.BuildSummary(); got != want {
			t.Fatalf("BuildSummary() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to single field", func(t *testing.T) {
		if got := (&UpdateAppReq{Requirement: "只保留需求"}).BuildSummary(); got != "只保留需求" {
			t.Fatalf("BuildSummary() = %q, want %q", got, "只保留需求")
		}
		if got := (&UpdateAppReq{ChangeDescription: "只保留描述"}).BuildSummary(); got != "只保留描述" {
			t.Fatalf("BuildSummary() = %q, want %q", got, "只保留描述")
		}
	})
}
