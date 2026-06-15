package service

import (
	"strings"

	appconfig "github.com/kageos/kageos/pkg/config"
)

func (s *AppManageService) appBinaryName(user, app, version string) string {
	cfg := (*appconfig.AppManageServiceConfig)(nil)
	if s != nil {
		cfg = s.config
	}
	format := cfg.GetBinaryNameFormat()
	binaryName := strings.ReplaceAll(format, "{user}", user)
	binaryName = strings.ReplaceAll(binaryName, "{app}", app)
	binaryName = strings.ReplaceAll(binaryName, "{version}", version)
	return binaryName
}

func (s *AppManageService) appContainerPath() string {
	if s == nil || s.runtimeConfig == nil {
		return (&appconfig.ContainerServiceConfig{}).GetContainerPath()
	}
	return s.runtimeConfig.GetContainerPath()
}

func (s *AppManageService) runtimeInstanceID() string {
	if s == nil || s.runtimeConfig == nil {
		return ""
	}
	return s.runtimeConfig.GetRuntimeInstanceID()
}
