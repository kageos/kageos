package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"gorm.io/gorm"
)

// FileRepository 文件仓储层
type FileRepository struct {
	db *gorm.DB
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

// GetUploadRecord 获取上传记录
func (r *FileRepository) GetUploadRecord(ctx context.Context, fileKey string) (*model.FileUpload, error) {
	var record model.FileUpload
	err := r.db.WithContext(ctx).Where("file_key = ?", fileKey).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
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

