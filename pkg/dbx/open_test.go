package dbx

import (
	"errors"
	"testing"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
	gormlogger "gorm.io/gorm/logger"
)

func TestIsRetryableMySQLStartupError(t *testing.T) {
	t.Parallel()

	retryable := []error{
		errors.New("driver: bad connection"),
		errors.New("dial tcp 127.0.0.1:3306: connect: connection refused"),
		errors.New("Error 1049 (42000): Unknown database 'app-storage'"),
	}
	for _, err := range retryable {
		if !isRetryableMySQLStartupError(err) {
			t.Fatalf("expected retryable error: %v", err)
		}
	}

	if isRetryableMySQLStartupError(errors.New("Error 1045 (28000): Access denied for user 'root'")) {
		t.Fatal("access denied should not be retried")
	}
}

func TestWithMySQLConnectRetry(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := withMySQLConnectRetry(OpenOptions{
		ConnectRetryTimeout:  100 * time.Millisecond,
		ConnectRetryInterval: time.Millisecond,
	}, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("driver: bad connection")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestParseGORMLogLevel(t *testing.T) {
	t.Parallel()

	cases := map[string]gormlogger.LogLevel{
		"":        gormlogger.Warn,
		"silent":  gormlogger.Silent,
		"error":   gormlogger.Error,
		"warn":    gormlogger.Warn,
		"warning": gormlogger.Warn,
		"info":    gormlogger.Info,
		"debug":   gormlogger.Info,
		"nope":    gormlogger.Warn,
	}
	for input, want := range cases {
		if got := parseGORMLogLevel(input); got != want {
			t.Fatalf("parseGORMLogLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestApplyDBLogOptions(t *testing.T) {
	t.Parallel()

	got := applyDBLogOptions(appconfig.DBConfig{
		LogLevel:      "silent",
		SlowThreshold: 750,
	}, OpenOptions{})
	if got.LogMode != gormlogger.Silent {
		t.Fatalf("LogMode = %v, want %v", got.LogMode, gormlogger.Silent)
	}
	if got.SlowThreshold != 750*time.Millisecond {
		t.Fatalf("SlowThreshold = %v, want 750ms", got.SlowThreshold)
	}

	overridden := applyDBLogOptions(appconfig.DBConfig{
		LogLevel:      "info",
		SlowThreshold: 750,
	}, OpenOptions{
		LogMode:       gormlogger.Error,
		SlowThreshold: time.Second,
	})
	if overridden.LogMode != gormlogger.Error {
		t.Fatalf("LogMode override = %v, want %v", overridden.LogMode, gormlogger.Error)
	}
	if overridden.SlowThreshold != time.Second {
		t.Fatalf("SlowThreshold override = %v, want 1s", overridden.SlowThreshold)
	}
}
