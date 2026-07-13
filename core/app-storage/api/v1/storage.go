package v1

import (
	"github.com/kageos/kageos/core/app-storage/service"
)

const (
	publicCompanyLogoRouter  = "public/company-logos"
	publicCompanyLogoUser    = "anonymous-company-register"
	publicCompanyLogoMaxSize = 512 * 1024
)

// Storage 存储相关API
type Storage struct {
	storageService *service.StorageService
}

// NewStorage 创建存储API（依赖注入）
func NewStorage(storageService *service.StorageService) *Storage {
	return &Storage{
		storageService: storageService,
	}
}
