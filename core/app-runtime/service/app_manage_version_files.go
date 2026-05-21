package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

type versionMetadataPaths struct {
	metadataDir        string
	versionJSONPath    string
	currentVersionPath string
	currentAppPath     string
}

func buildVersionMetadataPaths(appPaths runtimeAppPaths) versionMetadataPaths {
	return versionMetadataPaths{
		metadataDir:        appPaths.MetadataDir(),
		versionJSONPath:    appPaths.VersionJSONPath(),
		currentVersionPath: appPaths.CurrentVersionPath(),
		currentAppPath:     appPaths.CurrentAppPath(),
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
