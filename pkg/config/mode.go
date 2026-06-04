package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigModeDev  = "dev"
	ConfigModeProd = "prod"
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

func getConfigEnv() string {
	if mode := discoverWorkspaceMode(); mode != "" {
		return mode
	}
	return ConfigModeProd
}

func discoverWorkspaceMode() string {
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
		if mode := readWorkspaceModeFromRoot(root); mode != "" {
			return mode
		}
	}
	return ""
}

func readWorkspaceModeFromRoot(root string) string {
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
		if !ok || strings.TrimSpace(key) != "KAGEOS_MODE" {
			continue
		}
		return normalizeConfigMode(strings.Trim(strings.TrimSpace(value), `"'`))
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
