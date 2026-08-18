package main

import (
	"testing"
)

func TestLoadPublishConfigDefaultsAndPaths(t *testing.T) {
	values := map[string]string{
		"KAGEOS_WEB_S3_ENDPOINT":          "https://example.r2.cloudflarestorage.com",
		"KAGEOS_WEB_S3_BUCKET":            "static",
		"KAGEOS_WEB_CDN_URL":              "https://static.example.com/",
		"KAGEOS_WEB_ASSETS_RELEASE":       "sha256-abcd",
		"KAGEOS_WEB_S3_ACCESS_KEY_ID":     "access",
		"KAGEOS_WEB_S3_SECRET_ACCESS_KEY": "secret",
	}
	cfg, err := loadPublishConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourceDir != "/var/www/web/dist/assets" {
		t.Fatalf("SourceDir = %q", cfg.SourceDir)
	}
	if cfg.Prefix != "kageos/web" {
		t.Fatalf("Prefix = %q", cfg.Prefix)
	}
	if got := assetObjectName(cfg, "nested/app.js"); got != "kageos/web/releases/sha256-abcd/assets/nested/app.js" {
		t.Fatalf("assetObjectName = %q", got)
	}
	if cfg.PublicURL != "https://static.example.com" {
		t.Fatalf("PublicURL = %q", cfg.PublicURL)
	}
}

func TestLoadPublishConfigRejectsUnsafeValues(t *testing.T) {
	base := map[string]string{
		"KAGEOS_WEB_S3_ENDPOINT":          "https://s3.example.com",
		"KAGEOS_WEB_S3_BUCKET":            "static",
		"KAGEOS_WEB_CDN_URL":              "https://static.example.com",
		"KAGEOS_WEB_ASSETS_RELEASE":       "release-1",
		"KAGEOS_WEB_S3_ACCESS_KEY_ID":     "access",
		"KAGEOS_WEB_S3_SECRET_ACCESS_KEY": "secret",
	}
	for name, value := range map[string]string{
		"KAGEOS_WEB_S3_ENDPOINT":    "https://s3.example.com/path",
		"KAGEOS_WEB_CDN_URL":        "javascript:alert(1)",
		"KAGEOS_WEB_ASSETS_RELEASE": "../release",
		"KAGEOS_WEB_S3_PREFIX":      "kageos/../web",
	} {
		t.Run(name, func(t *testing.T) {
			values := make(map[string]string, len(base)+1)
			for key, original := range base {
				values[key] = original
			}
			values[name] = value
			if _, err := loadPublishConfig(func(key string) string { return values[key] }); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestContentTypeFor(t *testing.T) {
	for filePath, want := range map[string]string{
		"app.js":     "text/javascript; charset=utf-8",
		"app.css":    "text/css; charset=utf-8",
		"font.woff2": "font/woff2",
		"data.bin":   "application/octet-stream",
	} {
		if got := contentTypeFor(filePath); got != want {
			t.Fatalf("contentTypeFor(%q) = %q, want %q", filePath, got, want)
		}
	}
}
