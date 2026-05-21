package service

import (
	"context"

	"github.com/kageos/kageos/dto"
)

func (a *AppService) provisionRuntimeApp(ctx context.Context, hostID int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	return a.appCall.CreateApp(ctx, hostID, req)
}
