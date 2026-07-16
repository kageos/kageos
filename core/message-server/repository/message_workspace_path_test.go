package repository

import "testing"

func TestResolveMessageWorkspacePath(t *testing.T) {
	testCases := []struct {
		name         string
		sourcePath   string
		fullCodePath string
		parentPath   string
		templateType string
		want         string
	}{
		{
			name: "function source keeps concrete resource", sourcePath: "/system/democase/site_monitor/check_once.form",
			parentPath: "/system/democase/site_monitor", templateType: "form", want: "/system/democase/site_monitor/check_once.form",
		},
		{
			name: "query is stripped without changing resource ownership", fullCodePath: "/system/democase/site_monitor/check_once.form?source=notification",
			want: "/system/democase/site_monitor/check_once.form",
		},
		{
			name: "suffixless resource stays canonical", sourcePath: "/alice/demo/sweep", parentPath: "/alice/demo",
			templateType: "FormTemplate", want: "/alice/demo/sweep",
		},
		{
			name: "legacy message falls back to parent", parentPath: "/alice/demo", want: "/alice/demo",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveMessageWorkspacePath(tc.sourcePath, tc.fullCodePath, tc.parentPath, tc.templateType)
			if got != tc.want {
				t.Fatalf("ResolveMessageWorkspacePath() = %q, want %q", got, tc.want)
			}
		})
	}
}
