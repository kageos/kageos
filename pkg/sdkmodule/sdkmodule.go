package sdkmodule

import "strings"

const (
	ModulePath       = "github.com/kageos/kageos-sdk"
	Version          = "v0.1.0"
	AgentAppPrefix   = ModulePath + "/agent-app"
	LegacyModulePath = "github.com/kageos/kageos"
	LegacySDKPrefix  = LegacyModulePath + "/sdk/agent-app"
)

func AgentAppImport(packagePath string) string {
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		return AgentAppPrefix
	}
	return AgentAppPrefix + "/" + packagePath
}

func LegacyAgentAppImport(packagePath string) string {
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		return LegacySDKPrefix
	}
	return LegacySDKPrefix + "/" + packagePath
}
