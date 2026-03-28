package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
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

// createDirIfNotExists 创建目录（如果不存在）
func (s *AppManageService) createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// createVersionFiles 创建版本文件
func (s *AppManageService) createVersionFiles(metadataDir, user, app string) error {
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

	data, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	versionFile := filepath.Join(metadataDir, "version.json")
	if err := os.WriteFile(versionFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write version.json: %w", err)
	}

	return nil
}

// updateVersionJson 更新 version.json 文件
func (s *AppManageService) updateVersionJson(appDir, user, app, newVersion string) error {
	versionFile := filepath.Join(appDir, "workplace/metadata/version.json")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		return fmt.Errorf("failed to read version.json: %w", err)
	}

	var versionData VersionData
	if err := json.Unmarshal(data, &versionData); err != nil {
		return fmt.Errorf("failed to parse version.json: %w", err)
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

	updatedData, err := json.MarshalIndent(versionData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version.json: %w", err)
	}

	if err := os.WriteFile(versionFile, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write version.json: %w", err)
	}

	if err := s.updateCurrentVersionFiles(versionData.User, versionData.App, newVersion); err != nil {
		logger.Warnf(context.Background(), "[updateVersionJson] Failed to update current version files: %v", err)
	}

	return nil
}

// updateCurrentVersionFiles 更新纯文本版本文件，用于极速启动
func (s *AppManageService) updateCurrentVersionFiles(user, app, version string) error {
	metadataDir := filepath.Join("namespace", user, app, "workplace", "metadata")

	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	versionFile := filepath.Join(metadataDir, "current_version.txt")
	if err := os.WriteFile(versionFile, []byte(version), 0644); err != nil {
		return fmt.Errorf("failed to write current_version.txt: %w", err)
	}

	appFile := filepath.Join(metadataDir, "current_app.txt")
	appName := fmt.Sprintf("%s_%s", user, app)
	if err := os.WriteFile(appFile, []byte(appName), 0644); err != nil {
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
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

func main() {
	err := app.Run()
	if err != nil {
		panic(err)
	}
}
`)

	return os.WriteFile(mainGoPath, content, 0644)
}
