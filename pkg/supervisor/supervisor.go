package supervisor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultStartupTimeout is used when Group.StartupTimeout is not set.
const DefaultStartupTimeout = 100 * time.Second

// ServiceMain is the long-running entrypoint for a supervised service.
type ServiceMain func(ctx context.Context, stopCh <-chan struct{}, readyCh chan<- struct{}) error

// Service describes one long-running service and its startup dependencies.
type Service struct {
	Name      string
	Main      ServiceMain
	DependsOn []string
}

// Logger receives lifecycle logs from Group. It is optional.
type Logger interface {
	Infof(ctx context.Context, format string, args ...interface{})
	Errorf(ctx context.Context, format string, args ...interface{})
}

// Group starts services, waits for dependency readiness, and stops as one unit.
type Group struct {
	Services       []Service
	StartupTimeout time.Duration
	Logger         Logger
}

// Run starts all services and blocks until they stop, fail, or the context ends.
func (g Group) Run(ctx context.Context, stopCh <-chan struct{}) error {
	if err := g.validate(); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	timeout := g.StartupTimeout
	if timeout <= 0 {
		timeout = DefaultStartupTimeout
	}

	errCh := make(chan error, len(g.Services))
	readyMap, readyChannels := buildReadyMap(runCtx, g.Services)

	var wg sync.WaitGroup
	wg.Add(len(g.Services))
	for _, svc := range g.Services {
		svc := svc
		readyCh := readyChannels[svc.Name]
		go func() {
			defer wg.Done()
			if err := g.runService(runCtx, svc, stopCh, readyCh, readyMap, timeout); err != nil {
				errCh <- err
				cancel()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case err := <-errCh:
		cancel()
		<-done
		return err
	case <-ctx.Done():
		cancel()
		<-done
		return ctx.Err()
	}
}

func (g Group) validate() error {
	seen := make(map[string]struct{}, len(g.Services))
	for _, svc := range g.Services {
		if svc.Name == "" {
			return fmt.Errorf("service name is required")
		}
		if svc.Main == nil {
			return fmt.Errorf("%s service main is required", svc.Name)
		}
		if _, exists := seen[svc.Name]; exists {
			return fmt.Errorf("duplicate service registered: %s", svc.Name)
		}
		seen[svc.Name] = struct{}{}
	}

	for _, svc := range g.Services {
		for _, dep := range svc.DependsOn {
			if _, exists := seen[dep]; !exists {
				return fmt.Errorf("%s depends on unknown service %s", svc.Name, dep)
			}
		}
	}
	return nil
}

func buildReadyMap(ctx context.Context, services []Service) (map[string]<-chan struct{}, map[string]chan struct{}) {
	readyMap := make(map[string]<-chan struct{}, len(services))
	readyChannels := make(map[string]chan struct{}, len(services))
	for _, svc := range services {
		readyCh := make(chan struct{}, 1)
		readyDone := make(chan struct{})
		readyChannels[svc.Name] = readyCh
		readyMap[svc.Name] = readyDone

		go func() {
			select {
			case <-readyCh:
				close(readyDone)
			case <-ctx.Done():
			}
		}()
	}
	return readyMap, readyChannels
}

func (g Group) runService(ctx context.Context, svc Service, stopCh <-chan struct{}, readyCh chan<- struct{}, readyMap map[string]<-chan struct{}, timeout time.Duration) error {
	if err := g.waitForDependencies(ctx, svc, readyMap, timeout); err != nil {
		return err
	}

	g.infof(ctx, "[启动] %s 开始执行 Main 函数...", svc.Name)
	startedAt := time.Now()
	serviceReadyCh := make(chan struct{}, 1)
	go func() {
		select {
		case <-serviceReadyCh:
			g.infof(ctx, "[耗时] %s 启动就绪: %s", svc.Name, time.Since(startedAt).Round(time.Millisecond))
			select {
			case readyCh <- struct{}{}:
			case <-ctx.Done():
			}
		case <-ctx.Done():
		}
	}()
	if err := svc.Main(context.WithValue(ctx, "service_name", svc.Name), stopCh, serviceReadyCh); err != nil {
		g.errorf(ctx, "[启动] %s Main 函数返回错误: %v", svc.Name, err)
		return fmt.Errorf("%s 运行失败: %w", svc.Name, err)
	}
	return nil
}

func (g Group) waitForDependencies(ctx context.Context, svc Service, readyMap map[string]<-chan struct{}, timeout time.Duration) error {
	if len(svc.DependsOn) == 0 {
		return nil
	}

	g.infof(ctx, "[启动] %s 等待依赖服务: %v", svc.Name, svc.DependsOn)
	for _, depName := range svc.DependsOn {
		depReadyCh, exists := readyMap[depName]
		if !exists {
			return fmt.Errorf("%s 依赖的服务 %s 不存在", svc.Name, depName)
		}
		g.infof(ctx, "[启动] %s 等待依赖 %s 就绪...", svc.Name, depName)
		select {
		case <-depReadyCh:
			g.infof(ctx, "[启动] %s 的依赖 %s 已就绪", svc.Name, depName)
		case <-time.After(timeout):
			timeoutSeconds := int(timeout / time.Second)
			g.errorf(ctx, "[启动] %s 等待依赖 %s 超时（%d秒）", svc.Name, depName, timeoutSeconds)
			return fmt.Errorf("%s 等待依赖 %s 启动超时（%d秒）", svc.Name, depName, timeoutSeconds)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	g.infof(ctx, "[启动] %s 所有依赖已就绪，开始启动...", svc.Name)
	return nil
}

func (g Group) infof(ctx context.Context, format string, args ...interface{}) {
	if g.Logger != nil {
		g.Logger.Infof(ctx, format, args...)
	}
}

func (g Group) errorf(ctx context.Context, format string, args ...interface{}) {
	if g.Logger != nil {
		g.Logger.Errorf(ctx, format, args...)
	}
}
