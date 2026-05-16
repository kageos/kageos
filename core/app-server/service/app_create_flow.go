package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (a *AppService) createAppFlow(ctx context.Context, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	requestUser, tenantUser, err := a.validateCreateAppRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	selectedHost, err := a.selectHostForCreateApp()
	if err != nil {
		return nil, err
	}

	resp, err := a.provisionRuntimeApp(ctx, selectedHost.ID, req)
	if err != nil {
		return nil, err
	}

	app, rootNode := a.buildInitialAppAndRoot(requestUser, tenantUser, req, selectedHost)
	if err := a.persistCreatedApp(ctx, app, rootNode); err != nil {
		return nil, err
	}

	//a.grantCreateAppAdmins(ctx, tenantUser, req.Code, req.Admins)
	return resp, nil
}
