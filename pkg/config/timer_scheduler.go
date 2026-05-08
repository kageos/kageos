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
	Server    TimerSchedulerServerConfig    `mapstructure:"server"`
	Scheduler TimerSchedulerSchedulerConfig `mapstructure:"scheduler"`
	DB        DBConfig                      `mapstructure:"db"`
}

type TimerSchedulerServerConfig struct {
	Port       int    `mapstructure:"port"`
	ListenHost string `mapstructure:"listen_host"`
	LogLevel   string `mapstructure:"log_level"`
	Debug      bool   `mapstructure:"debug"`
}

type TimerSchedulerSchedulerConfig struct {
	PollIntervalSeconds   int `mapstructure:"poll_interval_seconds"`
	BatchSize             int `mapstructure:"batch_size"`
	DispatchLeaseSeconds  int `mapstructure:"dispatch_lease_seconds"`
	ExecutionLeaseSeconds int `mapstructure:"execution_lease_seconds"`
}

func (c *TimerSchedulerConfig) GetPort() int {
	if c == nil || c.Server.Port <= 0 {
		return 9108
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

func (c *TimerSchedulerConfig) GetSchedulerPollInterval() time.Duration {
	if c == nil || c.Scheduler.PollIntervalSeconds <= 0 {
		return time.Second
	}
	return time.Duration(c.Scheduler.PollIntervalSeconds) * time.Second
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
		return time.Hour
	}
	return time.Duration(c.Scheduler.ExecutionLeaseSeconds) * time.Second
}

func (c *TimerSchedulerConfig) GetDB() DBConfig {
	if c == nil {
		return DBConfig{Name: "timer-scheduler"}
	}
	db := c.DB
	if db.Name == "" {
		db.Name = "timer-scheduler"
	}
	return db
}
