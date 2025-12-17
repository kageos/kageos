package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// DirectoryUpdateHistoryService 目录更新历史服务
type DirectoryUpdateHistoryService struct {
	directoryUpdateHistoryRepo *repository.DirectoryUpdateHistoryRepository
}

// NewDirectoryUpdateHistoryService 创建目录更新历史服务
func NewDirectoryUpdateHistoryService(directoryUpdateHistoryRepo *repository.DirectoryUpdateHistoryRepository) *DirectoryUpdateHistoryService {
	return &DirectoryUpdateHistoryService{
		directoryUpdateHistoryRepo: directoryUpdateHistoryRepo,
	}
}

// GetAppVersionUpdateHistory 获取应用版本更新历史（App视角）
// 返回二维数组：一个app有多个版本，每个版本里有数组列举了多个目录的变更
// 如果指定了 appVersion，只返回该版本；如果未指定，返回所有版本
func (s *DirectoryUpdateHistoryService) GetAppVersionUpdateHistory(ctx context.Context, appID int64, appVersion string) (*dto.GetAppVersionUpdateHistoryResp, error) {
	var histories []*model.DirectoryUpdateHistory
	var err error

	if appVersion != "" {
		// 查询指定版本所有目录的变更
		histories, err = s.directoryUpdateHistoryRepo.GetUpdateHistoryByAppVersion(appID, appVersion)
	} else {
		// 查询所有版本的变更
		histories, err = s.directoryUpdateHistoryRepo.GetAllVersionsUpdateHistory(appID)
	}

	if err != nil {
		return nil, fmt.Errorf("查询更新历史失败: %w", err)
	}

	// 按版本分组，构建二维数组结构
	// 第一层：版本列表（每个版本一个元素）
	// 第二层：该版本下的所有目录变更（数组）
	versionMap := make(map[string][]*dto.DirectoryChangeInfo)

	for _, history := range histories {
		directoryChange := &dto.DirectoryChangeInfo{
			FullCodePath:      history.FullCodePath,
			DirVersion:        history.DirVersion,
			DirVersionNum:     history.DirVersionNum,
			AddedAPIs:         history.AddedAPIs,
			UpdatedAPIs:       history.UpdatedAPIs,
			DeletedAPIs:       history.DeletedAPIs,
			AddedCount:        history.AddedCount,
			UpdatedCount:      history.UpdatedCount,
			DeletedCount:      history.DeletedCount,
			Summary:           history.Summary,
			Requirement:       history.Requirement,
			ChangeDescription: history.ChangeDescription,
			Duration:          history.Duration,
			UpdatedBy:         history.UpdatedBy,
			CreatedAt:         time.Time(history.CreatedAt),
		}

		if versionMap[history.AppVersion] == nil {
			versionMap[history.AppVersion] = make([]*dto.DirectoryChangeInfo, 0)
		}
		versionMap[history.AppVersion] = append(versionMap[history.AppVersion], directoryChange)
	}

	// 转换为版本信息列表（按版本号倒序）
	versionInfos := make([]*dto.AppVersionUpdateInfo, 0, len(versionMap))
	for version, changes := range versionMap {
		versionInfos = append(versionInfos, &dto.AppVersionUpdateInfo{
			AppVersion:       version,
			DirectoryChanges: changes,
		})
	}

	// 按版本号倒序排序（按 AppVersionNum 排序）
	sort.Slice(versionInfos, func(i, j int) bool {
		// 从 histories 中找到对应的 AppVersionNum
		var iNum, jNum int
		for _, h := range histories {
			if h.AppVersion == versionInfos[i].AppVersion {
				iNum = h.AppVersionNum
			}
			if h.AppVersion == versionInfos[j].AppVersion {
				jNum = h.AppVersionNum
			}
		}
		return iNum > jNum // 倒序
	})

	resp := &dto.GetAppVersionUpdateHistoryResp{
		AppID:      appID,
		AppVersion: appVersion, // 如果为空，表示返回所有版本
		Versions:   versionInfos, // 二维数组：版本列表，每个版本包含目录变更数组
	}

	logger.Infof(ctx, "[DirectoryUpdateHistoryService] GetAppVersionUpdateHistory success: appID=%d, appVersion=%s, versionCount=%d", appID, appVersion, len(versionInfos))
	return resp, nil
}

// GetDirectoryUpdateHistory 获取目录更新历史（目录视角）
func (s *DirectoryUpdateHistoryService) GetDirectoryUpdateHistory(ctx context.Context, appID int64, fullCodePath string, page, pageSize int) (*dto.GetDirectoryUpdateHistoryResp, error) {
	// 限制分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 // 最大100条
	}

	offset := (page - 1) * pageSize
	histories, total, err := s.directoryUpdateHistoryRepo.GetUpdateHistoryByDirectory(appID, fullCodePath, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("查询更新历史失败: %w", err)
	}

	// 转换为响应格式
	directoryChanges := make([]*dto.DirectoryChangeInfo, len(histories))
	for i, history := range histories {
		directoryChanges[i] = &dto.DirectoryChangeInfo{
			FullCodePath:      history.FullCodePath,
			DirVersion:        history.DirVersion,
			DirVersionNum:     history.DirVersionNum,
			AppVersion:        history.AppVersion,
			AppVersionNum:     history.AppVersionNum,
			AddedAPIs:         history.AddedAPIs,
			UpdatedAPIs:       history.UpdatedAPIs,
			DeletedAPIs:       history.DeletedAPIs,
			AddedCount:        history.AddedCount,
			UpdatedCount:      history.UpdatedCount,
			DeletedCount:      history.DeletedCount,
			Summary:           history.Summary,
			Requirement:       history.Requirement,
			ChangeDescription: history.ChangeDescription,
			Duration:          history.Duration,
			UpdatedBy:         history.UpdatedBy,
			CreatedAt:         time.Time(history.CreatedAt),
		}
	}

	// 计算总页数
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	resp := &dto.GetDirectoryUpdateHistoryResp{
		AppID:         appID,
		FullCodePath:  fullCodePath,
		DirectoryChanges: directoryChanges,
		Paginated: &dto.PaginatedInfo{
			CurrentPage: page,
			TotalCount:  int(total),
			TotalPages:  totalPages,
			PageSize:    pageSize,
		},
	}

	logger.Infof(ctx, "[DirectoryUpdateHistoryService] GetDirectoryUpdateHistory success: appID=%d, fullCodePath=%s, count=%d, total=%d", appID, fullCodePath, len(histories), total)
	return resp, nil
}

