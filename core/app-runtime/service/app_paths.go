package service

import (
	"fmt"
	"path/filepath"
	"strings"
)

type runtimeAppPaths struct {
	user   string
	app    string
	appDir string
}

func newRuntimeAppPaths(basePath, user, app string) runtimeAppPaths {
	return newRuntimeAppPathsFromAppDir(filepath.Join(basePath, user, app), user, app)
}

func newRuntimeAppPathsFromAppDir(appDir, user, app string) runtimeAppPaths {
	return runtimeAppPaths{
		user:   user,
		app:    app,
		appDir: appDir,
	}
}

func (p runtimeAppPaths) UserDir() string {
	return filepath.Dir(p.appDir)
}

func (p runtimeAppPaths) AppDir() string {
	return p.appDir
}

func (p runtimeAppPaths) AppName() string {
	return fmt.Sprintf("%s_%s", p.user, p.app)
}

func (p runtimeAppPaths) APIDir() string {
	return filepath.Join(p.appDir, "code", "api")
}

func (p runtimeAppPaths) CmdAppDir() string {
	return filepath.Join(p.appDir, "code", "cmd", "app")
}

func (p runtimeAppPaths) MainGoPath() string {
	return filepath.Join(p.CmdAppDir(), "main.go")
}

func (p runtimeAppPaths) WorkplaceDir() string {
	return filepath.Join(p.appDir, "workplace")
}

func (p runtimeAppPaths) WorkplaceSubDir(name string) string {
	return filepath.Join(p.WorkplaceDir(), name)
}

func (p runtimeAppPaths) MetadataDir() string {
	return p.WorkplaceSubDir("metadata")
}

func (p runtimeAppPaths) VersionJSONPath() string {
	return filepath.Join(p.MetadataDir(), "version.json")
}

func (p runtimeAppPaths) CurrentVersionPath() string {
	return filepath.Join(p.MetadataDir(), "current_version.txt")
}

func (p runtimeAppPaths) CurrentAppPath() string {
	return filepath.Join(p.MetadataDir(), "current_app.txt")
}

func (p runtimeAppPaths) LogsDir() string {
	return p.WorkplaceSubDir("logs")
}

func (p runtimeAppPaths) LogFileName(version string) string {
	return fmt.Sprintf("%s_%s_%s.log", p.user, p.app, version)
}

func (p runtimeAppPaths) LogFile(version string) string {
	return filepath.Join(p.LogsDir(), p.LogFileName(version))
}

func (p runtimeAppPaths) BuildOutputDir(outputDir string) string {
	return filepath.Join(p.appDir, outputDir)
}

func (p runtimeAppPaths) NamespaceAPIImport(packagePath string) string {
	base := fmt.Sprintf("github.com/kageos/kageos/namespace/%s/%s/code/api", p.user, p.app)
	cleanPackagePath := strings.Trim(packagePath, "/")
	if cleanPackagePath == "" {
		return base
	}
	return base + "/" + cleanPackagePath
}

func (p runtimeAppPaths) TrimAppPrefix(fullCodePath string) string {
	prefix := fmt.Sprintf("/%s/%s", p.user, p.app)
	relativePath := strings.TrimSpace(fullCodePath)
	relativePath = strings.TrimPrefix(relativePath, prefix)
	return strings.TrimPrefix(relativePath, "/")
}
