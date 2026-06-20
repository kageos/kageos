package github

import "testing"

func TestNormalizeGitHubRepoSelection(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		wantOwner string
		wantRepo  string
	}{
		{name: "repo name uses owner", owner: "liu-cn", repo: "kageos", wantOwner: "liu-cn", wantRepo: "kageos"},
		{name: "full name wins", owner: "", repo: "openai/codex", wantOwner: "openai", wantRepo: "codex"},
		{name: "full name trims slash", owner: "ignored", repo: "/openai/codex/", wantOwner: "openai", wantRepo: "codex"},
		{name: "empty repo", owner: "liu-cn", repo: " ", wantOwner: "liu-cn", wantRepo: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo := normalizeGitHubRepoSelection(tt.owner, tt.repo)
			if gotOwner != tt.wantOwner || gotRepo != tt.wantRepo {
				t.Fatalf("normalizeGitHubRepoSelection() = (%q, %q), want (%q, %q)", gotOwner, gotRepo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
