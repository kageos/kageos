package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
)

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version   string `json:"version"`
	CreatedAt string `json:"created_at"`
	Status    string `json:"status"`
}

// VersionData version.json 文件结构体
type VersionData struct {
	User           string        `json:"user"`
	App            string        `json:"app"`
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version"`
	Versions       []VersionInfo `json:"versions"`
}

type RuntimeManifest struct {
	SchemaVersion     string `json:"schema_version"`
	User              string `json:"user"`
	App               string `json:"app"`
	Version           string `json:"version"`
	VersionNum        int    `json:"version_num"`
	BinaryName        string `json:"binary_name"`
	BinaryPath        string `json:"binary_path"`
	HostBinaryPath    string `json:"host_binary_path,omitempty"`
	ContainerName     string `json:"container_name"`
	RuntimeInstanceID string `json:"runtime_instance_id,omitempty"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
	BuiltAt           string `json:"built_at"`
	StartedAt         string `json:"started_at,omitempty"`
	UpdatedAt         string `json:"updated_at"`
	Error             string `json:"error,omitempty"`
}

type versionMetadataPaths struct {
	metadataDir         string
	versionJSONPath     string
	currentVersionPath  string
	currentAppPath      string
	runtimeManifestPath string
}

func buildVersionMetadataPaths(appPaths runtimeAppPaths) versionMetadataPaths {
	return versionMetadataPaths{
		metadataDir:         appPaths.MetadataDir(),
		versionJSONPath:     appPaths.VersionJSONPath(),
		currentVersionPath:  appPaths.CurrentVersionPath(),
		currentAppPath:      appPaths.CurrentAppPath(),
		runtimeManifestPath: appPaths.RuntimeManifestPath(),
	}
}

func (s *AppManageService) getVersionMetadataPaths(user, app string) versionMetadataPaths {
	return buildVersionMetadataPaths(newRuntimeAppPaths(s.config.GetBasePath(), user, app))
}

func (s *AppManageService) readVersionData(versionJSONPath string) (*VersionData, error) {
	data, err := os.ReadFile(versionJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read version.json: %w", err)
	}

	var versionData VersionData
	if err := json.Unmarshal(data, &versionData); err != nil {
		return nil, fmt.Errorf("failed to parse version.json: %w", err)
	}

	return &versionData, nil
}

func (s *AppManageService) writeVersionData(versionJSONPath string, versionData *VersionData) error {
	data, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	if err := writeFileAtomic(versionJSONPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write version.json: %w", err)
	}

	return nil
}

func (s *AppManageService) readRuntimeManifest(path string) (*RuntimeManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest RuntimeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse runtime manifest: %w", err)
	}
	return &manifest, nil
}

func (s *AppManageService) writeRuntimeManifest(path string, manifest *RuntimeManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime manifest: %w", err)
	}
	data = append(data, '\n')

	if err := writeFileAtomic(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write runtime manifest: %w", err)
	}
	return nil
}

func (s *AppManageService) buildRuntimeManifest(user, app string, appPaths runtimeAppPaths, version string, now time.Time) RuntimeManifest {
	binaryName := s.appBinaryName(user, app, version)
	containerBinaryPath := filepath.ToSlash(filepath.Join(s.appContainerPath(), "workplace", "bin", "releases", binaryName))
	hostBinaryPath := filepath.Join(appPaths.BuildOutputDir(s.config.GetBuildOutputDir()), binaryName)
	nowText := now.Format(time.RFC3339)

	return RuntimeManifest{
		SchemaVersion:     "1",
		User:              user,
		App:               app,
		Version:           version,
		VersionNum:        parseRuntimeVersionNum(version),
		BinaryName:        binaryName,
		BinaryPath:        containerBinaryPath,
		HostBinaryPath:    hostBinaryPath,
		ContainerName:     buildContainerName(user, app, version),
		RuntimeInstanceID: s.runtimeInstanceID(),
		Status:            "built",
		CreatedAt:         nowText,
		BuiltAt:           nowText,
		UpdatedAt:         nowText,
	}
}

func (s *AppManageService) writeBuiltRuntimeManifest(user, app string, appPaths runtimeAppPaths, version string) error {
	paths := buildVersionMetadataPaths(appPaths)
	if err := os.MkdirAll(paths.metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	manifest := s.buildRuntimeManifest(user, app, appPaths, version, time.Now())
	return s.writeRuntimeManifest(paths.runtimeManifestPath, &manifest)
}

func (s *AppManageService) updateRuntimeManifestStartup(notification *StartupNotification) error {
	if notification == nil {
		return nil
	}

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), notification.User, notification.App)
	paths := buildVersionMetadataPaths(appPaths)
	now := time.Now()
	startTime := notification.StartTime
	if startTime.IsZero() {
		startTime = now
	}

	manifest, err := s.readRuntimeManifest(paths.runtimeManifestPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		base := s.buildRuntimeManifest(notification.User, notification.App, appPaths, notification.Version, now)
		manifest = &base
	}
	if manifest.Version != "" && manifest.Version != notification.Version {
		return nil
	}

	manifest.SchemaVersion = nonEmpty(manifest.SchemaVersion, "1")
	manifest.User = notification.User
	manifest.App = notification.App
	manifest.Version = notification.Version
	manifest.VersionNum = parseRuntimeVersionNum(notification.Version)
	manifest.BinaryName = nonEmpty(manifest.BinaryName, s.appBinaryName(notification.User, notification.App, notification.Version))
	manifest.BinaryPath = nonEmpty(manifest.BinaryPath, filepath.ToSlash(filepath.Join(s.appContainerPath(), "workplace", "bin", "releases", manifest.BinaryName)))
	manifest.HostBinaryPath = nonEmpty(manifest.HostBinaryPath, filepath.Join(appPaths.BuildOutputDir(s.config.GetBuildOutputDir()), manifest.BinaryName))
	manifest.ContainerName = buildContainerName(notification.User, notification.App, notification.Version)
	manifest.RuntimeInstanceID = nonEmpty(manifest.RuntimeInstanceID, s.runtimeInstanceID())
	manifest.Status = normalizeRuntimeManifestStatus(notification.Status)
	manifest.StartedAt = startTime.Format(time.RFC3339)
	manifest.UpdatedAt = now.Format(time.RFC3339)
	manifest.Error = strings.TrimSpace(notification.Error)
	if manifest.CreatedAt == "" {
		manifest.CreatedAt = now.Format(time.RFC3339)
	}
	if manifest.BuiltAt == "" {
		manifest.BuiltAt = manifest.CreatedAt
	}

	if err := os.MkdirAll(paths.metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}
	return s.writeRuntimeManifest(paths.runtimeManifestPath, manifest)
}

func parseRuntimeVersionNum(version string) int {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	n, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return n
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func normalizeRuntimeManifestStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", "started":
		return "running"
	default:
		return strings.TrimSpace(status)
	}
}

func (s *AppManageService) readCurrentVersion(user, app string) (string, error) {
	paths := s.getVersionMetadataPaths(user, app)

	data, err := os.ReadFile(paths.currentVersionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read current version file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// createDirIfNotExists 创建目录（如果不存在）
func (s *AppManageService) createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// createVersionFiles 创建版本文件
func (s *AppManageService) createVersionFiles(user, app string) error {
	paths := s.getVersionMetadataPaths(user, app)
	versionData := VersionData{
		User:           user,
		App:            app,
		CurrentVersion: "v1",
		LatestVersion:  "v1",
		Versions: []VersionInfo{
			{
				Version:   "v1",
				CreatedAt: time.Now().Format(time.RFC3339),
				Status:    "active",
			},
		},
	}

	if err := s.createDirIfNotExists(paths.metadataDir); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	if err := s.writeVersionData(paths.versionJSONPath, &versionData); err != nil {
		return err
	}

	if err := s.updateCurrentVersionFiles(user, app, versionData.CurrentVersion); err != nil {
		return fmt.Errorf("failed to create current version files: %w", err)
	}

	return nil
}

// updateVersionJson 更新 version.json 文件
func (s *AppManageService) updateVersionJson(appDir, user, app, newVersion string) error {
	paths := buildVersionMetadataPaths(newRuntimeAppPathsFromAppDir(appDir, user, app))

	versionData, err := s.readVersionData(paths.versionJSONPath)
	if err != nil {
		return err
	}

	for i := range versionData.Versions {
		if versionData.Versions[i].Status == "active" {
			versionData.Versions[i].Status = "inactive"
		}
	}

	var versionExists bool
	for i := range versionData.Versions {
		if versionData.Versions[i].Version == newVersion {
			versionData.Versions[i].Status = "active"
			versionData.Versions[i].CreatedAt = time.Now().Format(time.RFC3339)
			versionExists = true
			break
		}
	}

	if !versionExists {
		versionData.Versions = append(versionData.Versions, VersionInfo{
			Version:   newVersion,
			CreatedAt: time.Now().Format(time.RFC3339),
			Status:    "active",
		})
	}

	versionData.CurrentVersion = newVersion
	versionData.LatestVersion = newVersion

	if err := s.writeVersionData(paths.versionJSONPath, versionData); err != nil {
		return err
	}

	if err := s.updateCurrentVersionFiles(versionData.User, versionData.App, newVersion); err != nil {
		logger.Warnf(context.Background(), "[updateVersionJson] Failed to update current version files: %v", err)
	}

	return nil
}

// updateCurrentVersionFiles 更新纯文本版本文件，用于极速启动
func (s *AppManageService) updateCurrentVersionFiles(user, app, version string) error {
	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	paths := buildVersionMetadataPaths(appPaths)

	if err := os.MkdirAll(paths.metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	if err := writeFileAtomic(paths.currentVersionPath, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write current_version.txt: %w", err)
	}

	if err := writeFileAtomic(paths.currentAppPath, []byte(appPaths.AppName()), 0644); err != nil {
		return fmt.Errorf("failed to write current_app.txt: %w", err)
	}

	return nil
}

// createMainGoFile 创建 main.go 文件（已存在则复用，不覆盖）
func (s *AppManageService) createMainGoFile(mainGoPath, user, app string) error {
	if _, err := os.Stat(mainGoPath); err == nil {
		logger.Infof(context.Background(), "[createMainGoFile] main.go already exists, skip: %s", mainGoPath)
		return nil
	}

	content := []byte(`package main

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	return writeFileAtomic(mainGoPath, content, 0644)
}
