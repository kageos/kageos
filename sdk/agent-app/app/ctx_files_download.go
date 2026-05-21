package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/publicshare"
)

type resolvedDownloadFile struct {
	name              string
	key               string
	hash              string
	downloadURL       string
	serverDownloadURL string
}

type downloadStats struct {
	downloadCount int
	skipCount     int
}

func (c *FS) prepareDownloadDir() (string, string, bool) {
	traceID := c.ctx.msg.TraceId
	if traceID == "" {
		traceID = "default"
		logger.Warnf(c.ctx, "[DownloadFiles] TraceId为空，使用默认目录: default")
	}

	downloadDir := filepath.Join("/app/workplace/uploads", traceID)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		logger.Errorf(c.ctx, "[DownloadFiles] 创建下载目录失败: %v", err)
		return traceID, downloadDir, false
	}
	return traceID, downloadDir, true
}

func (c *FS) resolveDownloadFiles(refs []string) ([]resolvedDownloadFile, bool) {
	req := &dto.ResolveFileRefsReq{
		Refs:     refs,
		Audience: "server",
	}
	ctx := c.resolveFileRefsContext()
	var resolveResp *dto.ResolveFileRefsResp
	var err error
	if shareID := c.publicShareID(); shareID != "" {
		resolveResp, err = apicall.ResolvePublicShareFileRefs(ctx, shareID, req)
	} else {
		resolveResp, err = apicall.ResolveFileRefs(ctx, req)
	}
	if err != nil {
		logger.Errorf(c.ctx, "[DownloadFiles] 解析文件引用失败: %v", err)
		return nil, false
	}
	return toResolvedDownloadFiles(resolveResp.Files), true
}

func (c *FS) resolveFileRefsContext() context.Context {
	ctx := c.ctx.apiCallContext()
	if c.ctx.anonymousToken != "" {
		ctx = context.WithValue(ctx, publicshare.AnonymousTokenHeader, c.ctx.anonymousToken)
	}
	return ctx
}

func (c *FS) publicShareID() string {
	if strings.TrimSpace(c.ctx.GetClientSource()) != "public_share" {
		return ""
	}
	if sourceType := strings.TrimSpace(contextx.GetSourceType(c.ctx.Context)); sourceType != "" && sourceType != "public_share" {
		return ""
	}
	return strings.TrimSpace(contextx.GetSourceRef(c.ctx.Context))
}

func toResolvedDownloadFiles(files []dto.ResolvedFile) []resolvedDownloadFile {
	resolvedFiles := make([]resolvedDownloadFile, 0, len(files))
	for _, item := range files {
		resolvedFiles = append(resolvedFiles, resolvedDownloadFile{
			name:              item.Name,
			key:               item.Key,
			hash:              item.Hash,
			downloadURL:       item.DownloadURL,
			serverDownloadURL: item.ServerDownloadURL,
		})
	}
	return resolvedFiles
}

func (c *FS) downloadResolvedFiles(files []resolvedDownloadFile, downloadDir string) ([]string, downloadStats) {
	var wg sync.WaitGroup
	localPaths := make([]string, len(files))
	stats := downloadStats{}

	for i, file := range files {
		downloadURL := file.preferredDownloadURL()
		if downloadURL == "" {
			logger.Warnf(c.ctx, "[DownloadFiles] 文件 %s 没有可用下载地址，跳过", file.name)
			stats.skipCount++
			continue
		}

		stats.downloadCount++
		wg.Add(1)
		go func(idx int, f resolvedDownloadFile, url string) {
			defer wg.Done()

			localPath, err := c.downloadResolvedFile(f, url, downloadDir)
			if err != nil {
				logger.Errorf(c.ctx, "[DownloadFiles] 下载文件失败 %s: %v", f.name, err)
				return
			}

			localPaths[idx] = localPath
			if f.hash != "" {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(缓存): %s", f.name)
			} else {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(无hash不缓存): %s", f.name)
			}
		}(i, file, downloadURL)
	}

	wg.Wait()
	return localPaths, stats
}

func (c *FS) downloadResolvedFile(file resolvedDownloadFile, downloadURL string, downloadDir string) (string, error) {
	targetPath := filepath.Join(downloadDir, file.targetFileName())
	if file.hash != "" {
		localPath, _, err := c.fileCache.GetOrDownload(c.ctx, file.hash, downloadURL, targetPath)
		return localPath, err
	}
	return c.fileCache.DownloadOnly(c.ctx, downloadURL, targetPath)
}

func (f resolvedDownloadFile) preferredDownloadURL() string {
	if f.serverDownloadURL != "" {
		return f.serverDownloadURL
	}
	return f.downloadURL
}

func (f resolvedDownloadFile) targetFileName() string {
	if f.name != "" {
		return f.name
	}
	return filepath.Base(f.key)
}

func compactNonEmptyStrings(values []string) []string {
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			compacted = append(compacted, value)
		}
	}
	return compacted
}
