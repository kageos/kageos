package github

import (
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestGitHubTemplatesDecodeSchema(t *testing.T) {
	formTemplates := map[string]*app.FormTemplate{
		"ConnectionStatusTemplate":    ConnectionStatusTemplate,
		"GitHubAPIGetTemplate":        GitHubAPIGetTemplate,
		"GitHubMeTemplate":            GitHubMeTemplate,
		"GitHubRepoDetailTemplate":    GitHubRepoDetailTemplate,
		"GitHubRepoLanguagesTemplate": GitHubRepoLanguagesTemplate,
	}
	for name, template := range formTemplates {
		if _, _, err := widget.DecodeForm(templateCallbacks(template.OnSelectFuzzyMap), template.Request, template.Response); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}

	tableTemplates := map[string]*app.TableTemplate{
		"GitHubEmailsTemplate":           GitHubEmailsTemplate,
		"GitHubFollowersTemplate":        GitHubFollowersTemplate,
		"GitHubFollowingTemplate":        GitHubFollowingTemplate,
		"GitHubGistsTemplate":            GitHubGistsTemplate,
		"GitHubOrgsTemplate":             GitHubOrgsTemplate,
		"GitHubRepoBranchesTemplate":     GitHubRepoBranchesTemplate,
		"GitHubRepoCommitsTemplate":      GitHubRepoCommitsTemplate,
		"GitHubRepoContentsTemplate":     GitHubRepoContentsTemplate,
		"GitHubRepoContributorsTemplate": GitHubRepoContributorsTemplate,
		"GitHubRepoIssuesTemplate":       GitHubRepoIssuesTemplate,
		"GitHubRepoPullsTemplate":        GitHubRepoPullsTemplate,
		"GitHubRepoReleasesTemplate":     GitHubRepoReleasesTemplate,
		"GitHubRepoTagsTemplate":         GitHubRepoTagsTemplate,
		"GitHubRepoWorkflowRunsTemplate": GitHubRepoWorkflowRunsTemplate,
		"GitHubRepoWorkflowsTemplate":    GitHubRepoWorkflowsTemplate,
		"GitHubReposTemplate":            GitHubReposTemplate,
		"GitHubStarredReposTemplate":     GitHubStarredReposTemplate,
	}
	for name, template := range tableTemplates {
		if _, _, err := widget.DecodeTable(templateCallbacks(template.OnSelectFuzzyMap), template.Request, template.AutoCrudTable); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}
}

func templateCallbacks(fuzzy map[string]app.OnSelectFuzzy) map[string][]string {
	if len(fuzzy) == 0 {
		return nil
	}
	callbacks := make(map[string][]string, len(fuzzy))
	for field := range fuzzy {
		callbacks[field] = []string{app.CallbackTypeOnSelectFuzzy}
	}
	return callbacks
}
