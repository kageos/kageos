package service

import (
	"context"
	"time"

	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
)

type OperateLogService struct {
	operateLogRepo *repository.OperateLogRepository
}

func NewOperateLogService(operateLogRepo *repository.OperateLogRepository) *OperateLogService {
	return &OperateLogService{operateLogRepo: operateLogRepo}
}

func (s *OperateLogService) GetOperateLogs(ctx context.Context, req *dto.GetOperateLogsReq) (*dto.GetOperateLogsResp, error) {
	logs, total, err := s.operateLogRepo.GetOperateLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	logItems := make([]dto.OperateLogItem, len(logs))
	for i, log := range logs {
		logItems[i] = dto.OperateLogItem{
			ID:            log.ID,
			TenantUser:    log.TenantUser,
			CompanyCode:   log.CompanyCode,
			App:           log.App,
			ActorUser:     log.ActorUser,
			Action:        log.Action,
			ResourceType:  log.ResourceType,
			ResourcePath:  log.ResourcePath,
			ResourceName:  log.ResourceName,
			TargetUser:    log.TargetUser,
			TargetID:      log.TargetID,
			Summary:       log.Summary,
			DetailsJSON:   log.DetailsJSON,
			OldValuesJSON: log.OldValuesJSON,
			NewValuesJSON: log.NewValuesJSON,
			Status:        log.Status,
			Source:        log.Source,
			IPAddress:     log.IPAddress,
			UserAgent:     log.UserAgent,
			TraceID:       log.TraceID,
			CreatedAt:     time.Time(log.CreatedAt).Format("2006-01-02 15:04:05"),
			UpdatedAt:     time.Time(log.UpdatedAt).Format("2006-01-02 15:04:05"),
		}
	}

	return &dto.GetOperateLogsResp{
		Logs:     logItems,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}
