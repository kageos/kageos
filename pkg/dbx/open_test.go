package dbx

import (
	"errors"
	"testing"
	"time"
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
