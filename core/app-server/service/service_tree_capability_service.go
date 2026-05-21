package service

import (
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
)

type serviceTreeCapabilityBundleService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	appService       *AppService
	docService       *DocService
}

func newServiceTreeCapabilityBundleService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
	docService *DocService,
) *serviceTreeCapabilityBundleService {
	return &serviceTreeCapabilityBundleService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		appService:       appService,
		docService:       docService,
	}
}

func capabilitySourceFilePath(file *model.FileSnapshot) string {
	if file == nil {
		return ""
	}
	if strings.TrimSpace(file.RelativePath) != "" {
		return strings.TrimSpace(file.RelativePath)
	}
	name := strings.TrimSpace(file.FileName)
	if name == "" {
		return ""
	}
	if path.Ext(name) != "" {
		return name
	}
	return name + ".go"
}

func splitCapabilityBundleFilePath(filePath string) (string, string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", "", fmt.Errorf("文件路径不能为空")
	}
	ext := path.Ext(filePath)
	if ext == "" {
		return "", "", fmt.Errorf("文件路径必须包含扩展名: %s", filePath)
	}
	return strings.TrimSuffix(path.Base(filePath), ext), strings.TrimPrefix(ext, "."), nil
}

func isGeneratedDirectoryInitFile(relativePath, fileName string) bool {
	relativePath = strings.TrimSpace(relativePath)
	fileName = strings.TrimSpace(fileName)
	return path.Base(relativePath) == "init_.go" || fileName == "init_"
}
