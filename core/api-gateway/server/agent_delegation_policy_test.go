package server

import (
	"net/http"
	"testing"
)

func TestAgentDelegationPolicyAllowsOnlyPOSTForWorkspaceFileReplace(t *testing.T) {
	const path = "/workspace/api/v1/workspace/files/replace"
	if !isAllowedAgentDelegatedAPI(http.MethodPost, path) {
		t.Fatal("POST workspace file replace should be an allowed Agent delegation")
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		if isAllowedAgentDelegatedAPI(method, path) {
			t.Fatalf("%s workspace file replace unexpectedly allowed", method)
		}
	}
}
