package service

import (
	"context"
	"encoding/json"
	"time"

	sharedDto "github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func (s *AppManageService) fetchVersionDiffPayload(
	ctx context.Context,
	user, app, version string,
	logPrefix string,
) (interface{}, error) {
	updateCallbackResponse, callbackErr := s.sendUpdateCallbackAndWait(ctx, user, app, version)
	if callbackErr != nil {
		logger.Warnf(ctx, "[%s] ❌ 获取 diff 失败: %v", logPrefix, callbackErr)
		return nil, callbackErr
	}

	logger.Infof(ctx, "[%s] ✅ 获取 diff 成功: %+v", logPrefix, updateCallbackResponse)
	return updateCallbackResponse.Data, nil
}

func (s *AppManageService) requestVersionDiff(ctx context.Context, user, app, version string, logPrefix string) *sharedDto.DiffData {
	diffPayload, err := s.fetchVersionDiffPayload(ctx, user, app, version, logPrefix)
	if err != nil {
		return nil
	}

	return s.parseDiffData(ctx, diffPayload, logPrefix)
}

func (s *AppManageService) requestRequiredVersionDiff(
	ctx context.Context,
	user, app, version string,
	logPrefix string,
) (*sharedDto.DiffData, error) {
	diffPayload, err := s.fetchVersionDiffPayload(ctx, user, app, version, logPrefix)
	if err != nil {
		return nil, err
	}

	return s.parseDiffData(ctx, diffPayload, logPrefix), nil
}

func (s *AppManageService) parseDiffData(ctx context.Context, data interface{}, logPrefix string) *sharedDto.DiffData {
	if data == nil {
		return nil
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		logger.Warnf(ctx, "[%s] 序列化 diff 数据失败: %v", logPrefix, err)
		return nil
	}

	var diffData sharedDto.DiffData
	if err := json.Unmarshal(dataBytes, &diffData); err != nil {
		logger.Warnf(ctx, "[%s] 反序列化 diff 数据失败: %v", logPrefix, err)
		return nil
	}

	return &diffData
}

func (s *AppManageService) collectVersionDiffFromTemporaryContainer(
	ctx context.Context,
	user, app, version, appDir string,
) *sharedDto.DiffData {
	if s.runtimeDriver == nil {
		return nil
	}

	waiterChan := s.registerStartupWaiter(user, app, version)
	defer s.unregisterStartupWaiter(user, app, version)

	if err := s.createVersionContainer(ctx, user, app, version, appDir); err != nil {
		logger.Warnf(ctx, "[BatchWriteFiles] 创建容器失败: %v，继续执行（不获取 diff）", err)
		return nil
	}

	logger.Infof(ctx, "[BatchWriteFiles] 等待新版本启动: %s/%s/%s", user, app, version)
	select {
	case <-waiterChan:
		logger.Infof(ctx, "[BatchWriteFiles] ✅ 新版本启动成功: %s/%s/%s", user, app, version)
		diff := s.requestVersionDiff(ctx, user, app, version, "BatchWriteFiles")
		if err := s.stopOldVersionContainer(ctx, user, app, version); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 停止临时容器失败: %v", err)
		}
		return diff
	case <-time.After(60 * time.Second):
		logger.Warnf(ctx, "[BatchWriteFiles] ⚠️ 等待新版本启动超时，不获取 diff")
		return nil
	}
}
