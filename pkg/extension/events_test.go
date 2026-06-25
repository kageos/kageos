package extension

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
)

func TestEmitUsesGlobalThenNamedListeners(t *testing.T) {
	resetListenerRegistryForTest(t)

	var calls []string
	RegisterListener("*", func(context.Context, Event) error {
		calls = append(calls, "global")
		return nil
	})
	RegisterListener("app.created", func(_ context.Context, event Event) error {
		calls = append(calls, event.Name)
		return nil
	})

	if err := Emit(context.Background(), Event{Name: "App.Created"}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if got := strings.Join(calls, ","); got != "global,app.created" {
		t.Fatalf("calls = %q, want global,app.created", got)
	}
}

func TestEmitNormalizesEventAndFillsTime(t *testing.T) {
	resetListenerRegistryForTest(t)

	var got Event
	RegisterListener("form.submitted", func(_ context.Context, event Event) error {
		got = event
		return nil
	})

	if err := Emit(context.Background(), Event{
		Name:      " Form.Submitted ",
		Source:    "app-server",
		Actor:     "alice",
		RequestID: "req-123",
		Resource: ResourceRef{
			Type:         "function",
			FullCodePath: "/alice/crm/leads/form",
			Name:         "Lead Form",
		},
		Payload: struct {
			SubmitID string
		}{SubmitID: "submit-1"},
	}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	if got.Name != "form.submitted" {
		t.Fatalf("event name = %q, want form.submitted", got.Name)
	}
	if got.Time.IsZero() {
		t.Fatal("event time was not filled")
	}
	if got.Time.Location() != time.UTC {
		t.Fatalf("event time location = %v, want UTC", got.Time.Location())
	}
	if got.Resource.FullCodePath != "/alice/crm/leads/form" {
		t.Fatalf("resource full_code_path = %q", got.Resource.FullCodePath)
	}
}

func TestEmitFillsCommonFieldsFromContext(t *testing.T) {
	resetListenerRegistryForTest(t)

	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:      "trace-1",
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceAgent,
	})

	var got Event
	RegisterListener("app.updated", func(_ context.Context, event Event) error {
		got = event
		return nil
	})

	if err := Emit(ctx, Event{Name: "app.updated"}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	if got.Actor != "alice" {
		t.Fatalf("actor = %q, want alice", got.Actor)
	}
	if got.RequestID != "trace-1" {
		t.Fatalf("request id = %q, want trace-1", got.RequestID)
	}
	if got.Source != contextx.ClientSourceAgent {
		t.Fatalf("source = %q, want %q", got.Source, contextx.ClientSourceAgent)
	}
}

func TestEmitDoesNotOverrideExplicitCommonFields(t *testing.T) {
	resetListenerRegistryForTest(t)

	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:      "trace-1",
		RequestUser:  "alice",
		ClientSource: contextx.ClientSourceAgent,
	})

	var got Event
	RegisterListener("app.updated", func(_ context.Context, event Event) error {
		got = event
		return nil
	})

	if err := Emit(ctx, Event{
		Name:      "app.updated",
		Source:    "app-server",
		Actor:     "system",
		RequestID: "manual-request",
	}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}

	if got.Actor != "system" {
		t.Fatalf("actor = %q, want system", got.Actor)
	}
	if got.RequestID != "manual-request" {
		t.Fatalf("request id = %q, want manual-request", got.RequestID)
	}
	if got.Source != "app-server" {
		t.Fatalf("source = %q, want app-server", got.Source)
	}
}

func TestAsPayloadReturnsTypedPayload(t *testing.T) {
	type appCreatedPayload struct {
		AppCode string
	}

	payload, ok := AsPayload[appCreatedPayload](Event{
		Payload: appCreatedPayload{AppCode: "crm"},
	})
	if !ok {
		t.Fatal("AsPayload returned ok=false")
	}
	if payload.AppCode != "crm" {
		t.Fatalf("payload AppCode = %q, want crm", payload.AppCode)
	}

	if _, ok := AsPayload[string](Event{Payload: appCreatedPayload{AppCode: "crm"}}); ok {
		t.Fatal("AsPayload returned ok=true for wrong payload type")
	}
}

func TestEmitReturnsJoinedListenerErrors(t *testing.T) {
	resetListenerRegistryForTest(t)

	errOne := errors.New("one")
	errTwo := errors.New("two")
	RegisterListener("app.created", func(context.Context, Event) error { return errOne })
	RegisterListener("app.created", func(context.Context, Event) error { return errTwo })

	err := Emit(context.Background(), Event{Name: "app.created"})
	if !errors.Is(err, errOne) || !errors.Is(err, errTwo) {
		t.Fatalf("Emit error = %v, want joined listener errors", err)
	}
}

func TestEmitNoListenersIsNoop(t *testing.T) {
	resetListenerRegistryForTest(t)

	if err := Emit(context.Background(), Event{Name: "missing"}); err != nil {
		t.Fatalf("Emit without listeners returned error: %v", err)
	}
}

func resetListenerRegistryForTest(t *testing.T) {
	t.Helper()
	listenerRegistry.Lock()
	listenerRegistry.listeners = make(map[string][]Listener)
	listenerRegistry.Unlock()
	t.Cleanup(func() {
		listenerRegistry.Lock()
		listenerRegistry.listeners = make(map[string][]Listener)
		listenerRegistry.Unlock()
	})
}
