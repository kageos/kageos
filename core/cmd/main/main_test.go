package main

import (
	"testing"

	"github.com/kageos/kageos/pkg/supervisor"
)

func TestUnifiedStartupUsesCoreMVPServices(t *testing.T) {
	serviceByName := make(map[string]supervisor.Service, len(services))
	for _, svc := range services {
		if svc.Name == "" {
			t.Fatal("services contains service with empty name")
		}
		if svc.Main == nil {
			t.Fatalf("%s service main is nil", svc.Name)
		}
		if _, exists := serviceByName[svc.Name]; exists {
			t.Fatalf("duplicate service registered: %s", svc.Name)
		}
		serviceByName[svc.Name] = svc
	}

	for _, name := range []string{"app-runtime", "app-storage", "hr-server", "agent-server", "connector-server", "timer-scheduler", "message-server", "app-server", "api-gateway"} {
		if _, exists := serviceByName[name]; !exists {
			t.Fatalf("%s is not registered in unified startup", name)
		}
	}

	if len(serviceByName) != 9 {
		t.Fatalf("unexpected service count in MVP unified startup: %d", len(serviceByName))
	}

	appServer := serviceByName["app-server"]
	if !hasDependency(appServer, "app-runtime") {
		t.Fatal("app-server should wait for app-runtime")
	}
	if !hasDependency(appServer, "hr-server") {
		t.Fatal("app-server should wait for the authoritative OpenAPI token store")
	}
	for _, name := range []string{"app-storage", "connector-server"} {
		if !hasDependency(serviceByName[name], "hr-server") {
			t.Fatalf("%s should wait for the authoritative OpenAPI token store", name)
		}
	}

	agentServer := serviceByName["agent-server"]
	if !hasDependency(agentServer, "hr-server") {
		t.Fatal("agent-server should wait for the authoritative OpenAPI token store")
	}

	timerScheduler := serviceByName["timer-scheduler"]
	if !hasDependency(timerScheduler, "hr-server") {
		t.Fatal("timer-scheduler should wait for the authoritative OpenAPI token store")
	}

	messageServer := serviceByName["message-server"]
	if !hasDependency(messageServer, "hr-server") {
		t.Fatal("message-server should wait for hr-server")
	}

	apiGateway := serviceByName["api-gateway"]
	for _, dep := range []string{"app-runtime", "app-storage", "hr-server", "agent-server", "connector-server", "timer-scheduler", "message-server", "app-server"} {
		if !hasDependency(apiGateway, dep) {
			t.Fatalf("api-gateway should wait for %s", dep)
		}
	}
}

func hasDependency(svc supervisor.Service, dep string) bool {
	for _, candidate := range svc.DependsOn {
		if candidate == dep {
			return true
		}
	}
	return false
}
