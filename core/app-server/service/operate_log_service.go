package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type OperateLogService struct {
	operateLogRepo *repository.OperateLogRepository
}

func NewOperateLogService(operateLogRepo *repository.OperateLogRepository) *OperateLogService {
	return &OperateLogService{operateLogRepo: operateLogRepo}
}

func (s *OperateLogService) GetTableOperateLogs(ctx context.Context, req *dto.GetTableOperateLogsReq) (*dto.GetTableOperateLogsResp, error) {
	logs, total, err := s.operateLogRepo.GetTableOperateLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	logItems := make([]interface{}, len(logs))
	for i, log := range logs {
		logItems[i] = log
	}

	return &dto.GetTableOperateLogsResp{
		Logs:     logItems,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
