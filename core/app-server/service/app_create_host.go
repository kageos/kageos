package service

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
)

func (a *AppService) selectHostForCreateApp() (*model.Host, error) {
	hostRepo := repository.NewHostRepository(a.appRepo.GetDB())
	hosts, err := hostRepo.GetHostList()
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
