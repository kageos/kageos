package config

import (
	"fmt"
	"strings"
)

const minimumControlPlaneSecretLength = 32

// ControlPlaneConfig contains credentials used only by trusted core services
// to authenticate internal control-plane messages. It must never be copied to
// SDK env_vars or user App containers.
type ControlPlaneConfig struct {
	Secret string `mapstructure:"secret"`
}

func (c ControlPlaneConfig) GetSecret() (string, error) {
	secret := strings.TrimSpace(c.Secret)
	if len([]byte(secret)) < minimumControlPlaneSecretLength {
		return "", fmt.Errorf("control_plane.secret must contain at least %d bytes", minimumControlPlaneSecretLength)
	}
	return secret, nil
}

func GetControlPlaneSecret() (string, error) {
	cfg := GetGlobalSharedConfig()
	if cfg == nil {
		return "", fmt.Errorf("global shared config is unavailable")
	}
	return cfg.ResolveControlPlaneSecret()
}

// ResolveControlPlaneSecret supports existing installations created before
// control_plane.secret existed. New deployments always receive a dedicated
// secret; legacy configs temporarily reuse the already-private JWT secret.
// controlauth still derives independent keys for every scope.
func (c *GlobalSharedConfig) ResolveControlPlaneSecret() (string, error) {
	if c == nil {
		return "", fmt.Errorf("global shared config is unavailable")
	}
	if strings.TrimSpace(c.ControlPlane.Secret) != "" {
		return c.ControlPlane.GetSecret()
	}
	return (ControlPlaneConfig{Secret: c.JWT.Secret}).GetSecret()
}
