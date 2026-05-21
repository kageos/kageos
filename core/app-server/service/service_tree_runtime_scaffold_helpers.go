package service

import (
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
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
