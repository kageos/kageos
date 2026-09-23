package model

import (
	"encoding/json"
	"testing"
)

func TestLegacyArchiveScopeRequiresCompleteSinglePathEvidence(t *testing.T) {
	for _, item := range []struct{ summary, want string }{
		{`{"top_resource_paths":[{"resource_path":"/alice/ops/team/a.form","count":10}]}`, "/alice/ops/team/a.form"},
		{`{"top_resource_paths":[{"resource_path":"/alice/ops/team/a.form","count":9}]}`, ""},
		{`{"top_resource_paths":[{"resource_path":"/alice/ops/team/a.form","count":5},{"resource_path":"/alice/ops/other/b.form","count":5}]}`, ""},
		{`{"top_resource_paths":[{"resource_path":"/alice/ops2/a.form","count":10}]}`, ""},
		{`not-json`, ""},
	} {
		batch := LogArchiveBatch{TenantUser: "alice", App: "ops", RecordCount: 10, SummaryJSON: json.RawMessage(item.summary)}
		if got := batch.SingleResourcePath(); got != item.want {
			t.Fatalf("scope=%q want=%q for %s", got, item.want, item.summary)
		}
	}
}
