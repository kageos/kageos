package app

import (
	"os"
	"path/filepath"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// FileInfo 文件信息（用于批量上传）
type FileInfo struct {
	Path        string   // 文件路径
	FileName    string   // 文件名
	FileSize    int64    // 文件大小
	ContentType string   // MIME类型
	Hash        string   // SHA256 hash
	File        *os.File // 文件句柄（用于上传）
}

func (c *Context) GetFS() *FS {
	return &FS{
		ctx:       c,
		fileCache: GetFileCache(), // 使用全局文件缓存
	}
}

type FS struct {
	ctx       *Context
	fileCache *FileCache // 文件缓存管理器（通过hash实现去重）
}

// ResponseDirFiles 把指定文件夹下的所有文件都给上传了
func (c *FS) ResponseDirFiles(dir string) string {
	// 1. 读取目录下的所有文件
	files, err := readDirFiles(dir)
	if err != nil {
		logger.Errorf(c.ctx, "[ResponseDirFiles] Failed to read directory: %v", err)
		return ""
	}

	// 2. 批量上传
	return c.ctx.batchUploadFiles(files)
}

// ResponseFiles 上传多个文件，返回 bucket/object_key 字符串；多文件用逗号分隔。
func (c *FS) ResponseFiles(filePaths []string) string {
	// 转换为文件信息列表
	files := make([]string, 0, len(filePaths))
	for _, path := range filePaths {
		if path != "" {
			files = append(files, path)
		}
	}

	// 批量上传
	return c.ctx.batchUploadFiles(files)
}

// GetTraceOutputDir 获取基于 TraceId 的唯一输出目录
// 注意：此目录已经基于 TraceId 生成，是唯一的，文件名无需再包含 TraceId
// 如果目录不存在，会自动创建
func (c *FS) GetTraceOutputDir() string {
	outputDir := filepath.Join("/app/workplace/output", c.ctx.msg.TraceId)
	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Errorf(c.ctx, "[GetTraceOutputDir] 创建输出目录失败: %v", err)
		// 即使创建失败也返回路径，让调用方处理错误
	}
	return outputDir
}

// DownloadFiles 下载 files 字符串中的文件到本地，返回本地文件路径列表。
// 根据TraceId创建目录，使用文件缓存机制避免重复下载相同hash的文件
func (c *FS) DownloadFiles(fileRefs string) []string {
	refs := types.ParseFileRefs(fileRefs)
	if len(refs) == 0 {
		logger.Warnf(c.ctx, "[DownloadFiles] 文件列表为空，跳过下载")
		return nil
	}

	traceID, downloadDir, ok := c.prepareDownloadDir()
	if !ok {
		return nil
	}

	logger.Infof(c.ctx, "[DownloadFiles] 开始下载文件，TraceId=%s, 目录=%s, 文件数量=%d", traceID, downloadDir, len(refs))

	resolvedFiles, ok := c.resolveDownloadFiles(refs)
	if !ok {
		return nil
	}

	localPaths, stats := c.downloadResolvedFiles(resolvedFiles, downloadDir)
	logger.Infof(c.ctx, "[DownloadFiles] 下载完成: 总文件数=%d, 下载=%d, 跳过=%d", len(resolvedFiles), stats.downloadCount, stats.skipCount)
	return compactNonEmptyStrings(localPaths)
}

// RemoveFiles 删除 DownloadFiles 下载到本地的文件。
func (c *FS) RemoveFiles(files []string) {
	if len(files) == 0 {
		return
	}

	// 释放文件缓存引用（减少引用计数）
	for _, localPath := range files {
		if localPath != "" {
			c.fileCache.Release(localPath)
		}
	}

	// 根据TraceId删除下载目录
	traceID := c.ctx.msg.TraceId
	if traceID == "" {
		traceID = "default"
	}
	downloadDir := filepath.Join("/app/workplace/uploads", traceID)
	if err := os.RemoveAll(downloadDir); err != nil {
		logger.Errorf(c.ctx, "[RemoveFiles] 删除下载目录失败: %v", err)
	} else {
		logger.Infof(c.ctx, "[RemoveFiles] 已删除下载目录: %s", downloadDir)
	}
}

// readDirFiles 读取目录下的所有文件
func readDirFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件，跳过目录
		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
