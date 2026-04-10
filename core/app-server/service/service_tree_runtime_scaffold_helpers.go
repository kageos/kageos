package service

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func newSingleDirectoryScaffoldRuntimeReq(user, app string, serviceTree *model.ServiceTree) *dto.BatchCreateDirectoryTreeRuntimeReq {
	return &dto.BatchCreateDirectoryTreeRuntimeReq{
		User: user,
		App:  app,
		Items: []*dto.DirectoryScaffoldItem{
			{
				FullCodePath: serviceTree.FullCodePath,
				Name:         serviceTree.Name,
				Description:  serviceTree.Description,
				Tags:         serviceTree.Tags,
			},
		},
	}
}
