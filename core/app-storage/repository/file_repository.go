package repository

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-storage/model"
	"gorm.io/gorm"
)

// FileRepository 文件仓储层
type FileRepository struct {
	db *gorm.DB
}

type SystemFileAssetRow struct {
	model.FileUpload
	DownloadCount  int64        `gorm:"column:download_count"`
	PreviewCount   int64        `gorm:"column:preview_count"`
	LastAccessedAt FlexibleTime `gorm:"column:last_accessed_at"`
}

type FlexibleTime struct {
	Time  time.Time
	Valid bool
}

func (value *FlexibleTime) Scan(source interface{}) error {
	if source == nil {
		value.Valid = false
		return nil
	}
	if parsed, ok := source.(time.Time); ok {
		value.Time, value.Valid = parsed, true
		return nil
	}
	text := fmt.Sprint(source)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			value.Time, value.Valid = parsed, true
			return nil
		}
	}
	return fmt.Errorf("unsupported time value %q", text)
}

func (value FlexibleTime) Value() (driver.Value, error) {
	if !value.Valid {
		return nil, nil
	}
	return value.Time, nil
}

type SystemFileAssetFilter struct {
	RouterPrefix string
	Status       string
	Keyword      string
}

type SystemFileAssetSummary struct {
	ActiveFiles  int64 `gorm:"column:active_files"`
	ActiveBytes  int64 `gorm:"column:active_bytes"`
	DeletedFiles int64 `gorm:"column:deleted_files"`
	FailedFiles  int64 `gorm:"column:failed_files"`
}

type SystemFileAssetDirectory struct {
	Router    string `gorm:"column:router"`
	FileCount int64  `gorm:"column:file_count"`
	SizeBytes int64  `gorm:"column:size_bytes"`
}

// NewFileRepository 创建文件仓储
func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{
		db: db,
	}
}

// CreateUploadRecord 创建上传记录
func (r *FileRepository) CreateUploadRecord(ctx context.Context, record *model.FileUpload) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// UpdateUploadStatus 更新上传状态
func (r *FileRepository) UpdateUploadStatus(ctx context.Context, fileKey string, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("file_key = ?", fileKey).
		Update("status", status).Error
}

func (r *FileRepository) UpdateUploadStatusByBucketKey(ctx context.Context, bucket string, fileKey string, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("bucket = ? AND file_key = ?", bucket, fileKey).
		Update("status", status).Error
}

func (r *FileRepository) UpdateDescriptionByBucketKey(ctx context.Context, bucket string, fileKey string, description string) error {
	result := r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("bucket = ? AND file_key = ?", bucket, fileKey).
		Update("description", description)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetUploadRecord 获取上传记录
func (r *FileRepository) GetUploadRecord(ctx context.Context, fileKey string) (*model.FileUpload, error) {
	var record model.FileUpload
	err := r.db.WithContext(ctx).Where("file_key = ?", fileKey).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *FileRepository) GetUploadRecordByBucketKey(ctx context.Context, bucket string, fileKey string) (*model.FileUpload, error) {
	var record model.FileUpload
	err := r.db.WithContext(ctx).
		Where("bucket = ? AND file_key = ?", bucket, fileKey).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateDeletionState 保留上传元数据，只更新物理文件删除生命周期。
func (r *FileRepository) UpdateDeletionState(ctx context.Context, bucket string, fileKey string, status string, deletedBy string, deletedAt *time.Time, deleteError string) error {
	updates := map[string]interface{}{
		"status":       status,
		"deleted_by":   deletedBy,
		"deleted_at":   deletedAt,
		"delete_error": deleteError,
	}
	result := r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("bucket = ? AND file_key = ?", bucket, fileKey).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListUploadsByUser 列举用户的上传记录
func (r *FileRepository) ListUploadsByUser(ctx context.Context, userID int64, limit, offset int) ([]*model.FileUpload, int64, error) {
	var records []*model.FileUpload
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ? AND status = ?", userID, "completed")

	// 获取总数
	if err := query.Model(&model.FileUpload{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Order("uploaded_at DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListUploadsByRouter 列举函数的上传记录
func (r *FileRepository) ListUploadsByRouter(ctx context.Context, router string, limit, offset int) ([]*model.FileUpload, int64, error) {
	var records []*model.FileUpload
	var total int64

	query := r.db.WithContext(ctx).Where("router = ? AND status = ?", router, "completed")

	// 获取总数
	if err := query.Model(&model.FileUpload{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Order("uploaded_at DESC").Limit(limit).Offset(offset).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// GetStorageStatsByUser 获取用户的存储统计
func (r *FileRepository) GetStorageStatsByUser(ctx context.Context, userID int64) (fileCount int64, totalSize int64, err error) {
	err = r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("user_id = ? AND status = ?", userID, "completed").
		Select("COUNT(*) as file_count, SUM(file_size) as total_size").
		Row().
		Scan(&fileCount, &totalSize)
	return
}

// GetStorageStatsByRouter 获取函数的存储统计
func (r *FileRepository) GetStorageStatsByRouter(ctx context.Context, router string) (fileCount int64, totalSize int64, err error) {
	err = r.db.WithContext(ctx).
		Model(&model.FileUpload{}).
		Where("router = ? AND status = ?", router, "completed").
		Select("COUNT(*) as file_count, SUM(file_size) as total_size").
		Row().
		Scan(&fileCount, &totalSize)
	return
}

// CreateDownloadRecord 创建下载记录（可选）
func (r *FileRepository) CreateDownloadRecord(ctx context.Context, record *model.FileDownload) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// ListSystemFileAssets 查询平台已经登记的文件资产。对象存储中绕过 app-storage 直接写入的对象不在此清单内。
func (r *FileRepository) ListSystemFileAssets(ctx context.Context, filter SystemFileAssetFilter, limit, offset int) ([]SystemFileAssetRow, int64, error) {
	base := r.applySystemFileAssetFilter(r.db.WithContext(ctx).Model(&model.FileUpload{}), filter)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	accesses := r.db.WithContext(ctx).
		Model(&model.FileDownload{}).
		Select(`file_key,
			SUM(CASE WHEN action = 'preview' THEN 0 ELSE 1 END) AS download_count,
			SUM(CASE WHEN action = 'preview' THEN 1 ELSE 0 END) AS preview_count,
			MAX(downloaded_at) AS last_accessed_at`).
		Group("file_key")
	var rows []SystemFileAssetRow
	query := r.applySystemFileAssetFilter(r.db.WithContext(ctx).Table("file_uploads"), filter).
		Select("file_uploads.*, COALESCE(fa.download_count, 0) AS download_count, COALESCE(fa.preview_count, 0) AS preview_count, fa.last_accessed_at").
		Joins("LEFT JOIN (?) AS fa ON fa.file_key = file_uploads.file_key", accesses).
		Order("file_uploads.uploaded_at DESC, file_uploads.id DESC").
		Limit(limit).
		Offset(offset)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *FileRepository) ListFileAccessAudits(ctx context.Context, fileKey string, limit int) ([]model.FileDownload, error) {
	var rows []model.FileDownload
	err := r.db.WithContext(ctx).
		Where("file_key = ?", fileKey).
		Order("downloaded_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *FileRepository) GetSystemFileAssetSummary(ctx context.Context) (SystemFileAssetSummary, error) {
	var summary SystemFileAssetSummary
	err := r.db.WithContext(ctx).Model(&model.FileUpload{}).
		Select(`
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS active_files,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN file_size ELSE 0 END), 0) AS active_bytes,
			COALESCE(SUM(CASE WHEN status = 'deleted' THEN 1 ELSE 0 END), 0) AS deleted_files,
			COALESCE(SUM(CASE WHEN status IN ('failed', 'delete_failed') THEN 1 ELSE 0 END), 0) AS failed_files
		`).Scan(&summary).Error
	return summary, err
}

func (r *FileRepository) ListSystemFileAssetDirectories(ctx context.Context, limit int) ([]SystemFileAssetDirectory, error) {
	var rows []SystemFileAssetDirectory
	query := r.db.WithContext(ctx).Model(&model.FileUpload{}).
		Select("router, COUNT(*) AS file_count, COALESCE(SUM(file_size), 0) AS size_bytes").
		Where("status = ?", "completed").
		Group("router").
		Order("size_bytes DESC, router ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Scan(&rows).Error
	return rows, err
}

func (r *FileRepository) applySystemFileAssetFilter(query *gorm.DB, filter SystemFileAssetFilter) *gorm.DB {
	prefix := strings.Trim(strings.TrimSpace(filter.RouterPrefix), "/")
	if prefix != "" {
		query = query.Where("file_uploads.router = ? OR file_uploads.router LIKE ?", prefix, prefix+"/%")
	}
	if status := strings.TrimSpace(filter.Status); status != "" && status != "all" {
		if status == "failed_all" {
			query = query.Where("file_uploads.status IN ?", []string{"failed", "delete_failed"})
		} else {
			query = query.Where("file_uploads.status = ?", status)
		}
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("file_uploads.file_name LIKE ? OR file_uploads.file_key LIKE ? OR file_uploads.router LIKE ? OR file_uploads.username LIKE ?", like, like, like, like)
	}
	return query
}
