package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const appNATSSecretTarget = "kageos-nats"

// AppVersionRef identifies one runnable app version.
type AppVersionRef struct {
	User    string
	App     string
	Version string
}

func (r AppVersionRef) RuntimeName() string {
	return BuildContainerName(r.User, r.App, r.Version)
}

func (r AppVersionRef) AppKey() string {
	return r.User + "/" + r.App
}

// AppVersionSpec describes how an app version should be started.
type AppVersionSpec struct {
	Ref           AppVersionRef
	Image         string
	HostPath      string
	ContainerPath string
	Command       []string
	EnvVars       []string
	Secrets       []RuntimeSecret
}

// RuntimeSecret is sensitive data provisioned by the runtime and mounted into
// a container. Data is sent to Podman through stdin and must never be placed in
// command arguments, environment variables, or logs.
type RuntimeSecret struct {
	Name   string
	Target string
	Data   []byte
}

func appNATSSecretName(ref AppVersionRef) string {
	return ref.RuntimeName() + "-nats"
}

// AppRuntimeInstance is the runtime-neutral view of one app version instance.
type AppRuntimeInstance struct {
	Ref         AppVersionRef
	RuntimeName string
	Running     bool
}

// AppRuntimeDriver is the business-level runtime boundary.
//
// Podman, Kubernetes, or future runtimes should implement this interface. Upper
// layers should talk in app-version lifecycle terms instead of container verbs.
type AppRuntimeDriver interface {
	IsAvailable() bool
	CreateAppVersion(ctx context.Context, spec AppVersionSpec) error
	StartAppVersion(ctx context.Context, spec AppVersionSpec) error
	StopAppVersion(ctx context.Context, ref AppVersionRef) error
	RemoveAppVersion(ctx context.Context, ref AppVersionRef) error
	IsAppVersionRunning(ctx context.Context, ref AppVersionRef) (bool, error)
	ListAppVersions(ctx context.Context) ([]AppRuntimeInstance, error)
}

// PodmanAppRuntimeDriver adapts the existing low-level ContainerOperator to the
// app-version runtime boundary.
type PodmanAppRuntimeDriver struct {
	containerService ContainerOperator
}

func NewPodmanAppRuntimeDriver(containerService ContainerOperator) AppRuntimeDriver {
	if containerService == nil {
		return nil
	}
	return &PodmanAppRuntimeDriver{containerService: containerService}
}

func (d *PodmanAppRuntimeDriver) IsAvailable() bool {
	return d != nil && d.containerService != nil && d.containerService.IsRunning()
}

func (d *PodmanAppRuntimeDriver) CreateAppVersion(ctx context.Context, spec AppVersionSpec) error {
	if d == nil || d.containerService == nil {
		return fmt.Errorf("app runtime driver not available")
	}

	name := spec.Ref.RuntimeName()
	running, err := d.containerService.IsContainerRunning(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check app runtime instance: %w", err)
	}
	if running {
		return fmt.Errorf("app runtime instance %s already exists and is running", name)
	}

	spec, err = normalizeAppVersionSpec(spec)
	if err != nil {
		return err
	}

	createdSecrets := make([]RuntimeSecret, 0, len(spec.Secrets))
	for _, secret := range spec.Secrets {
		if err := d.containerService.CreateSecret(ctx, secret); err != nil {
			cleanupErr := d.removeSecrets(ctx, createdSecrets)
			return errors.Join(fmt.Errorf("failed to create app runtime secret %s: %w", secret.Name, err), cleanupErr)
		}
		createdSecrets = append(createdSecrets, secret)
	}

	if err := d.containerService.RunContainerWithCommand(ctx, spec.Image, name, spec.HostPath, spec.ContainerPath, spec.Command, spec.Secrets, spec.EnvVars...); err != nil {
		cleanupErr := d.removeSecrets(ctx, createdSecrets)
		return errors.Join(fmt.Errorf("failed to create app runtime instance: %w", err), cleanupErr)
	}
	return nil
}

func (d *PodmanAppRuntimeDriver) removeSecrets(ctx context.Context, secrets []RuntimeSecret) error {
	var errs []error
	for _, secret := range secrets {
		if err := d.containerService.RemoveSecret(ctx, secret.Name); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove app runtime secret %s: %w", secret.Name, err))
		}
	}
	return errors.Join(errs...)
}

func (d *PodmanAppRuntimeDriver) StartAppVersion(ctx context.Context, spec AppVersionSpec) error {
	if d == nil || d.containerService == nil {
		return fmt.Errorf("app runtime driver not available")
	}

	name := spec.Ref.RuntimeName()
	running, err := d.containerService.IsContainerRunning(ctx, name)
	if err == nil && running {
		return nil
	}

	if err := d.containerService.StartContainer(ctx, name); err == nil {
		return nil
	}

	return d.CreateAppVersion(ctx, spec)
}

func (d *PodmanAppRuntimeDriver) StopAppVersion(ctx context.Context, ref AppVersionRef) error {
	if d == nil || d.containerService == nil {
		return fmt.Errorf("app runtime driver not available")
	}
	return d.containerService.StopContainer(ctx, ref.RuntimeName())
}

func (d *PodmanAppRuntimeDriver) RemoveAppVersion(ctx context.Context, ref AppVersionRef) error {
	if d == nil || d.containerService == nil {
		return fmt.Errorf("app runtime driver not available")
	}
	containerErr := d.containerService.RemoveContainer(ctx, ref.RuntimeName())
	secretErr := d.containerService.RemoveSecret(ctx, appNATSSecretName(ref))
	return errors.Join(containerErr, secretErr)
}

func (d *PodmanAppRuntimeDriver) IsAppVersionRunning(ctx context.Context, ref AppVersionRef) (bool, error) {
	if d == nil || d.containerService == nil {
		return false, fmt.Errorf("app runtime driver not available")
	}
	return d.containerService.IsContainerRunning(ctx, ref.RuntimeName())
}

func (d *PodmanAppRuntimeDriver) ListAppVersions(ctx context.Context) ([]AppRuntimeInstance, error) {
	if d == nil || d.containerService == nil {
		return nil, fmt.Errorf("app runtime driver not available")
	}

	containers, err := d.containerService.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	instances := make([]AppRuntimeInstance, 0, len(containers))
	for _, c := range containers {
		if len(c.Names) == 0 {
			continue
		}

		name := c.Names[0]
		user, app, version, parseErr := parseContainerName(name)
		if parseErr != nil {
			continue
		}

		instances = append(instances, AppRuntimeInstance{
			Ref: AppVersionRef{
				User:    user,
				App:     app,
				Version: version,
			},
			RuntimeName: name,
			Running:     !c.Exited,
		})
	}

	return instances, nil
}

func normalizeAppVersionSpec(spec AppVersionSpec) (AppVersionSpec, error) {
	if spec.Image == "" {
		return spec, fmt.Errorf("app runtime image cannot be empty")
	}
	if spec.HostPath == "" {
		return spec, fmt.Errorf("app runtime host path cannot be empty")
	}
	if !filepath.IsAbs(spec.HostPath) {
		absHostPath, err := filepath.Abs(spec.HostPath)
		if err != nil {
			return spec, fmt.Errorf("failed to resolve app runtime host path: %w", err)
		}
		spec.HostPath = absHostPath
	}
	if spec.ContainerPath == "" {
		return spec, fmt.Errorf("app runtime container path cannot be empty")
	}
	if len(spec.Command) == 0 {
		return spec, fmt.Errorf("app runtime command cannot be empty")
	}
	seenSecrets := make(map[string]struct{}, len(spec.Secrets))
	for _, secret := range spec.Secrets {
		if strings.TrimSpace(secret.Name) == "" {
			return spec, fmt.Errorf("app runtime secret name cannot be empty")
		}
		if strings.Contains(secret.Name, ",") {
			return spec, fmt.Errorf("app runtime secret name cannot contain commas: %s", secret.Name)
		}
		if strings.TrimSpace(secret.Target) == "" {
			return spec, fmt.Errorf("app runtime secret target cannot be empty: %s", secret.Name)
		}
		if strings.Contains(secret.Target, ",") {
			return spec, fmt.Errorf("app runtime secret target cannot contain commas: %s", secret.Target)
		}
		if len(secret.Data) == 0 {
			return spec, fmt.Errorf("app runtime secret data cannot be empty: %s", secret.Name)
		}
		if _, exists := seenSecrets[secret.Name]; exists {
			return spec, fmt.Errorf("duplicate app runtime secret: %s", secret.Name)
		}
		seenSecrets[secret.Name] = struct{}{}
	}
	return spec, nil
}
