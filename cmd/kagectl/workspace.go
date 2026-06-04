package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultStateDir = ".kageos"
	defaultStateEnv = "kageos.env"

	workspaceModeDev  = "dev"
	workspaceModeProd = "prod"
)

type workspaceConfig struct {
	Mode string
	Dev  workspaceDevConfig
}

type workspaceDevConfig struct {
	Engine string
}

func workspaceEnvPath(paths Paths) string {
	if paths.StateEnvPath != "" {
		return paths.StateEnvPath
	}
	return filepath.Join(paths.RepoRoot, defaultStateDir, defaultStateEnv)
}

func workspaceStateDir(paths Paths) string {
	if paths.StateDir != "" {
		return paths.StateDir
	}
	return filepath.Join(paths.RepoRoot, defaultStateDir)
}

func workspaceModeFileExists(paths Paths) bool {
	return fileExists(workspaceEnvPath(paths))
}

func loadWorkspaceConfig(paths Paths) workspaceConfig {
	return loadWorkspaceEnvConfig(paths)
}

func loadWorkspaceEnvConfig(paths Paths) workspaceConfig {
	values, err := readEnvFile(workspaceEnvPath(paths))
	if err != nil {
		return workspaceConfig{}
	}
	cfg := workspaceConfig{
		Mode: normalizeWorkspaceMode(values["KAGEOS_MODE"]),
		Dev: workspaceDevConfig{
			Engine: normalizeDevEngine(values["KAGEOS_DEV_ENGINE"]),
		},
	}
	if cfg.Mode == "" {
		return workspaceConfig{}
	}
	return cfg
}

func writeWorkspaceConfig(paths Paths, mode string, dev workspaceDevConfig) error {
	mode = normalizeWorkspaceMode(mode)
	if mode == "" {
		return fmt.Errorf("workspace mode must be dev or prod")
	}
	if mode == workspaceModeDev {
		dev.Engine = normalizeDevEngine(dev.Engine)
	}
	cfg := workspaceConfig{Mode: mode}
	if mode == workspaceModeDev {
		cfg.Dev = dev
	}
	if err := os.MkdirAll(workspaceStateDir(paths), 0755); err != nil {
		return err
	}
	if err := writeWorkspaceEnvConfig(paths, cfg); err != nil {
		return err
	}
	fmt.Printf("workspace mode: %s (%s)\n", mode, workspaceEnvPath(paths))
	return nil
}

func writeWorkspaceEnvConfig(paths Paths, cfg workspaceConfig) error {
	lines := []string{
		"KAGEOS_MODE=" + cfg.Mode,
	}
	if cfg.Mode == workspaceModeDev {
		lines = append(lines, "KAGEOS_DEV_ENGINE="+normalizeDevEngine(cfg.Dev.Engine))
	}
	lines = append(lines, "")
	return os.WriteFile(workspaceEnvPath(paths), []byte(strings.Join(lines, "\n")), 0600)
}

func currentWorkspaceMode(paths Paths) string {
	if mode := loadWorkspaceConfig(paths).Mode; mode != "" {
		return mode
	}
	return workspaceModeProd
}

func currentDevEngine(paths Paths) string {
	return normalizeDevEngine(loadWorkspaceConfig(paths).Dev.Engine)
}

func normalizeWorkspaceMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case workspaceModeDev:
		return workspaceModeDev
	case workspaceModeProd:
		return workspaceModeProd
	default:
		return ""
	}
}

func normalizeDevEngine(engine string) string {
	engine = strings.ToLower(strings.TrimSpace(engine))
	switch engine {
	case "auto", "docker", "podman":
		return engine
	default:
		return "podman"
	}
}
