package infra

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

var (
	syncPodmanMachineClock = syncPodmanMachineClockCommand
	podmanCommandAvailable = defaultPodmanCommandAvailable
)

func CheckMinIOClockSkewWithDevPodmanRepair(ctx context.Context, rawURL string, maxSkew time.Duration, now func() time.Time) error {
	err := CheckMinIOClockSkew(ctx, rawURL, maxSkew, now)
	if err == nil || !IsMinIOClockSkewError(err) || !shouldAttemptDevPodmanClockRepair(rawURL) {
		return err
	}

	logger.Warnf(ctx, "[MinIOClock] 检测到本地 Podman MinIO 时间偏移，尝试同步 Podman VM 时间...")
	if repairErr := syncPodmanMachineClock(ctx); repairErr != nil {
		return fmt.Errorf("%w；自动同步 Podman VM 时间失败: %v", err, repairErr)
	}

	logger.Infof(ctx, "[MinIOClock] Podman VM 时间同步完成，重新检查 MinIO 时间")
	if retryErr := CheckMinIOClockSkew(ctx, rawURL, maxSkew, now); retryErr != nil {
		return fmt.Errorf("自动同步 Podman VM 时间后 MinIO 时间检查仍失败: %w", retryErr)
	}
	return nil
}

func shouldAttemptDevPodmanClockRepair(rawURL string) bool {
	if !config.IsDevMode() || !isLocalMinIOClockURL(rawURL) {
		return false
	}

	switch config.GetDevEngine() {
	case config.ConfigDevEnginePodman:
		return podmanCommandAvailable()
	case config.ConfigDevEngineAuto:
		return podmanCommandAvailable()
	default:
		return false
	}
}

func isLocalMinIOClockURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func defaultPodmanCommandAvailable() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}

func syncPodmanMachineClockCommand(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	script := strings.Join([]string{
		`if ! command -v chronyc >/dev/null 2>&1; then echo "chronyc not found in Podman machine" >&2; exit 127; fi`,
		`sudo chronyc -a 'burst 4/4' >/dev/null 2>&1 || true`,
		`sudo chronyc -a makestep`,
	}, "; ")

	cmd := exec.CommandContext(ctx, "podman", "machine", "ssh", script)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}
