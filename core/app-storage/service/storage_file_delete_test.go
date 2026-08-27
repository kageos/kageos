package service

import (
	"context"
	"io"
	"slices"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-storage/model"
	"github.com/kageos/kageos/core/app-storage/repository"
	storagepkg "github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type deleteTestStorage struct {
	deleted []string
}

func (s *deleteTestStorage) GetCDNDomain() string            { return "" }
func (s *deleteTestStorage) GetUploadEndpoint(string) string { return "" }
func (s *deleteTestStorage) GenerateUploadCredentials(context.Context, string, string, string, time.Duration, string) (*storagepkg.UploadCredentials, error) {
	return nil, nil
}
func (s *deleteTestStorage) GenerateDownloadURLs(context.Context, string, string, time.Duration, map[string]string) (string, string, error) {
	return "/download", "http://storage/download", nil
}
func (s *deleteTestStorage) DeleteObject(_ context.Context, bucket, key string) error {
	s.deleted = append(s.deleted, bucket+"/"+key)
	return nil
}
func (s *deleteTestStorage) GetObjectInfo(_ context.Context, _ string, key string) (*storagepkg.ObjectInfo, error) {
	return &storagepkg.ObjectInfo{Key: key, Size: 42}, nil
}
func (s *deleteTestStorage) ListObjects(context.Context, string, string, bool) ([]storagepkg.ObjectInfo, error) {
	return nil, nil
}
func (s *deleteTestStorage) EnsureBucket(context.Context, string, string) error { return nil }
func (s *deleteTestStorage) UploadObject(context.Context, string, string, io.Reader, int64, string) error {
	return nil
}
func (s *deleteTestStorage) DownloadObject(context.Context, string, string) (io.ReadCloser, error) {
	return nil, nil
}

func newDeleteTestService(t *testing.T) (*StorageService, *repository.FileRepository, *deleteTestStorage) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.FileUpload{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewFileRepository(db)
	backend := &deleteTestStorage{}
	cfg := &config.AppStorageConfig{}
	cfg.Storage.MinIO.DefaultBucket = "kageos"
	cfg.Audit.UploadTracking.Enabled = true
	return NewStorageService(backend, cfg, repo), repo, backend
}

func createDeleteTestRecord(t *testing.T, repo *repository.FileRepository, record *model.FileUpload) {
	t.Helper()
	if err := repo.CreateUploadRecord(context.Background(), record); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteOwnedFileDeletesObjectAndThumbnailAndKeepsTombstones(t *testing.T) {
	service, repo, backend := newDeleteTestService(t)
	createDeleteTestRecord(t, repo, &model.FileUpload{
		Bucket: "kageos", FileKey: "alice/tools/result.xlsx", Router: "alice/tools/run.form",
		FileName: "result.xlsx", FileSize: 2048, Username: "alice", Tenant: "alice", Status: "completed",
		ThumbnailRef: "kageos/alice/tools/result.xlsx.thumb.webp",
	})
	createDeleteTestRecord(t, repo, &model.FileUpload{
		Bucket: "kageos", FileKey: "alice/tools/result.xlsx.thumb.webp", Router: "alice/tools/run.form",
		FileName: "result.thumb.webp", FileSize: 256, Username: "alice", Tenant: "alice", Status: "completed",
	})

	result, err := service.DeleteOwnedFile(context.Background(), "kageos/alice/tools/result.xlsx", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "deleted" || result.ReleasedBytes != 2304 || result.DeletedAt == 0 {
		t.Fatalf("unexpected delete result: %#v", result)
	}
	wantDeleted := []string{"kageos/alice/tools/result.xlsx.thumb.webp", "kageos/alice/tools/result.xlsx"}
	if !slices.Equal(backend.deleted, wantDeleted) {
		t.Fatalf("deleted objects = %#v, want %#v", backend.deleted, wantDeleted)
	}

	for _, key := range []string{"alice/tools/result.xlsx", "alice/tools/result.xlsx.thumb.webp"} {
		record, err := repo.GetUploadRecordByBucketKey(context.Background(), "kageos", key)
		if err != nil {
			t.Fatal(err)
		}
		if record.Status != "deleted" || record.DeletedAt == nil || record.DeletedBy != "alice" {
			t.Fatalf("record %s did not keep a complete tombstone: %#v", key, record)
		}
	}
}

func TestDeleteOwnedFileRejectsAnotherUser(t *testing.T) {
	service, repo, backend := newDeleteTestService(t)
	createDeleteTestRecord(t, repo, &model.FileUpload{
		Bucket: "kageos", FileKey: "alice/private.pdf", Router: "alice/tools/run.form",
		FileName: "private.pdf", FileSize: 1024, Username: "alice", Tenant: "alice", Status: "completed",
	})

	if _, err := service.DeleteOwnedFile(context.Background(), "kageos/alice/private.pdf", "bob"); err == nil {
		t.Fatal("expected ownership rejection")
	}
	if len(backend.deleted) != 0 {
		t.Fatalf("unauthorized delete reached object storage: %#v", backend.deleted)
	}
	record, err := repo.GetUploadRecordByBucketKey(context.Background(), "kageos", "alice/private.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != "completed" {
		t.Fatalf("unauthorized delete changed status to %q", record.Status)
	}
}

func TestResolveDeletedFileReturnsTombstoneWithoutURL(t *testing.T) {
	service, repo, _ := newDeleteTestService(t)
	deletedAt := time.Now().Add(-time.Minute)
	createDeleteTestRecord(t, repo, &model.FileUpload{
		Bucket: "kageos", FileKey: "alice/old.csv", Router: "alice/tools/run.form",
		FileName: "old.csv", FileSize: 4096, Username: "alice", Tenant: "alice", Status: "deleted",
		DeletedAt: &deletedAt, DeletedBy: "alice",
	})

	ctx := contextx.WithRequestUser(context.Background(), "alice")
	files, err := service.ResolveFileRefs(ctx, []string{"kageos/alice/old.csv"}, "browser")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("resolved file count = %d", len(files))
	}
	file := files[0]
	if file.Status != "deleted" || file.DownloadURL != "" || file.CanDelete || file.DeletedAt == 0 || file.DeletedBy != "alice" {
		t.Fatalf("unexpected tombstone: %#v", file)
	}
}
