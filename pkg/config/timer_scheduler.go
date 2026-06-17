package config

import (
	"fmt"
	"sync"
	"time"
)

var (
	timerSchedulerConfig *TimerSchedulerConfig
	timerSchedulerOnce   sync.Once
	timerSchedulerMu     sync.RWMutex
)

func GetTimerSchedulerConfig() *TimerSchedulerConfig {
	timerSchedulerOnce.Do(func() {
		cfg := &TimerSchedulerConfig{}
		if err := loadYAMLConfig("timer-scheduler.yaml", cfg); err != nil {
			fmt.Printf("Failed to load timer-scheduler config: %v\n", err)
			cfg = &TimerSchedulerConfig{}
		}
		timerSchedulerMu.Lock()
		timerSchedulerConfig = cfg
		timerSchedulerMu.Unlock()
	})

	timerSchedulerMu.RLock()
	defer timerSchedulerMu.RUnlock()
	return timerSchedulerConfig
}

type TimerSchedulerConfig struct {
	Server    TimerSchedulerServerConfig `mapstructure:"server"`
	DB        DBConfig                   `mapstructure:"db"`
	Scheduler TimerSchedulerLoopConfig   `mapstructure:"scheduler"`
}

type TimerSchedulerServerConfig struct {
	Port        int    `mapstructure:"port"`
	ListenHost  string `mapstructure:"listen_host"`
	LogLevel    string `mapstructure:"log_level"`
	Debug       bool   `mapstructure:"debug"`
	EnablePprof *bool  `mapstructure:"enable_pprof"`
}

type TimerSchedulerLoopConfig struct {
	PollIntervalMillis     int `mapstructure:"poll_interval_millis"`
	BatchSize              int `mapstructure:"batch_size"`
	DispatchLeaseSeconds   int `mapstructure:"dispatch_lease_seconds"`
	ExecutionLeaseSeconds  int `mapstructure:"execution_lease_seconds"`
	QueueAckTimeoutSeconds int `mapstructure:"queue_ack_timeout_seconds"`
	MaxDispatchAttempts    int `mapstructure:"max_dispatch_attempts"`
	MaxHeartbeatMisses     int `mapstructure:"max_heartbeat_misses"`
	MaxOutboxAttempts      int `mapstructure:"max_outbox_attempts"`
	PayloadLimitBytes      int `mapstructure:"payload_limit_bytes"`
}

func (c *TimerSchedulerConfig) GetPort() int {
	if c == nil || c.Server.Port <= 0 {
		return 9098
	}
	return c.Server.Port
}

func (c *TimerSchedulerConfig) GetListenHost() string {
	if c == nil {
		return normalizeListenHost("")
	}
	return normalizeListenHost(c.Server.ListenHost)
}

func (c *TimerSchedulerConfig) GetLogLevel() string {
	if c == nil || c.Server.LogLevel == "" {
		return "info"
	}
	return c.Server.LogLevel
}

func (c *TimerSchedulerConfig) IsDebug() bool {
	return c != nil && c.Server.Debug
}

func (c *TimerSchedulerConfig) IsPprofEnabled() bool {
	if c == nil {
		return true
	}
	return boolConfigValue(c.Server.EnablePprof, true)
}

func (c *TimerSchedulerConfig) GetDB() DBConfig {
	if c == nil || c.DB.Type == "" {
		return DBConfig{Type: "sqlite", Name: "data/timer-scheduler.db"}
	}
	return c.DB
}

func (c *TimerSchedulerConfig) GetSchedulerPollInterval() time.Duration {
	ms := 1000
	if c != nil && c.Scheduler.PollIntervalMillis > 0 {
		ms = c.Scheduler.PollIntervalMillis
	}
	return time.Duration(ms) * time.Millisecond
}

func (c *TimerSchedulerConfig) GetSchedulerBatchSize() int {
	if c == nil || c.Scheduler.BatchSize <= 0 {
		return 50
	}
	return c.Scheduler.BatchSize
}

func (c *TimerSchedulerConfig) GetDispatchLeaseDuration() time.Duration {
	if c == nil || c.Scheduler.DispatchLeaseSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.Scheduler.DispatchLeaseSeconds) * time.Second
}

func (c *TimerSchedulerConfig) GetExecutionLeaseDuration() time.Duration {
	if c == nil || c.Scheduler.ExecutionLeaseSeconds <= 0 {
		return 3 * time.Minute
	}
	return time.Duration(c.Scheduler.ExecutionLeaseSeconds) * time.Second
}

func (c *TimerSchedulerConfig) GetQueueAckTimeout() time.Duration {
	if c == nil || c.Scheduler.QueueAckTimeoutSeconds <= 0 {
		return 2 * time.Minute
	}
	return time.Duration(c.Scheduler.QueueAckTimeoutSeconds) * time.Second
}

func (c *TimerSchedulerConfig) GetMaxDispatchAttempts() int {
	if c == nil || c.Scheduler.MaxDispatchAttempts <= 0 {
		return 3
	}
	return c.Scheduler.MaxDispatchAttempts
}

func (c *TimerSchedulerConfig) GetMaxHeartbeatMisses() int {
	if c == nil || c.Scheduler.MaxHeartbeatMisses <= 0 {
		return 3
	}
	return c.Scheduler.MaxHeartbeatMisses
}

func (c *TimerSchedulerConfig) GetMaxOutboxAttempts() int {
	if c == nil || c.Scheduler.MaxOutboxAttempts <= 0 {
		return 8
	}
	return c.Scheduler.MaxOutboxAttempts
}

func (c *TimerSchedulerConfig) GetPayloadLimitBytes() int {
	if c == nil || c.Scheduler.PayloadLimitBytes <= 0 {
		return 256 * 1024
	}
	return c.Scheduler.PayloadLimitBytes
}
