package supervisor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGroupWaitsForServiceMainToReturn(t *testing.T) {
	stopCh := make(chan struct{})
	started := make(chan struct{})
	stopped := make(chan struct{})

	group := Group{
		Services: []Service{{
			Name: "test-service",
			Main: func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
				close(started)
				readyCh <- struct{}{}
				<-stopCh
				close(stopped)
				return nil
			},
		}},
		StartupTimeout: time.Second,
	}

	done := make(chan error, 1)
	go func() {
		done <- group.Run(context.Background(), stopCh)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("service did not start")
	}

	select {
	case err := <-done:
		t.Fatalf("group finished before stopCh was closed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	close(stopCh)

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("service did not observe stopCh")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected group error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("group did not finish after stopCh was closed")
	}
}

func TestGroupWaitsForDependencies(t *testing.T) {
	stopCh := make(chan struct{})
	depStarted := make(chan struct{})
	depReady := make(chan struct{})
	childStarted := make(chan struct{})

	group := Group{
		Services: []Service{
			{
				Name: "database",
				Main: func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
					close(depStarted)
					<-depReady
					readyCh <- struct{}{}
					<-stopCh
					return nil
				},
			},
			{
				Name:      "api",
				DependsOn: []string{"database"},
				Main: func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
					close(childStarted)
					readyCh <- struct{}{}
					<-stopCh
					return nil
				},
			},
		},
		StartupTimeout: time.Second,
	}

	done := make(chan error, 1)
	go func() {
		done <- group.Run(context.Background(), stopCh)
	}()

	select {
	case <-depStarted:
	case <-time.After(time.Second):
		t.Fatal("dependency did not start")
	}

	select {
	case <-childStarted:
		t.Fatal("dependent service started before dependency was ready")
	case <-time.After(20 * time.Millisecond):
	}

	close(depReady)

	select {
	case <-childStarted:
	case <-time.After(time.Second):
		t.Fatal("dependent service did not start after dependency was ready")
	}

	close(stopCh)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected group error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("group did not finish after stopCh was closed")
	}
}

func TestGroupRejectsUnknownDependency(t *testing.T) {
	err := Group{
		Services: []Service{{
			Name:      "api",
			DependsOn: []string{"database"},
			Main: func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error {
				return nil
			},
		}},
	}.Run(context.Background(), make(chan struct{}))

	if err == nil || !strings.Contains(err.Error(), "depends on unknown service database") {
		t.Fatalf("expected unknown dependency error, got %v", err)
	}
}
