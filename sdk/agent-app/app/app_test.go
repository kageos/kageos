package app

import (
	"context"
	"testing"

	"github.com/kageos/kageos/pkg/subjects"
	"github.com/kageos/kageos/sdk/agent-app/env"
)

func TestResolveNATSURL(t *testing.T) {
	t.Setenv("NATS_URL", "")
	if got := resolveNATSURL(); got != "nats://127.0.0.1:4222" {
		t.Fatalf("expected default NATS URL, got %s", got)
	}

	t.Setenv("NATS_URL", "nats://example:4222")
	if got := resolveNATSURL(); got != "nats://example:4222" {
		t.Fatalf("expected env NATS URL, got %s", got)
	}
}

func TestBuildAppSubjects(t *testing.T) {
	oldUser, oldApp, oldVersion := env.User, env.App, env.Version
	env.User, env.App, env.Version = "alice", "demo", "v7"
	defer func() {
		env.User, env.App, env.Version = oldUser, oldApp, oldVersion
	}()

	got := buildAppSubjects()
	if got.InvokeCommand != subjects.BuildAppInvokeSubject("alice", "demo", "v7") {
		t.Fatalf("unexpected invoke command subject: %s", got.InvokeCommand)
	}
	if got.InvokeReply != subjects.BuildAppServerAppInvokeReplySubject("alice", "demo", "v7") {
		t.Fatalf("unexpected invoke reply subject: %s", got.InvokeReply)
	}
	if got.ControlCommand != subjects.BuildAppControlSubject("alice", "demo", "v7") {
		t.Fatalf("unexpected control command subject: %s", got.ControlCommand)
	}
	if got.LifecycleEvent != subjects.BuildRuntimeLifecycleEventSubject("alice", "demo", "v7") {
		t.Fatalf("unexpected lifecycle event subject: %s", got.LifecycleEvent)
	}
	if got.DiscoveryRequest != subjects.AppDiscoveryRequestSubject {
		t.Fatalf("unexpected discovery request subject: %s", got.DiscoveryRequest)
	}
}

func TestMarkShutdownRequestedIsIdempotent(t *testing.T) {
	app := &App{}

	if ok := app.markShutdownRequested(); !ok {
		t.Fatal("expected first shutdown mark to succeed")
	}
	if ok := app.markShutdownRequested(); ok {
		t.Fatal("expected second shutdown mark to be skipped")
	}
	if !app.shutdownRequested {
		t.Fatal("expected shutdownRequested to stay true")
	}
}

func TestCloseExitSignalIsIdempotent(t *testing.T) {
	app := &App{
		exit: make(chan struct{}),
	}

	app.closeExitSignal()
	select {
	case <-app.exit:
	default:
		t.Fatal("expected exit channel to be closed")
	}

	app.closeExitSignal()
	select {
	case <-app.exit:
	default:
		t.Fatal("expected exit channel to remain closed")
	}
}

func TestRuntimeShutdownRequestedCanBeResetForCleanup(t *testing.T) {
	app := &App{}
	ctx := context.Background()

	if ok := app.markRuntimeShutdownRequested(ctx); !ok {
		t.Fatal("expected first runtime shutdown mark to succeed")
	}
	if ok := app.markRuntimeShutdownRequested(ctx); ok {
		t.Fatal("expected duplicate runtime shutdown mark to be skipped")
	}
	if !app.shutdownRequested {
		t.Fatal("expected shutdownRequested to be true")
	}

	app.resetShutdownRequestedForCleanup()
	if app.shutdownRequested {
		t.Fatal("expected shutdownRequested to be reset before cleanup")
	}
}
