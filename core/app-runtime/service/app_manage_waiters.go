package service

import (
	"context"
	"fmt"
	"time"
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
	s.notifyStartupWaiter(notification.User, notification.App, notification.Version, notification)
}

// NotifyClose 通知应用关闭完成（由 NATS 消息处理器调用）
func (s *AppManageService) NotifyClose(notification *CloseNotification) {
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
