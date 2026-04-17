package storage

import (
	"fmt"
	"strings"
)

// Factory 存储工厂
type Factory struct{}

// NewFactory 创建存储工厂
func NewFactory() *Factory {
	return &Factory{}
}

// CreateStorage 根据类型创建存储实例。
// 当前仅支持 MinIO；其他类型不再假装“可切换但未实现”。
func (f *Factory) CreateStorage(storageType string, cfg Config) (Storage, error) {
	switch strings.ToLower(strings.TrimSpace(storageType)) {
	case "", "minio":
		return NewMinIOStorage(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s (only minio is supported)", storageType)
	}
}
