package main

import "testing"

func TestUnifiedStartupIncludesMessageServer(t *testing.T) {
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

	messageServer := serviceByName["message-server"]
	if messageServer == nil {
		t.Fatal("message-server is not registered in unified startup")
	}
	if messageServer.Main == nil {
		t.Fatal("message-server Main is nil")
	}
	if !hasDependency(messageServer, "hr-server") {
		t.Fatal("message-server should wait for hr-server")
	}

	agentServer := serviceByName["agent-server"]
	if agentServer == nil || !hasDependency(agentServer, "message-server") {
		t.Fatal("agent-server should wait for message-server")
	}

	appServer := serviceByName["app-server"]
	if appServer == nil || !hasDependency(appServer, "message-server") {
		t.Fatal("app-server should wait for message-server")
	}

	apiGateway := serviceByName["api-gateway"]
	if apiGateway == nil || !hasDependency(apiGateway, "message-server") {
		t.Fatal("api-gateway should wait for message-server")
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
