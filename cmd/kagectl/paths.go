package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolvePaths(opts commonOptions) (Paths, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return Paths{}, err
	}
	stateDir := filepath.Join(repoRoot, defaultStateDir)

	prodDir := opts.ProdDir
	if prodDir == "" {
		prodDir = filepath.Join(repoRoot, defaultProdDir)
	} else if !filepath.IsAbs(prodDir) {
		prodDir = filepath.Join(repoRoot, prodDir)
	}

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = filepath.Join(prodDir, defaultConfigName)
	} else if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(repoRoot, configPath)
	}

	return Paths{
		RepoRoot:     repoRoot,
		StateDir:     stateDir,
		StateEnvPath: filepath.Join(stateDir, defaultStateEnv),
		ProdDir:      prodDir,
		ConfigPath:   configPath,
		GeneratedDir: filepath.Join(prodDir, defaultGenerated),
	}, nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "deploy", "prod")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot find repository root from %s", wd)
		}
		dir = parent
	}
}

func resolveRelativePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
