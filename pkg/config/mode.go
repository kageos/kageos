package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigModeDev         = "dev"
	ConfigModeProd        = "prod"
	ConfigDevEngineAuto   = "auto"
	ConfigDevEngineDocker = "docker"
	ConfigDevEnginePodman = "podman"
)

// GetConfigEnv returns the local Kageos runtime mode.
//
// The canonical source is .kageos/kageos.env, written by kagectl init/init --dev.
func GetConfigEnv() string {
	return getConfigEnv()
}

func IsDevMode() bool {
	return getConfigEnv() == ConfigModeDev
}

func GetDevEngine() string {
	if !IsDevMode() {
		return ""
	}
	if engine := normalizeDevEngine(os.Getenv("KAGEOS_DEV_ENGINE")); engine != "" {
		return engine
	}
	if engine := discoverWorkspaceDevEngine(); engine != "" {
		return engine
	}
	return ConfigDevEnginePodman
}

func getConfigEnv() string {
	if mode := discoverWorkspaceMode(); mode != "" {
		return mode
	}
	return ConfigModeProd
}

func discoverWorkspaceMode() string {
	return discoverWorkspaceValue(readWorkspaceModeFromRoot)
}

func discoverWorkspaceDevEngine() string {
	return discoverWorkspaceValue(readWorkspaceDevEngineFromRoot)
}

func discoverWorkspaceValue(read func(string) string) string {
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 8)
	addRoot := func(root string) {
		root = strings.TrimSpace(root)
		if root == "" {
			return
		}
		abs, err := filepath.Abs(root)
		if err == nil {
			root = abs
		}
		root = filepath.Clean(root)
		if _, ok := seen[root]; ok {
			return
		}
		seen[root] = struct{}{}
		candidates = append(candidates, root)
	}

	addRoot(os.Getenv("KAGEOS_ROOT"))
	addRoot(GetKageosRoot())
	if cwd, err := os.Getwd(); err == nil {
		for _, dir := range ancestorDirs(cwd) {
			addRoot(dir)
		}
	}

	for _, root := range candidates {
		if value := read(root); value != "" {
			return value
		}
	}
	return ""
}

func readWorkspaceModeFromRoot(root string) string {
	return normalizeConfigMode(readWorkspaceEnvValueFromRoot(root, "KAGEOS_MODE"))
}

func readWorkspaceDevEngineFromRoot(root string) string {
	return normalizeDevEngine(readWorkspaceEnvValueFromRoot(root, "KAGEOS_DEV_ENGINE"))
}

func readWorkspaceEnvValueFromRoot(root string, wantKey string) string {
	data, err := os.ReadFile(filepath.Join(root, ".kageos", "kageos.env"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != wantKey {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return ""
}

func normalizeConfigMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case ConfigModeDev:
		return ConfigModeDev
	case ConfigModeProd:
		return ConfigModeProd
	default:
		return ""
	}
}

func normalizeDevEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case ConfigDevEngineAuto:
		return ConfigDevEngineAuto
	case ConfigDevEngineDocker:
		return ConfigDevEngineDocker
	case ConfigDevEnginePodman:
		return ConfigDevEnginePodman
	default:
		return ""
	}
}
