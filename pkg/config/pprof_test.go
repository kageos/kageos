package config

import "testing"

func TestPprofDefaultsDisabled(t *testing.T) {
	tests := []struct {
		name    string
		enabled func() bool
	}{
		{name: "api gateway nil", enabled: func() bool { return (*APIGatewayConfig)(nil).IsPprofEnabled() }},
		{name: "api gateway default", enabled: func() bool { return (&APIGatewayConfig{}).IsPprofEnabled() }},
		{name: "app server nil", enabled: func() bool { return (*AppServerConfig)(nil).IsPprofEnabled() }},
		{name: "app server default", enabled: func() bool { return (&AppServerConfig{}).IsPprofEnabled() }},
		{name: "app storage nil", enabled: func() bool { return (*AppStorageConfig)(nil).IsPprofEnabled() }},
		{name: "app storage default", enabled: func() bool { return (&AppStorageConfig{}).IsPprofEnabled() }},
		{name: "agent server nil", enabled: func() bool { return (*AgentServerConfig)(nil).IsPprofEnabled() }},
		{name: "agent server default", enabled: func() bool { return (&AgentServerConfig{}).IsPprofEnabled() }},
		{name: "connector server nil", enabled: func() bool { return (*ConnectorServerConfig)(nil).IsPprofEnabled() }},
		{name: "connector server default", enabled: func() bool { return (&ConnectorServerConfig{}).IsPprofEnabled() }},
		{name: "hr server nil", enabled: func() bool { return (*HRServerConfig)(nil).IsPprofEnabled() }},
		{name: "hr server default", enabled: func() bool { return (&HRServerConfig{}).IsPprofEnabled() }},
		{name: "message server nil", enabled: func() bool { return (*MessageServerConfig)(nil).IsPprofEnabled() }},
		{name: "message server default", enabled: func() bool { return (&MessageServerConfig{}).IsPprofEnabled() }},
		{name: "timer scheduler nil", enabled: func() bool { return (*TimerSchedulerConfig)(nil).IsPprofEnabled() }},
		{name: "timer scheduler default", enabled: func() bool { return (&TimerSchedulerConfig{}).IsPprofEnabled() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.enabled() {
				t.Fatal("pprof should be disabled by default")
			}
		})
	}
}

func TestPprofExplicitConfig(t *testing.T) {
	enabled := true
	disabled := false
	tests := []struct {
		name    string
		enabled func(*bool) bool
	}{
		{name: "api gateway", enabled: func(value *bool) bool {
			return (&APIGatewayConfig{Server: GatewayServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "app server", enabled: func(value *bool) bool {
			return (&AppServerConfig{Server: AppServerServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "app storage", enabled: func(value *bool) bool { return (&AppStorageConfig{}).withPprof(value).IsPprofEnabled() }},
		{name: "agent server", enabled: func(value *bool) bool {
			return (&AgentServerConfig{Server: AgentServerServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "connector server", enabled: func(value *bool) bool {
			return (&ConnectorServerConfig{Server: ConnectorServerServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "hr server", enabled: func(value *bool) bool {
			return (&HRServerConfig{Server: HRServerServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "message server", enabled: func(value *bool) bool {
			return (&MessageServerConfig{Server: MessageServerHTTPConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
		{name: "timer scheduler", enabled: func(value *bool) bool {
			return (&TimerSchedulerConfig{Server: TimerSchedulerServerConfig{EnablePprof: value}}).IsPprofEnabled()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.enabled(&enabled) {
				t.Fatal("explicit enable_pprof=true should enable pprof")
			}
			if tt.enabled(&disabled) {
				t.Fatal("explicit enable_pprof=false should disable pprof")
			}
		})
	}
}

func (c *AppStorageConfig) withPprof(value *bool) *AppStorageConfig {
	c.Server.EnablePprof = value
	return c
}
