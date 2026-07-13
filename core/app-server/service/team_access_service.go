package service

import (
	"context"

	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
)

type TeamAccessService struct {
	teamAccessRepo *repository.TeamAccessRepository
	operateLogRepo *repository.OperateLogRepository
	appRepo        *repository.AppRepository
	userLookup     func(ctx context.Context, username string) (*dto.UserInfo, error)
}

func NewTeamAccessService(
	teamAccessRepo *repository.TeamAccessRepository,
	operateLogRepo *repository.OperateLogRepository,
	appRepo *repository.AppRepository,
) *TeamAccessService {
	return &TeamAccessService{
		teamAccessRepo: teamAccessRepo,
		operateLogRepo: operateLogRepo,
		appRepo:        appRepo,
		userLookup:     lookupUserForTeamAccess,
	}
}
