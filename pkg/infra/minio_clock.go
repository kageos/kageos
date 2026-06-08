package infra

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const MinIOClockSkewThreshold = 15 * time.Minute

type MinIOClockSkewError struct {
	URL        string
	LocalTime  time.Time
	ServerTime time.Time
	Skew       time.Duration
	MaxSkew    time.Duration
}

func (e *MinIOClockSkewError) Error() string {
	return fmt.Sprintf(
		"MinIO 时间同步错误: 客户端与 MinIO 服务器时间差 %s，超过允许阈值 %s。当前系统时间: %s，MinIO Date: %s。请同步宿主机和容器运行时 VM 的时间；macOS 可执行 `sudo sntp -sS time.apple.com`，Podman 本地开发可执行 `podman machine stop && podman machine start`，Linux 可执行 `sudo timedatectl set-ntp true`",
		e.Skew.Round(time.Second),
		e.MaxSkew.Round(time.Second),
		e.LocalTime.Format("2006-01-02 15:04:05 MST"),
		e.ServerTime.Format("2006-01-02 15:04:05 MST"),
	)
}

func IsMinIOClockSkewError(err error) bool {
	var skewErr *MinIOClockSkewError
	return errors.As(err, &skewErr)
}

func MinIOHealthURL(endpoint string, useSSL bool) string {
	endpoint = strings.TrimSpace(endpoint)
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return strings.TrimRight(endpoint, "/") + "/minio/health/ready"
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/minio/health/ready", scheme, endpoint)
}

func CheckMinIOClockSkew(ctx context.Context, rawURL string, maxSkew time.Duration, now func() time.Time) error {
	if maxSkew <= 0 {
		maxSkew = MinIOClockSkewThreshold
	}
	if now == nil {
		now = time.Now
	}

	serverTime, sourceURL, err := fetchHTTPDate(ctx, rawURL)
	if err != nil {
		return err
	}

	localTime := now().UTC()
	skew := localTime.Sub(serverTime.UTC())
	if skew < 0 {
		skew = -skew
	}
	if skew > maxSkew {
		return &MinIOClockSkewError{
			URL:        sourceURL,
			LocalTime:  localTime,
			ServerTime: serverTime.UTC(),
			Skew:       skew,
			MaxSkew:    maxSkew,
		}
	}
	return nil
}

func fetchHTTPDate(ctx context.Context, rawURL string) (time.Time, string, error) {
	serverTime, err := fetchHTTPDateOnce(ctx, rawURL)
	if err == nil {
		return serverTime, rawURL, nil
	}
	if !errors.Is(err, errMissingHTTPDate) {
		return time.Time{}, "", err
	}

	fallbackURL, ok := rootURL(rawURL)
	if !ok || fallbackURL == rawURL {
		return time.Time{}, "", err
	}
	serverTime, fallbackErr := fetchHTTPDateOnce(ctx, fallbackURL)
	if fallbackErr != nil {
		return time.Time{}, "", fmt.Errorf("%w; fallback %s failed: %v", err, fallbackURL, fallbackErr)
	}
	return serverTime, fallbackURL, nil
}

var errMissingHTTPDate = errors.New("missing HTTP Date header")

func fetchHTTPDateOnce(ctx context.Context, rawURL string) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return time.Time{}, err
	}
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	dateHeader := strings.TrimSpace(resp.Header.Get("Date"))
	if dateHeader == "" {
		return time.Time{}, fmt.Errorf("%w: %s", errMissingHTTPDate, rawURL)
	}

	serverTime, err := http.ParseTime(dateHeader)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse MinIO Date header %q: %w", dateHeader, err)
	}
	return serverTime, nil
}

func rootURL(rawURL string) (string, bool) {
	parsed, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil || parsed.URL == nil {
		return "", false
	}
	u := *parsed.URL
	u.Path = "/"
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), true
}
