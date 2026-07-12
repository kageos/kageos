package service

import (
	"context"
	"fmt"
	"path/filepath"
)

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
	Secrets       []ContainerSecret
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
	if err := d.containerService.RunContainerWithCommandAndSecrets(ctx, spec.Image, name, spec.HostPath, spec.ContainerPath, spec.Command, spec.Secrets, spec.EnvVars...); err != nil {
		return fmt.Errorf("failed to create app runtime instance: %w", err)
	}
	return nil
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
	return d.containerService.RemoveContainer(ctx, ref.RuntimeName())
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
	return spec, nil
}
