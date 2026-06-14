package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/infra"
)

func runLayerChecksJSON(name string, checks []layerCheck) error {
	report := runLayerChecksReport(name, checks)
	if err := writeJSON(report); err != nil {
		return err
	}
	if !report.OK {
		return fmt.Errorf("%s failed with %d issue(s)", name, report.Failures)
	}
	return nil
}

func waitLayerChecks(name string, checks []layerCheck, timeout, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	started := time.Now()
	deadline := started.Add(timeout)
	attempt := 0
	var last checkReport
	for {
		attempt++
		last = runLayerChecksReport(name, checks)
		if last.OK {
			fmt.Printf("%s passed after %s\n", name, time.Since(started).Round(time.Second))
			return nil
		}
		if time.Now().After(deadline) {
			fmt.Printf("\n%s final failure report\n", name)
			printLayerCheckReport(last)
			return fmt.Errorf("%s did not pass within %s (%d issue(s) remaining)", name, timeout, last.Failures)
		}
		remaining := time.Until(deadline)
		sleep := interval
		if remaining < sleep {
			sleep = remaining
		}
		fmt.Printf("  waiting for %s: %d issue(s) remaining (attempt %d, next check in %s)\n", name, last.Failures, attempt, sleep.Round(time.Second))
		time.Sleep(sleep)
	}
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func doctorLayerChecks(rt RuntimeConfig) []layerCheck {
	checks := []layerCheck{
		{Layer: layerControl, Name: "config validation", Target: rt.Paths.ConfigPath, Fn: func() error { return validateConfig(rt) }},
		{Layer: layerControl, Name: "compose command", Target: "podman compose / docker compose", Fn: checkComposeCommand},
		{Layer: layerControl, Name: "compose runtime", Target: "compose ls", Fn: checkComposeRuntime},
		{Layer: layerControl, Name: "linux host", Target: runtime.GOOS, Fn: checkLinuxHost},
		{Layer: layerInfra, Name: "storage root parent", Target: rt.Storage.Root, Fn: func() error { return checkStorageRoot(rt.Storage.Root) }},
	}
	checks = appendExternalDependencyChecks(checks, rt)
	return checks
}

func appendExternalDependencyChecks(checks []layerCheck, rt RuntimeConfig) []layerCheck {
	if rt.MySQL.Mode == "external" {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "external mysql",
			Target: tcpTarget(rt.MySQL.Host, rt.MySQL.Port),
			Fn:     func() error { return checkTCP("mysql", rt.MySQL.Host, rt.MySQL.Port) },
		})
	}
	if rt.NATS.Mode == "external" {
		host, port := natsHostPort(rt.Config)
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "external nats",
			Target: tcpTarget(host, port),
			Fn:     func() error { return checkTCP("nats", host, port) },
		})
	}
	if rt.MinIO.Mode == "external" {
		host, port, err := splitHostPortDefault(rt.MinIO.Endpoint, 9000)
		check := layerCheck{
			Layer:  layerInfra,
			Name:   "external minio",
			Target: rt.MinIO.Endpoint,
		}
		if err != nil {
			check.Fn = func() error { return err }
		} else {
			check.Target = tcpTarget(host, port)
			check.Fn = func() error { return checkTCP("minio", host, port) }
		}
		checks = append(checks, check)
		if err == nil {
			checks = append(checks, layerCheck{
				Layer:  layerInfra,
				Name:   "external minio clock",
				Target: minIOClockCheckURL(host, port, rt.MinIO.UseSSL),
				Fn:     func() error { return checkMinIOClock(host, port, rt.MinIO.UseSSL) },
			})
		}
	}
	return checks
}

func startupDependencyChecks(rt RuntimeConfig) []layerCheck {
	checks := make([]layerCheck, 0, 3)
	if rt.IncludeMySQL {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "mysql initialized",
			Target: "compose exec mysql SELECT required databases",
			Fn:     func() error { return checkBundledMySQLInitialized(rt) },
		})
	} else {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "mysql tcp",
			Target: rt.MySQLAddress,
			Fn:     func() error { return checkTCP("mysql", rt.MySQLHostForMain, rt.MySQLPortForMain) },
		})
	}
	if rt.IncludeNATS {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "nats tcp",
			Target: tcpTarget(rt.NATSHostForMain, rt.NATSPortForMain),
			Fn:     func() error { return checkTCP("nats", rt.NATSHostForMain, rt.NATSPortForMain) },
		})
	}
	if rt.IncludeMinIO {
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "minio tcp",
			Target: tcpTarget(rt.MinIOHostForMain, rt.MinIOPortForMain),
			Fn:     func() error { return checkTCP("minio", rt.MinIOHostForMain, rt.MinIOPortForMain) },
		})
		checks = append(checks, layerCheck{
			Layer:  layerInfra,
			Name:   "minio clock",
			Target: minIOClockCheckURL(rt.MinIOHostForMain, rt.MinIOPortForMain, rt.MinIO.UseSSL),
			Fn:     func() error { return checkMinIOClock(rt.MinIOHostForMain, rt.MinIOPortForMain, rt.MinIO.UseSSL) },
		})
	}
	return checks
}

func verifyLayerChecks(rt RuntimeConfig) []layerCheck {
	checks := []layerCheck{
		{Layer: layerControl, Name: "config validation", Target: rt.Paths.ConfigPath, Fn: func() error { return validateConfig(rt) }},
		{Layer: layerControl, Name: "rendered compose", Target: rt.ComposeConfigPath, Fn: func() error { return requireGeneratedCompose(rt.Paths) }},
		{Layer: layerEdge, Name: "nginx http listener", Target: tcpTarget("127.0.0.1", rt.Site.HTTPPort), Fn: func() error { return checkTCP("nginx", "127.0.0.1", rt.Site.HTTPPort) }},
		{Layer: layerEdge, Name: "main edge probe", Target: "compose exec main /app/health/edge.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/edge.sh")
		}},
	}
	checks = append(checks[:2], append(startupDependencyChecks(rt), checks[2:]...)...)
	if rt.Site.TLSMode == "https" || rt.Site.TLSMode == "redirect" {
		checks = append(checks, layerCheck{Layer: layerEdge, Name: "nginx https listener", Target: tcpTarget("127.0.0.1", rt.Site.HTTPSPort), Fn: func() error { return checkTCP("nginx", "127.0.0.1", rt.Site.HTTPSPort) }})
	}
	checks = append(checks,
		layerCheck{Layer: layerPlatform, Name: "api-gateway", Target: "http://127.0.0.1:9090/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9090/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-server", Target: "http://127.0.0.1:9091/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9091/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-storage", Target: "http://127.0.0.1:9092/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9092/health") }},
		layerCheck{Layer: layerPlatform, Name: "agent-server", Target: "http://127.0.0.1:9095/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9095/health") }},
		layerCheck{Layer: layerPlatform, Name: "connector-server", Target: "http://127.0.0.1:9096/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9096/health") }},
		layerCheck{Layer: layerPlatform, Name: "hr-server", Target: "http://127.0.0.1:9097/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9097/health") }},
		layerCheck{Layer: layerPlatform, Name: "timer-scheduler", Target: "http://127.0.0.1:9098/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9098/health") }},
		layerCheck{Layer: layerPlatform, Name: "message-server", Target: "http://127.0.0.1:9099/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9099/health") }},
		layerCheck{Layer: layerPlatform, Name: "main platform probe", Target: "compose exec main /app/health/platform.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/platform.sh")
		}},
		layerCheck{Layer: layerRuntime, Name: "app-runtime", Target: "http://127.0.0.1:9093/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9093/health") }},
		layerCheck{Layer: layerRuntime, Name: "main runtime probe", Target: "compose exec main /app/health/runtime.sh", Fn: func() error {
			return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "/app/health/runtime.sh")
		}},
	)
	checks = appendSDKEndpointChecks(checks, rt)
	return checks
}

func appendSDKEndpointChecks(checks []layerCheck, rt RuntimeConfig) []layerCheck {
	checks = append(checks, layerCheck{
		Layer:  layerApps,
		Name:   "sdk gateway endpoint",
		Target: rt.SDKGatewayURL,
		Fn:     func() error { return requireContains(rt.SDKGatewayURL, "127.0.0.1") },
	})
	if rt.NATS.Mode == "bundled" {
		checks = append(checks, layerCheck{
			Layer:  layerApps,
			Name:   "sdk nats endpoint",
			Target: redactURLCredentials(rt.SDKNATSURL),
			Fn:     func() error { return requireContains(rt.SDKNATSURL, "127.0.0.1") },
		})
	}
	if rt.MinIO.Mode == "bundled" {
		checks = append(checks, layerCheck{
			Layer:  layerApps,
			Name:   "sdk minio endpoint",
			Target: rt.SDKMinIOEndpoint,
			Fn:     func() error { return requireContains(rt.SDKMinIOEndpoint, "127.0.0.1") },
		})
	}
	return checks
}

func requireContains(value, needle string) error {
	if !strings.Contains(value, needle) {
		return fmt.Errorf("expected %q to contain %q", value, needle)
	}
	return nil
}

func checkHTTP(rawURL string) error {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned HTTP %d", rawURL, resp.StatusCode)
	}
	return nil
}

func minIOClockCheckURL(host string, port int, useSSL bool) string {
	return infra.MinIOHealthURL(tcpTarget(host, port), useSSL)
}

func checkMinIOClock(host string, port int, useSSL bool) error {
	return infra.CheckMinIOClockSkewWithDevPodmanRepair(context.Background(), minIOClockCheckURL(host, port, useSSL), infra.MinIOClockSkewThreshold, nil)
}

func checkLinuxHost() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("prod single-node deployment currently requires Linux, current=%s", runtime.GOOS)
	}
	return nil
}

func checkStorageRoot(root string) error {
	if fileExists(root) {
		return nil
	}
	parent := filepath.Dir(root)
	info, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", parent)
	}
	return nil
}

func checkTCP(label, host string, port int) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return fmt.Errorf("%s %s:%d not reachable: %w", label, host, port, err)
	}
	_ = conn.Close()
	return nil
}

func checkBundledMySQLInitialized(rt RuntimeConfig) error {
	databases := requiredMySQLDatabases(rt)
	if len(databases) == 0 {
		return runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "mysql", "mysqladmin", "ping", "-h", "127.0.0.1", "--connect-timeout=3", "-u"+rt.MySQL.User, "-p"+rt.MySQL.Password)
	}

	sql := fmt.Sprintf(
		"SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME IN (%s);",
		mysqlStringList(databases),
	)
	output, err := runComposeOutput(rt.Paths.GeneratedDir, "exec", "-T", "mysql", "mysql", "-h", "127.0.0.1", "--connect-timeout=3", "-u"+rt.MySQL.User, "-p"+rt.MySQL.Password, "-N", "-B", "-e", sql)
	if err != nil {
		return err
	}
	count, err := parseMySQLCountOutput(output)
	if err != nil {
		return err
	}
	if count != len(databases) {
		return fmt.Errorf("mysql initialized databases not ready: got %d/%d (%s)", count, len(databases), strings.Join(databases, ", "))
	}
	return nil
}

func requiredMySQLDatabases(rt RuntimeConfig) []string {
	return uniqueNonEmptyStrings([]string{
		rt.MySQL.AppDatabase,
		rt.MySQL.StorageDatabase,
		rt.MySQL.AgentDatabase,
		rt.MySQL.ConnectorDatabase,
		rt.MySQL.HRDatabase,
		rt.MySQL.TimerDatabase,
		rt.MySQL.MessageDatabase,
	})
}

func parseMySQLCountOutput(output string) (int, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		count, err := strconv.Atoi(line)
		if err == nil {
			return count, nil
		}
	}
	return 0, fmt.Errorf("parse mysql database count from %q: no numeric line found", output)
}
