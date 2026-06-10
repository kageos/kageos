package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	ref               string
	key               string
	hash              string
	errorMessage      string
	downloadURL       string
	serverDownloadURL string
}

type downloadStats struct {
	downloadCount int
	skipCount     int
}

type DownloadFilesResult struct {
	Paths         []string
	Refs          []string
	ResolvedCount int
	DownloadCount int
	SkipCount     int
	Issues        []string
}

func (r DownloadFilesResult) ErrorMessage() string {
	return strings.Join(compactNonEmptyStrings(r.Issues), "；")
}

type downloadCandidate struct {
	label string
	url   string
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
		Audience: "all",
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
	if resolveResp == nil {
		logger.Errorf(c.ctx, "[DownloadFiles] 解析文件引用失败: storage 返回空响应")
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
			ref:               item.Ref,
			name:              item.Name,
			key:               item.Key,
			hash:              item.Hash,
			errorMessage:      item.Error,
			downloadURL:       item.DownloadURL,
			serverDownloadURL: item.ServerDownloadURL,
		})
	}
	return resolvedFiles
}

func (c *FS) downloadResolvedFiles(files []resolvedDownloadFile, downloadDir string) ([]string, downloadStats, []string) {
	var wg sync.WaitGroup
	localPaths := make([]string, len(files))
	issues := make([]string, len(files))
	stats := downloadStats{}

	for i, file := range files {
		if file.errorMessage != "" {
			issues[i] = fmt.Sprintf("文件 %s 解析失败: %s", file.label(), file.errorMessage)
			logger.Warnf(c.ctx, "[DownloadFiles] %s", issues[i])
			stats.skipCount++
			continue
		}
		if len(file.downloadCandidates()) == 0 {
			issues[i] = fmt.Sprintf("文件 %s 没有可用下载地址", file.label())
			logger.Warnf(c.ctx, "[DownloadFiles] %s，跳过", issues[i])
			stats.skipCount++
			continue
		}

		stats.downloadCount++
		wg.Add(1)
		go func(idx int, f resolvedDownloadFile) {
			defer wg.Done()

			localPath, err := c.downloadResolvedFile(f, downloadDir)
			if err != nil {
				issues[idx] = fmt.Sprintf("文件 %s 下载失败: %v", f.label(), err)
				logger.Errorf(c.ctx, "[DownloadFiles] %s", issues[idx])
				return
			}

			localPaths[idx] = localPath
			if f.hash != "" {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(缓存): %s", f.label())
			} else {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(无hash不缓存): %s", f.label())
			}
		}(i, file)
	}

	wg.Wait()
	return localPaths, stats, compactNonEmptyStrings(issues)
}

func (c *FS) downloadResolvedFile(file resolvedDownloadFile, downloadDir string) (string, error) {
	targetPath := filepath.Join(downloadDir, file.targetFileName())
	failures := make([]string, 0, 2)
	for _, candidate := range file.downloadCandidates() {
		if file.hash != "" {
			localPath, _, err := c.fileCache.GetOrDownload(c.ctx, file.hash, candidate.url, targetPath)
			if err == nil {
				return localPath, nil
			}
			failures = append(failures, fmt.Sprintf("%s URL: %s", candidate.label, summarizeDownloadError(err)))
			continue
		}
		localPath, err := c.fileCache.DownloadOnly(c.ctx, candidate.url, targetPath)
		if err == nil {
			return localPath, nil
		}
		failures = append(failures, fmt.Sprintf("%s URL: %s", candidate.label, summarizeDownloadError(err)))
	}
	return "", errors.New(strings.Join(failures, "; "))
}

func summarizeDownloadError(err error) string {
	if err == nil {
		return ""
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Err != nil {
			return fmt.Sprintf("%s: %v", urlErr.Op, urlErr.Err)
		}
		return urlErr.Op
	}
	return err.Error()
}

func (f resolvedDownloadFile) preferredDownloadURL() string {
	if f.serverDownloadURL != "" {
		return f.serverDownloadURL
	}
	return f.downloadURL
}

func (f resolvedDownloadFile) downloadCandidates() []downloadCandidate {
	candidates := make([]downloadCandidate, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, candidate := range []downloadCandidate{
		{label: "server", url: strings.TrimSpace(f.serverDownloadURL)},
		{label: "browser", url: strings.TrimSpace(f.downloadURL)},
	} {
		if candidate.url == "" {
			continue
		}
		if _, ok := seen[candidate.url]; ok {
			continue
		}
		seen[candidate.url] = struct{}{}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func (f resolvedDownloadFile) targetFileName() string {
	if f.name != "" {
		return f.name
	}
	return filepath.Base(f.key)
}

func (f resolvedDownloadFile) label() string {
	for _, value := range []string{f.name, f.ref, f.key} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "unknown"
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
