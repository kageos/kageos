package sdkmodule

import "strings"

const (
	ModulePath     = "github.com/kageos/kageos-sdk"
	Version        = "v0.2.1"
	AgentAppPrefix = ModulePath + "/agent-app"
)

func AgentAppImport(packagePath string) string {
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		return AgentAppPrefix
	}
	return AgentAppPrefix + "/" + packagePath
}
