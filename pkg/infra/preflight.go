package infra

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
)

// Preflight performs a lightweight infrastructure readiness check.
//
// Dev infrastructure is started by deploy/dev/scripts/infra.sh.
// Prod infrastructure is rendered and started by kagectl.
// The application process should not create or mutate middleware containers.
func Preflight(ctx context.Context) error {
	start := time.Now()
	logger.Infof(ctx, "[Preflight] ========== 基础设施预检开始 ==========")
	waitForMySQLTCP(ctx, mysqlPreflightAddress(), 30*time.Second)
	logger.Infof(ctx, "[Preflight] ========== 基础设施预检完成 | 总耗时=%s ==========", time.Since(start).Round(time.Millisecond))
	return nil
}

func mysqlPreflightAddress() string {
	host := strings.TrimSpace(os.Getenv("MYSQL_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}

	port := 3306
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "dev") {
		port = 3318
	}
	if rawPort := strings.TrimSpace(os.Getenv("MYSQL_PORT")); rawPort != "" {
		if parsed, err := strconv.Atoi(rawPort); err == nil && parsed > 0 {
			port = parsed
		}
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func waitForMySQLTCP(ctx context.Context, addr string, maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)
	logger.Infof(ctx, "[Preflight] 等待 MySQL TCP %s（最多 %s）...", addr, maxWait)
	for time.Now().Before(deadline) {
		d := net.Dialer{Timeout: 2 * time.Second}
		c, err := d.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = c.Close()
			logger.Infof(ctx, "[Preflight] ✅ MySQL TCP 已连通: %s", addr)
			return
		}
		time.Sleep(time.Second)
	}
	logger.Warnf(ctx, "[Preflight] ⚠ MySQL TCP %s 在 %s 内未连通", addr, maxWait)
}
