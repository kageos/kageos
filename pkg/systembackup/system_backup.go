package systembackup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/secretvault"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	ConfigFileName = "config.json"
	StateFileName  = "state.json"
	vaultPrefix    = "sealed:backup:"
)

type Config struct {
	Enabled               bool   `json:"enabled"`
	ScheduleTime          string `json:"schedule_time"`
	Endpoint              string `json:"endpoint"`
	Region                string `json:"region"`
	Bucket                string `json:"bucket"`
	Prefix                string `json:"prefix"`
	AccessKeyID           string `json:"access_key_id"`
	SecretAccessKey       string `json:"secret_access_key,omitempty"`
	SecretAccessKeySet    bool   `json:"secret_access_key_set"`
	UseSSL                bool   `json:"use_ssl"`
	ForcePathStyle        bool   `json:"force_path_style"`
	KeepLocal             int    `json:"keep_local"`
	RetentionDays         int    `json:"retention_days"`
	RunNowRequestedAt     string `json:"run_now_requested_at,omitempty"`
	LastRunNowProcessedAt string `json:"last_run_now_processed_at,omitempty"`
}

type Record struct {
	ID           string `json:"id"`
	TriggeredBy  string `json:"triggered_by"`
	Status       string `json:"status"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
	ArchiveName  string `json:"archive_name,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
	Bucket       string `json:"bucket,omitempty"`
	ObjectKey    string `json:"object_key,omitempty"`
	ETag         string `json:"etag,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type State struct {
	AgentLastSeenAt string   `json:"agent_last_seen_at,omitempty"`
	Running         bool     `json:"running"`
	Records         []Record `json:"records"`
}

func DefaultConfig() Config {
	return Config{ScheduleTime: "03:30", Region: "us-east-1", Prefix: "kageos-backups", UseSSL: true, KeepLocal: 2, RetentionDays: 30}
}

func StateDir() string {
	if value := strings.TrimSpace(os.Getenv("KAGEOS_BACKUP_STATE_DIR")); value != "" {
		return filepath.Clean(value)
	}
	return filepath.Join("data", "system-backup")
}

func LoadConfig(dir, encryptionSecret string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("decode backup config: %w", err)
	}
	if cfg.SecretAccessKey != "" {
		vault, err := secretvault.New(encryptionSecret, "system-backup-s3", secretvault.WithPrefix(vaultPrefix))
		if err != nil {
			return cfg, err
		}
		cfg.SecretAccessKey, err = vault.Open(cfg.SecretAccessKey)
		if err != nil {
			return cfg, fmt.Errorf("decrypt backup S3 secret: %w", err)
		}
		cfg.SecretAccessKeySet = cfg.SecretAccessKey != ""
	}
	return normalizeConfig(cfg), nil
}

func SaveConfig(dir, encryptionSecret string, cfg Config) error {
	cfg = normalizeConfig(cfg)
	if err := ValidateConfig(cfg, cfg.Enabled); err != nil {
		return err
	}
	if cfg.SecretAccessKey != "" {
		vault, err := secretvault.New(encryptionSecret, "system-backup-s3", secretvault.WithPrefix(vaultPrefix))
		if err != nil {
			return err
		}
		sealed, err := vault.Seal(cfg.SecretAccessKey)
		if err != nil {
			return err
		}
		cfg.SecretAccessKey = sealed
		cfg.SecretAccessKeySet = true
	}
	return writeJSON(filepath.Join(dir, ConfigFileName), cfg, 0644)
}

func LoadState(dir string) (State, error) {
	var state State
	data, err := os.ReadFile(filepath.Join(dir, StateFileName))
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, fmt.Errorf("decode backup state: %w", err)
	}
	return state, nil
}

func SaveState(dir string, state State) error {
	if len(state.Records) > 50 {
		state.Records = state.Records[:50]
	}
	return writeJSON(filepath.Join(dir, StateFileName), state, 0644)
}

func PublicConfig(cfg Config) Config {
	cfg.SecretAccessKey = ""
	return cfg
}

func ValidateConfig(cfg Config, requireDestination bool) error {
	if _, err := time.Parse("15:04", cfg.ScheduleTime); err != nil {
		return fmt.Errorf("schedule_time must use HH:mm")
	}
	if cfg.KeepLocal < 0 || cfg.KeepLocal > 30 {
		return fmt.Errorf("keep_local must be between 0 and 30")
	}
	if cfg.RetentionDays < 1 || cfg.RetentionDays > 3650 {
		return fmt.Errorf("retention_days must be between 1 and 3650")
	}
	if !requireDestination {
		return nil
	}
	if strings.TrimSpace(cfg.Bucket) == "" || strings.TrimSpace(cfg.Prefix) == "" || strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return fmt.Errorf("bucket, prefix, access_key_id and secret_access_key are required")
	}
	_, _, err := endpointFor(cfg)
	return err
}

func Due(cfg Config, state State, now time.Time) (bool, string) {
	if !cfg.Enabled {
		return false, ""
	}
	if cfg.RunNowRequestedAt != "" && cfg.RunNowRequestedAt != cfg.LastRunNowProcessedAt {
		return true, "manual"
	}
	clock, err := time.Parse("15:04", cfg.ScheduleTime)
	if err != nil {
		return false, ""
	}
	dueAt := time.Date(now.Year(), now.Month(), now.Day(), clock.Hour(), clock.Minute(), 0, 0, now.Location())
	if now.Before(dueAt) {
		return false, ""
	}
	for _, record := range state.Records {
		started, err := time.Parse(time.RFC3339, record.StartedAt)
		if err == nil && started.In(now.Location()).YearDay() == now.YearDay() && started.In(now.Location()).Year() == now.Year() && record.TriggeredBy == "schedule" {
			return false, ""
		}
	}
	return true, "schedule"
}

func TestS3(ctx context.Context, cfg Config) error {
	if err := ValidateConfig(cfg, true); err != nil {
		return err
	}
	client, err := newClient(cfg)
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", cfg.Bucket)
	}
	key := objectKey(cfg.Prefix, ".kageos-connection-test-"+time.Now().UTC().Format("20060102T150405Z"))
	body := strings.NewReader("kageos backup connection test")
	if _, err := client.PutObject(ctx, cfg.Bucket, key, body, body.Size(), minio.PutObjectOptions{ContentType: "text/plain"}); err != nil {
		return fmt.Errorf("upload test object: %w", err)
	}
	defer client.RemoveObject(context.Background(), cfg.Bucket, key, minio.RemoveObjectOptions{})
	if _, err := client.StatObject(ctx, cfg.Bucket, key, minio.StatObjectOptions{}); err != nil {
		return fmt.Errorf("read test object: %w", err)
	}
	if err := client.RemoveObject(ctx, cfg.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete test object: %w", err)
	}
	return nil
}

func Upload(ctx context.Context, cfg Config, archivePath string) (object string, etag string, size int64, checksum string, err error) {
	client, err := newClient(cfg)
	if err != nil {
		return "", "", 0, "", err
	}
	checksum, size, err = fileChecksum(archivePath)
	if err != nil {
		return "", "", 0, "", err
	}
	object = objectKey(cfg.Prefix, time.Now().UTC().Format("2006/01/02"), filepath.Base(archivePath))
	info, err := client.FPutObject(ctx, cfg.Bucket, object, archivePath, minio.PutObjectOptions{
		ContentType:  "application/gzip",
		UserMetadata: map[string]string{"kageos-sha256": checksum},
	})
	if err != nil {
		return "", "", 0, "", fmt.Errorf("upload backup: %w", err)
	}
	remote, err := client.StatObject(ctx, cfg.Bucket, object, minio.StatObjectOptions{})
	if err != nil {
		return "", "", 0, "", fmt.Errorf("verify uploaded backup: %w", err)
	}
	if remote.Size != size {
		return "", "", 0, "", fmt.Errorf("uploaded backup size mismatch: local=%d remote=%d", size, remote.Size)
	}
	return object, info.ETag, size, checksum, nil
}

func PruneLocal(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "kageos-backup-") && strings.HasSuffix(entry.Name(), ".tar.gz") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	for _, path := range paths[keep:] {
		if err := os.Remove(path); err != nil {
			return err
		}
		_ = os.Remove(path + ".sha256")
	}
	return nil
}

func PruneRemote(ctx context.Context, cfg Config, now time.Time) error {
	client, err := newClient(cfg)
	if err != nil {
		return err
	}
	cutoff := now.AddDate(0, 0, -cfg.RetentionDays)
	for item := range client.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Prefix: strings.Trim(cfg.Prefix, "/"), Recursive: true}) {
		if item.Err != nil {
			return item.Err
		}
		name := filepath.Base(item.Key)
		if strings.HasPrefix(name, "kageos-backup-") && strings.HasSuffix(name, ".tar.gz") && !item.LastModified.IsZero() && item.LastModified.Before(cutoff) {
			if err := client.RemoveObject(ctx, cfg.Bucket, item.Key, minio.RemoveObjectOptions{}); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if strings.TrimSpace(cfg.ScheduleTime) == "" {
		cfg.ScheduleTime = defaults.ScheduleTime
	}
	if strings.TrimSpace(cfg.Region) == "" {
		cfg.Region = defaults.Region
	}
	if cfg.KeepLocal == 0 {
		cfg.KeepLocal = defaults.KeepLocal
	}
	if cfg.RetentionDays == 0 {
		cfg.RetentionDays = defaults.RetentionDays
	}
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.Region = strings.TrimSpace(cfg.Region)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	cfg.Prefix = strings.Trim(strings.TrimSpace(cfg.Prefix), "/")
	cfg.AccessKeyID = strings.TrimSpace(cfg.AccessKeyID)
	return cfg
}

func newClient(cfg Config) (*minio.Client, error) {
	endpoint, secure, err := endpointFor(cfg)
	if err != nil {
		return nil, err
	}
	return minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""), Secure: secure, Region: cfg.Region, BucketLookup: bucketLookup(cfg.ForcePathStyle)})
}

func bucketLookup(forcePath bool) minio.BucketLookupType {
	if forcePath {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func endpointFor(cfg Config) (string, bool, error) {
	raw := strings.TrimSpace(cfg.Endpoint)
	if raw == "" {
		if cfg.Region == "" || cfg.Region == "us-east-1" {
			return "s3.amazonaws.com", true, nil
		}
		return "s3." + cfg.Region + ".amazonaws.com", true, nil
	}
	secure := cfg.UseSSL
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return "", false, fmt.Errorf("invalid S3 endpoint")
		}
		if parsed.Path != "" && parsed.Path != "/" {
			return "", false, fmt.Errorf("S3 endpoint must not contain a path")
		}
		raw = parsed.Host
		secure = parsed.Scheme == "https"
	}
	if strings.ContainsAny(raw, "/?#") {
		return "", false, fmt.Errorf("invalid S3 endpoint")
	}
	return raw, secure, nil
}

func objectKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.Trim(part, "/"); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return strings.Join(cleaned, "/")
}

func fileChecksum(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func writeJSON(path string, value any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Chmod(mode)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
