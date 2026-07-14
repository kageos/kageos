package service

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
)

func (a *AppService) selectHostForCreateApp(ctx context.Context) (*model.Host, error) {
	hostRepo := repository.NewHostRepository(a.appRepo.GetDB(ctx))
	hosts, err := hostRepo.GetHostList(ctx)
	if err != nil || len(hosts) == 0 {
		return nil, fmt.Errorf("无法获取可用的主机: %w", err)
	}

	var selectedHost *model.Host
	for _, host := range hosts {
		if host.Status == "enabled" {
			if selectedHost == nil || host.AppCount < selectedHost.AppCount {
				selectedHost = host
			}
		}
	}
	if selectedHost == nil {
		return nil, fmt.Errorf("没有可用的主机")
	}
	return selectedHost, nil
}
