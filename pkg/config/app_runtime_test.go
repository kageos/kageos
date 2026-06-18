package config

import "testing"

func TestAppRuntimeValidateAppliesContainerDefaults(t *testing.T) {
	t.Setenv("KAGEOS_APP_BASE_IMAGE", "")

	cfg := &AppRuntimeConfig{
		Runtime: RuntimeConfig{
			Port:     9093,
			LogLevel: "info",
		},
		Timeouts: AppRuntimeTimeoutConfig{
			ContainerStartup: 2,
			ContainerCleanup: 10,
		},
		Container: ContainerServiceConfig{
			Timeout:     30,
			NetworkMode: "host",
			Image: ImageConfig{
				BaseImage: "custom-runtime:latest",
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if got := cfg.Container.Runtime; got != defaultContainerRuntime {
		t.Fatalf("Container.Runtime = %q, want %q", got, defaultContainerRuntime)
	}
	if got := cfg.Container.LSMMode; got != defaultContainerLSMMode {
		t.Fatalf("Container.LSMMode = %q, want %q", got, defaultContainerLSMMode)
	}
	if got := cfg.Container.AppArmorProfile; got != defaultAppArmorProfile {
		t.Fatalf("Container.AppArmorProfile = %q, want %q", got, defaultAppArmorProfile)
	}
	if got := cfg.Container.Image.ContainerPath; got != defaultContainerPath {
		t.Fatalf("Container.Image.ContainerPath = %q, want %q", got, defaultContainerPath)
	}
	if got := cfg.Container.Image.BaseImage; got != "custom-runtime:latest" {
		t.Fatalf("Container.Image.BaseImage = %q, want custom-runtime:latest", got)
	}
	if got := cfg.Container.NetworkMode; got != "host" {
		t.Fatalf("Container.NetworkMode = %q, want host", got)
	}
}

func TestAppRuntimeBaseImageEnvOverride(t *testing.T) {
	t.Setenv("KAGEOS_APP_BASE_IMAGE", "registry.example.com/kagebase:stable")

	cfg := &AppRuntimeConfig{
		Container: ContainerServiceConfig{
			Image: ImageConfig{
				BaseImage: "custom-runtime:latest",
			},
		},
	}

	if got := cfg.GetContainerBaseImage(); got != "registry.example.com/kagebase:stable" {
		t.Fatalf("GetContainerBaseImage() = %q, want env override", got)
	}
}

func TestAppRuntimeValidateRejectsUnsupportedContainerRuntime(t *testing.T) {
	cfg := &AppRuntimeConfig{
		Runtime: RuntimeConfig{
			Port:     9093,
			LogLevel: "info",
		},
		Container: ContainerServiceConfig{
			Runtime: "docker",
			Timeout: 30,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported runtime error")
	}
}

func TestAppRuntimeStartupNotificationTimeoutDefault(t *testing.T) {
	cfg := &AppRuntimeConfig{}
	if got := cfg.GetAppStartupNotificationTimeout(); got != 300 {
		t.Fatalf("GetAppStartupNotificationTimeout() = %d, want 300", got)
	}

	cfg.Timeouts.AppStartupNotification = 600
	if got := cfg.GetAppStartupNotificationTimeout(); got != 600 {
		t.Fatalf("GetAppStartupNotificationTimeout() = %d, want 600", got)
	}
}
