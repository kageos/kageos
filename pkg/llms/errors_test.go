package llms

import (
	"errors"
	"testing"
)

func TestContextWindowClassifierUsesStructuredCodesAndRejectsTokenRateLimits(t *testing.T) {
	if !IsContextWindowProviderError("context_length_exceeded", "invalid_request_error", "messages", "request rejected") {
		t.Fatal("structured context_length_exceeded code should be classified")
	}
	for _, message := range []string{
		"Rate limit reached for tokens per minute (TPM)",
		"insufficient_quota: please add credits",
		"max_tokens parameter must be less than 4096",
		"too many tokens consumed this minute",
	} {
		if IsContextWindowErrorMessage(message) {
			t.Fatalf("non-context token error was misclassified: %q", message)
		}
	}
}

func TestProviderHTTPErrorPreservesContextWindowDetails(t *testing.T) {
	err := providerHTTPError(400, []byte(`{"error":{"message":"This model's maximum context length is 128,000 tokens.","type":"invalid_request_error","param":"messages","code":"context_length_exceeded"}}`))
	if !IsContextWindowError(err) {
		t.Fatalf("error = %v, want context-window classification", err)
	}
	if got := ContextWindowLimitFromError(err); got != 128000 {
		t.Fatalf("limit = %d, want 128000", got)
	}
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Code != "context_length_exceeded" || providerErr.Param != "messages" {
		t.Fatalf("provider error fields not preserved: %#v", providerErr)
	}
}
