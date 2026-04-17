package storage

import "errors"

var (
	// ErrUnsupportedStorage 不支持的存储引擎
	ErrUnsupportedStorage = errors.New("不支持的存储引擎")

	// ErrUploadFailed 上传失败
	ErrUploadFailed = errors.New("上传失败")

	// ErrInvalidCredentials 无效的上传凭证
	ErrInvalidCredentials = errors.New("无效的上传凭证")
)
