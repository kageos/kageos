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
			name:       "scheduled site monitor form uses package directory",
			sourcePath: "/system/democase/site_monitor/sweep.form", parentPath: "/system/democase/site_monitor",
			templateType: "form", want: "/system/democase/site_monitor",
		},
		{
			name:         "typed function derives directory without metadata",
			fullCodePath: "/system/democase/site_monitor/sweep.form?source=notification",
			want:         "/system/democase/site_monitor",
		},
		{
			name:       "directory source remains unchanged",
			sourcePath: "/system/democase/site_monitor", parentPath: "/system/democase",
			want: "/system/democase/site_monitor",
		},
		{
			name:       "legacy function route uses template metadata",
			sourcePath: "/alice/demo/sweep", parentPath: "/alice/demo", templateType: "FormTemplate",
			want: "/alice/demo",
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
