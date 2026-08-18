package main

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const immutableCacheControl = "public, max-age=31536000, immutable"

var releasePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type publishConfig struct {
	SourceDir    string
	Endpoint     string
	Bucket       string
	Prefix       string
	Release      string
	PublicURL    string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Region       string
}

func main() {
	cfg, err := loadPublishConfig(os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := publish(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: publish web assets: %v\n", err)
		os.Exit(1)
	}
}

func loadPublishConfig(getenv func(string) string) (publishConfig, error) {
	cfg := publishConfig{
		SourceDir:    strings.TrimSpace(getenv("KAGEOS_WEB_ASSETS_SOURCE_DIR")),
		Endpoint:     strings.TrimSpace(getenv("KAGEOS_WEB_S3_ENDPOINT")),
		Bucket:       strings.TrimSpace(getenv("KAGEOS_WEB_S3_BUCKET")),
		Prefix:       strings.Trim(strings.TrimSpace(getenv("KAGEOS_WEB_S3_PREFIX")), "/"),
		Release:      strings.TrimSpace(getenv("KAGEOS_WEB_ASSETS_RELEASE")),
		PublicURL:    strings.TrimRight(strings.TrimSpace(getenv("KAGEOS_WEB_CDN_URL")), "/"),
		AccessKey:    strings.TrimSpace(getenv("KAGEOS_WEB_S3_ACCESS_KEY_ID")),
		SecretKey:    strings.TrimSpace(getenv("KAGEOS_WEB_S3_SECRET_ACCESS_KEY")),
		SessionToken: strings.TrimSpace(getenv("KAGEOS_WEB_S3_SESSION_TOKEN")),
		Region:       strings.TrimSpace(getenv("KAGEOS_WEB_S3_REGION")),
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = "/var/www/web/dist/assets"
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "kageos/web"
	}

	for name, value := range map[string]string{
		"KAGEOS_WEB_S3_ENDPOINT":          cfg.Endpoint,
		"KAGEOS_WEB_S3_BUCKET":            cfg.Bucket,
		"KAGEOS_WEB_CDN_URL":              cfg.PublicURL,
		"KAGEOS_WEB_ASSETS_RELEASE":       cfg.Release,
		"KAGEOS_WEB_S3_ACCESS_KEY_ID":     cfg.AccessKey,
		"KAGEOS_WEB_S3_SECRET_ACCESS_KEY": cfg.SecretKey,
	} {
		if value == "" {
			return publishConfig{}, fmt.Errorf("%s is required", name)
		}
	}
	if !releasePattern.MatchString(cfg.Release) {
		return publishConfig{}, fmt.Errorf("KAGEOS_WEB_ASSETS_RELEASE contains unsupported characters")
	}
	if err := validatePrefix(cfg.Prefix); err != nil {
		return publishConfig{}, err
	}
	if _, _, err := parseEndpoint(cfg.Endpoint); err != nil {
		return publishConfig{}, err
	}
	if err := validatePublicURL(cfg.PublicURL); err != nil {
		return publishConfig{}, err
	}
	return cfg, nil
}

func validatePrefix(prefix string) error {
	for _, part := range strings.Split(prefix, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("KAGEOS_WEB_S3_PREFIX must be a clean object prefix")
		}
	}
	return nil
}

func parseEndpoint(raw string) (string, bool, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", false, fmt.Errorf("KAGEOS_WEB_S3_ENDPOINT must be an absolute http(s) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false, fmt.Errorf("KAGEOS_WEB_S3_ENDPOINT must use http or https")
	}
	if parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false, fmt.Errorf("KAGEOS_WEB_S3_ENDPOINT must not contain a path, query, or fragment")
	}
	return parsed.Host, parsed.Scheme == "https", nil
}

func validatePublicURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("KAGEOS_WEB_CDN_URL must be an absolute http(s) URL")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("KAGEOS_WEB_CDN_URL must not contain a query or fragment")
	}
	return nil
}

func objectBase(cfg publishConfig) string {
	return path.Join(cfg.Prefix, "releases", cfg.Release)
}

func assetObjectName(cfg publishConfig, relativePath string) string {
	return path.Join(objectBase(cfg), "assets", filepath.ToSlash(relativePath))
}

func publish(ctx context.Context, cfg publishConfig) error {
	endpoint, secure, err := parseEndpoint(cfg.Endpoint)
	if err != nil {
		return err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, cfg.SessionToken),
		Secure: secure,
		Region: cfg.Region,
	})
	if err != nil {
		return fmt.Errorf("create S3 client: %w", err)
	}

	uploaded := 0
	err = filepath.WalkDir(cfg.SourceDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(cfg.SourceDir, filePath)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		objectName := assetObjectName(cfg, relativePath)
		_, err = client.FPutObject(ctx, cfg.Bucket, objectName, filePath, minio.PutObjectOptions{
			ContentType:  contentTypeFor(filePath),
			CacheControl: immutableCacheControl,
		})
		if err != nil {
			return fmt.Errorf("upload %s (%d bytes): %w", relativePath, info.Size(), err)
		}
		uploaded++
		return nil
	})
	if err != nil {
		return err
	}
	if uploaded == 0 {
		return fmt.Errorf("no files found under %s", cfg.SourceDir)
	}

	assetBase := cfg.PublicURL + "/" + objectBase(cfg)
	fmt.Printf("Published %d immutable web assets\n", uploaded)
	fmt.Printf("Web asset base: %s\n", assetBase)
	return nil
}

func contentTypeFor(filePath string) string {
	extension := strings.ToLower(filepath.Ext(filePath))
	switch extension {
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json", ".map":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	}
	if value := mime.TypeByExtension(extension); value != "" {
		return value
	}
	return "application/octet-stream"
}
