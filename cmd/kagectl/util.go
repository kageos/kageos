package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mysqlStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, mysqlStringLiteral(value))
	}
	return strings.Join(quoted, ", ")
}

func mysqlStringLiteral(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "'", "''")
	return "'" + replacer.Replace(value) + "'"
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func natsURLForMain(cfg Config) string {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		return cfg.NATS.URL
	}
	host, port := natsHostPortForMain(cfg)
	return buildNATSURL(cfg, host, port)
}

func natsURLForSDK(cfg Config) string {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		return cfg.NATS.URL
	}
	if cfg.NATS.Mode == "bundled" {
		return buildNATSURL(cfg, "127.0.0.1", 4222)
	}
	return buildNATSURL(cfg, cfg.NATS.Host, cfg.NATS.Port)
}

func buildNATSURL(cfg Config, host string, port int) string {
	userInfo := ""
	if cfg.NATS.AuthEnabled {
		userInfo = url.UserPassword(cfg.NATS.User, cfg.NATS.Password).String() + "@"
	}
	return fmt.Sprintf("nats://%s%s", userInfo, net.JoinHostPort(host, strconv.Itoa(port)))
}

func natsHostPortForMain(cfg Config) (string, int) {
	host := cfg.NATS.Host
	port := cfg.NATS.Port
	if cfg.NATS.Mode == "bundled" {
		host = "127.0.0.1"
		port = 4222
	}
	return host, port
}

func natsHostPort(cfg Config) (string, int) {
	if cfg.NATS.Mode == "external" && cfg.NATS.URL != "" {
		parsed, err := url.Parse(cfg.NATS.URL)
		if err == nil && parsed.Host != "" {
			host, port, err := splitHostPortDefault(parsed.Host, 4222)
			if err == nil {
				return host, port
			}
		}
	}
	host := cfg.NATS.Host
	port := cfg.NATS.Port
	if cfg.NATS.Mode == "bundled" {
		host = "127.0.0.1"
		port = 4222
	}
	return host, port
}

func splitHostPortDefault(value string, defaultPort int) (string, int, error) {
	host, portStr, err := net.SplitHostPort(value)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			return value, defaultPort, nil
		}
		return "", 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
