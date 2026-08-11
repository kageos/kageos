package server

import (
	"strings"
	"testing"
)

func TestDecodeAuthorityResponseRejectsOversizedPayload(t *testing.T) {
	var out map[string]interface{}
	err := decodeAuthorityResponse(strings.NewReader(strings.Repeat("x", maxAuthorityResponseBytes+1)), &out)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("decodeAuthorityResponse error = %v, want size limit", err)
	}
}
