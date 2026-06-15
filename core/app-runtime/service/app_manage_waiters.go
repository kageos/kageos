package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
)

// registerStartupWaiter 注册启动等待器
func (s *AppManageService) registerStartupWaiter(user, app, version string) chan *StartupNotification {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	waiterChan := make(chan *StartupNotification, 1)
	s.startupWaiters[key] = waiterChan
	return waiterChan
}

// unregisterStartupWaiter 注销启动等待器
func (s *AppManageService) unregisterStartupWaiter(user, app, version string) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	if waiterChan, exists := s.startupWaiters[key]; exists {
		close(waiterChan)
		delete(s.startupWaiters, key)
	}
}

// notifyStartupWaiter 通知启动等待器
func (s *AppManageService) notifyStartupWaiter(user, app, version string, notification *StartupNotification) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.startupWaitersMu.RLock()
	waiterChan, exists := s.startupWaiters[key]
	s.startupWaitersMu.RUnlock()

	if exists {
		select {
		case waiterChan <- notification:
		default:
		}
	}
}

// registerCloseWaiter 注册关闭等待器
func (s *AppManageService) registerCloseWaiter(user, app, version string) chan *CloseNotification {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.Lock()
	defer s.closeWaitersMu.Unlock()

	waiterChan := make(chan *CloseNotification, 1)
	s.closeWaiters[key] = waiterChan
	return waiterChan
}

// unregisterCloseWaiter 注销关闭等待器
func (s *AppManageService) unregisterCloseWaiter(user, app, version string) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.Lock()
	defer s.closeWaitersMu.Unlock()

	if waiterChan, exists := s.closeWaiters[key]; exists {
		close(waiterChan)
		delete(s.closeWaiters, key)
	}
}

// notifyCloseWaiter 通知关闭等待器
func (s *AppManageService) notifyCloseWaiter(user, app, version string, notification *CloseNotification) {
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.closeWaitersMu.RLock()
	waiterChan, exists := s.closeWaiters[key]
	s.closeWaitersMu.RUnlock()

	if exists {
		select {
		case waiterChan <- notification:
		default:
			// 通道已满或已关闭，忽略
		}
	}
}

// NotifyStartup 通知应用启动完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyStartup(notification *StartupNotification) {
	if notification == nil {
		return
	}
	s.notifyStartupWaiter(notification.User, notification.App, notification.Version, notification)
	if err := s.updateRuntimeManifestStartup(notification); err != nil {
		logger.Warnf(context.Background(), "[NotifyStartup] Failed to update runtime manifest: %v", err)
	}
}

// NotifyClose 通知应用关闭完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyClose(notification *CloseNotification) {
	if notification == nil {
		return
	}
	s.notifyCloseWaiter(notification.User, notification.App, notification.Version, notification)
}

// RegisterStartupWaiter 注册启动等待器
func (s *AppManageService) RegisterStartupWaiter(key string) {
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	if _, exists := s.startupWaiters[key]; !exists {
		s.startupWaiters[key] = make(chan *StartupNotification, 1)
	}
}

// UnregisterStartupWaiter 注销启动等待器
func (s *AppManageService) UnregisterStartupWaiter(key string) {
	s.startupWaitersMu.Lock()
	defer s.startupWaitersMu.Unlock()

	delete(s.startupWaiters, key)
}

// GetStartupWaiter 获取启动等待器
func (s *AppManageService) GetStartupWaiter(key string) chan *StartupNotification {
	s.startupWaitersMu.RLock()
	defer s.startupWaitersMu.RUnlock()

	return s.startupWaiters[key]
}

// waitForStartup 等待应用启动完成（内部方法）
func (s *AppManageService) waitForStartup(ctx context.Context, user, app, version string, timeout time.Duration) (*StartupNotification, error) {
	waiterChan := s.registerStartupWaiter(user, app, version)
	defer s.unregisterStartupWaiter(user, app, version)

	select {
	case notification := <-waiterChan:
		return notification, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for startup notification")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *AppManageService) waitForStartupNotificationOrRuntimeExit(
	ctx context.Context,
	ref AppVersionRef,
	waiterChan <-chan *StartupNotification,
	timeout time.Duration,
) (*StartupNotification, error) {
	if s.runtimeDriver == nil {
		return nil, fmt.Errorf("app runtime driver not available")
	}
	if timeout <= 0 {
		timeout = s.appStartupNotificationTimeout()
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(startupRuntimePollInterval(timeout))
	defer ticker.Stop()

	for {
		select {
		case notification, ok := <-waiterChan:
			if !ok || notification == nil {
				return nil, fmt.Errorf("startup waiter closed before notification: %s/%s/%s", ref.User, ref.App, ref.Version)
			}
			return notification, nil
		case <-ticker.C:
			running, err := s.runtimeDriver.IsAppVersionRunning(ctx, ref)
			if err != nil {
				continue
			}
			if !running {
				return nil, fmt.Errorf("app runtime exited before startup notification: %s/%s/%s", ref.User, ref.App, ref.Version)
			}
		case <-timer.C:
			running, err := s.runtimeDriver.IsAppVersionRunning(ctx, ref)
			if err != nil {
				return nil, fmt.Errorf("timeout waiting for app startup notification: %s/%s/%s (failed to check runtime status: %w)", ref.User, ref.App, ref.Version, err)
			}
			if !running {
				return nil, fmt.Errorf("app runtime exited before startup notification: %s/%s/%s", ref.User, ref.App, ref.Version)
			}
			return nil, fmt.Errorf("timeout waiting for app startup notification: %s/%s/%s (runtime still running)", ref.User, ref.App, ref.Version)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func startupRuntimePollInterval(timeout time.Duration) time.Duration {
	switch {
	case timeout <= 100*time.Millisecond:
		return 10 * time.Millisecond
	case timeout <= time.Second:
		return 50 * time.Millisecond
	default:
		return time.Second
	}
}
