package apperror

import (
	"errors"
	"testing"
)

func TestTypedErrorPreservesKindAndCause(t *testing.T) {
	cause := errors.New("database detail")
	err := NotFound("resource not found", cause)

	typed, ok := As(err)
	if !ok || typed.Kind() != KindNotFound || typed.Message() != "resource not found" {
		t.Fatalf("unexpected typed error: %#v", typed)
	}
	if !errors.Is(err, cause) {
		t.Fatal("typed error did not preserve cause")
	}
}
