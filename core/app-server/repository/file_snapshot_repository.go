package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

type FileSnapshotRepository struct {
	db *gorm.DB
}

func NewFileSnapshotRepository(db *gorm.DB) *FileSnapshotRepository {
	return &FileSnapshotRepository{db: db}
}

// CreateBatch 批量创建文件快照
func (r *FileSnapshotRepository) CreateBatch(snapshots []*model.FileSnapshot) error {
	ctx := context.Background()
	if len(snapshots) == 0 {
		return nil
	}

	logger.Infof(ctx, "[FileSnapshotRepository.CreateBatch] 批量创建文件快照: count=%d", len(snapshots))

	if err := r.db.CreateInBatches(snapshots, 100).Error; err != nil {
		logger.Errorf(ctx, "[FileSnapshotRepository.CreateBatch] 批量创建失败: error=%v", err)
		return err
	}

	logger.Infof(ctx, "[FileSnapshotRepository.CreateBatch] 批量创建成功: count=%d", len(snapshots))
	return nil
}

// GetByDirectoryAndVersion 根据目录路径和目录版本获取该目录下所有文件的快照（用于目录回滚）
func (r *FileSnapshotRepository) GetByDirectoryAndVersion(appID int64, fullCodePath, dirVersion string) ([]*model.FileSnapshot, error) {
	var snapshots []*model.FileSnapshot
	err := r.db.Where("app_id = ? AND full_code_path = ? AND dir_version = ?",
		appID, fullCodePath, dirVersion).
		Order("relative_path ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// GetLatestFileSnapshot 获取文件的最新快照（用于变更检测）
func (r *FileSnapshotRepository) GetLatestFileSnapshot(appID int64, fullCodePath, fileName string) (*model.FileSnapshot, error) {
	var snapshot model.FileSnapshot
	err := r.db.Where("app_id = ? AND full_code_path = ? AND file_name = ?",
		appID, fullCodePath, fileName).
		Order("file_version_num DESC").
		First(&snapshot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 文件不存在，返回 nil
		}
		return nil, err
	}
	return &snapshot, nil
}

// GetLatestFileSnapshots 批量获取多个文件的最新快照（用于批量变更检测）
func (r *FileSnapshotRepository) GetLatestFileSnapshots(appID int64, fullCodePath string, fileNames []string) (map[string]*model.FileSnapshot, error) {
	if len(fileNames) == 0 {
		return make(map[string]*model.FileSnapshot), nil
	}

	// 使用子查询获取每个文件的最新快照
	var snapshots []*model.FileSnapshot
	err := r.db.Where("app_id = ? AND full_code_path = ? AND file_name IN ?",
		appID, fullCodePath, fileNames).
		Order("file_name ASC, file_version_num DESC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}

	// 构建结果映射（每个文件只保留最新版本）
	result := make(map[string]*model.FileSnapshot)
	for _, snapshot := range snapshots {
		if existing, exists := result[snapshot.FileName]; !exists || snapshot.FileVersionNum > existing.FileVersionNum {
			result[snapshot.FileName] = snapshot
		}
	}

	return result, nil
}

// GetByFileAndVersion 根据文件路径和文件版本获取文件快照（用于文件回滚）
func (r *FileSnapshotRepository) GetByFileAndVersion(appID int64, fullCodePath, fileName, fileVersion string) (*model.FileSnapshot, error) {
	var snapshot model.FileSnapshot
	err := r.db.Where("app_id = ? AND full_code_path = ? AND file_name = ? AND file_version = ?",
		appID, fullCodePath, fileName, fileVersion).
		First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// GetCurrentVersionByDirectory 获取目录当前版本的所有文件快照（需要配合 ServiceTreeRepository）
func (r *FileSnapshotRepository) GetCurrentVersionByDirectory(appID int64, fullCodePath string, serviceTreeRepo *ServiceTreeRepository) ([]*model.FileSnapshot, error) {
	// 先获取目录节点（ServiceTree）
	serviceTree, err := serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return nil, err
	}

	// 如果版本为空或为0，返回空列表（节点还没有快照）
	if serviceTree.Version == "" || serviceTree.VersionNum == 0 {
		return []*model.FileSnapshot{}, nil
	}

	// 获取该目录版本的所有文件快照
	return r.GetByDirectoryAndVersion(appID, fullCodePath, serviceTree.Version)
}

// GetCurrentVersionsByDirectories 批量获取多个目录的当前版本文件快照
func (r *FileSnapshotRepository) GetCurrentVersionsByDirectories(appID int64, paths []string, serviceTreeRepo *ServiceTreeRepository) (map[string][]*model.FileSnapshot, error) {
	if len(paths) == 0 {
		return make(map[string][]*model.FileSnapshot), nil
	}

	// 先批量获取目录节点（ServiceTree）
	serviceTrees, err := serviceTreeRepo.GetServiceTreeByFullPaths(paths)
	if err != nil {
		return nil, err
	}

	// 批量获取文件快照
	result := make(map[string][]*model.FileSnapshot)
	for path, serviceTree := range serviceTrees {
		// 如果版本为空或为0，跳过（节点还没有快照）
		if serviceTree.Version == "" || serviceTree.VersionNum == 0 {
			result[path] = []*model.FileSnapshot{}
			continue
		}

		snapshots, err := r.GetByDirectoryAndVersion(appID, path, serviceTree.Version)
		if err != nil {
			return nil, err
		}
		result[path] = snapshots
	}

	return result, nil
}

// GetByPathAndAppVersion 根据目录路径和应用版本获取文件快照（用于版本回滚）
func (r *FileSnapshotRepository) GetByPathAndAppVersion(appID int64, fullCodePath string, appVersionNum int) ([]*model.FileSnapshot, error) {
	var snapshots []*model.FileSnapshot
	err := r.db.Where("app_id = ? AND full_code_path = ? AND app_version_num <= ?",
		appID, fullCodePath, appVersionNum).
		Order("app_version_num DESC, relative_path ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}


