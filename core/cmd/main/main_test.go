package main

import "testing"

func TestUnifiedStartupUsesCoreMVPServices(t *testing.T) {
	serviceByName := make(map[string]*ServiceInfo, len(services))
	for _, svc := range services {
		if svc == nil {
			t.Fatal("services contains nil ServiceInfo")
		}
		if _, exists := serviceByName[svc.Name]; exists {
			t.Fatalf("duplicate service registered: %s", svc.Name)
		}
		serviceByName[svc.Name] = svc
	}

	for _, name := range []string{"app-runtime", "app-storage", "hr-server", "agent-server", "connector-server", "timer-scheduler", "message-server", "app-server", "api-gateway"} {
		if serviceByName[name] == nil {
			t.Fatalf("%s is not registered in unified startup", name)
		}
	}

	if len(serviceByName) != 9 {
		t.Fatalf("unexpected service count in MVP unified startup: %d", len(serviceByName))
	}

	appServer := serviceByName["app-server"]
	if appServer == nil || !hasDependency(appServer, "app-runtime") {
		t.Fatal("app-server should wait for app-runtime")
	}

	agentServer := serviceByName["agent-server"]
	if agentServer == nil || len(agentServer.DependsOn) != 0 {
		t.Fatal("agent-server should start without service dependencies")
	}

	timerScheduler := serviceByName["timer-scheduler"]
	if timerScheduler == nil || len(timerScheduler.DependsOn) != 0 {
		t.Fatal("timer-scheduler should start without service dependencies")
	}

	messageServer := serviceByName["message-server"]
	if messageServer == nil || !hasDependency(messageServer, "hr-server") {
		t.Fatal("message-server should wait for hr-server")
	}

	apiGateway := serviceByName["api-gateway"]
	for _, dep := range []string{"app-runtime", "app-storage", "hr-server", "agent-server", "connector-server", "timer-scheduler", "message-server", "app-server"} {
		if apiGateway == nil || !hasDependency(apiGateway, dep) {
			t.Fatalf("api-gateway should wait for %s", dep)
		}
	}
}

func hasDependency(svc *ServiceInfo, dep string) bool {
	if svc == nil {
		return false
	}
	for _, candidate := range svc.DependsOn {
		if candidate == dep {
			return true
		}
	}
	return false
}
