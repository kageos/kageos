package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/storage"
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

	// 根据TraceId创建下载目录
	traceID := c.ctx.msg.TraceId
	if traceID == "" {
		traceID = "default"
		logger.Warnf(c.ctx, "[DownloadFiles] TraceId为空，使用默认目录: default")
	}
	downloadDir := filepath.Join("/app/workplace/uploads", traceID)
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		logger.Errorf(c.ctx, "[DownloadFiles] 创建下载目录失败: %v", err)
		return nil
	}

	logger.Infof(c.ctx, "[DownloadFiles] 开始下载文件，TraceId=%s, 目录=%s, 文件数量=%d", traceID, downloadDir, len(refs))

	ctx := apicall.NewContext(c.ctx.token, c.ctx.msg.TraceId)
	resolveResp, err := apicall.ResolveFileRefs(ctx, &dto.ResolveFileRefsReq{
		Refs:     refs,
		Audience: "server",
	})
	if err != nil {
		logger.Errorf(c.ctx, "[DownloadFiles] 解析文件引用失败: %v", err)
		return nil
	}

	type resolvedFile struct {
		name              string
		key               string
		hash              string
		downloadURL       string
		serverDownloadURL string
	}

	resolvedFiles := make([]resolvedFile, 0, len(resolveResp.Files))
	for _, item := range resolveResp.Files {
		resolvedFiles = append(resolvedFiles, resolvedFile{
			name:              item.Name,
			key:               item.Key,
			hash:              item.Hash,
			downloadURL:       item.DownloadURL,
			serverDownloadURL: item.ServerDownloadURL,
		})
	}

	// 并发下载所有文件
	var wg sync.WaitGroup
	localPaths := make([]string, len(resolvedFiles))

	downloadCount := 0
	skipCount := 0

	for i, file := range resolvedFiles {
		// 容器内优先使用服务端下载地址。
		downloadURL := file.serverDownloadURL
		if downloadURL == "" {
			downloadURL = file.downloadURL
		}
		if downloadURL == "" {
			logger.Warnf(c.ctx, "[DownloadFiles] 文件 %s 没有可用下载地址，跳过", file.name)
			skipCount++
			continue
		}

		downloadCount++
		hasHash := file.hash != ""

		wg.Add(1)
		go func(idx int, f resolvedFile, url string, useCache bool) {
			defer wg.Done()

			fileName := f.name
			if fileName == "" {
				fileName = filepath.Base(f.key)
			}
			targetPath := filepath.Join(downloadDir, fileName)
			var localPath string
			var err error

			if useCache {
				localPath, _, err = c.fileCache.GetOrDownload(c.ctx, f.hash, url, targetPath)
			} else {
				localPath, err = c.fileCache.DownloadOnly(c.ctx, url, targetPath)
			}
			if err != nil {
				logger.Errorf(c.ctx, "[DownloadFiles] 下载文件失败 %s: %v", f.name, err)
				return
			}

			localPaths[idx] = localPath

			if useCache {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(缓存): %s", f.name)
			} else {
				logger.Infof(c.ctx, "[DownloadFiles] 下载文件完成(无hash不缓存): %s", f.name)
			}
		}(i, file, downloadURL, hasHash)
	}

	wg.Wait()

	logger.Infof(c.ctx, "[DownloadFiles] 下载完成: 总文件数=%d, 下载=%d, 跳过=%d", len(resolvedFiles), downloadCount, skipCount)
	downloadedPaths := make([]string, 0, len(localPaths))
	for _, path := range localPaths {
		if path != "" {
			downloadedPaths = append(downloadedPaths, path)
		}
	}
	return downloadedPaths
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

// calculateSHA256 计算文件的SHA256 hash
func calculateSHA256(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
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

// batchUploadFiles 批量上传文件（核心实现）
func (c *Context) batchUploadFiles(filePaths []string) string {
	if len(filePaths) == 0 {
		return ""
	}

	// 限制批量大小（最多100个）
	const maxBatchSize = 100
	if len(filePaths) > maxBatchSize {
		logger.Warnf(c, "[batchUploadFiles] 文件数量超过限制 (%d > %d)，只处理前 %d 个", len(filePaths), maxBatchSize, maxBatchSize)
		filePaths = filePaths[:maxBatchSize]
	}

	// 1. 收集所有文件信息（并行计算hash）
	fileInfos, err := c.collectFileInfos(filePaths)
	if err != nil {
		logger.Errorf(c, "[batchUploadFiles] Failed to collect file infos: %v", err)
		return ""
	}

	if len(fileInfos) == 0 {
		return ""
	}

	// 2. 批量获取上传凭证
	batchTokenReq := &dto.BatchGetUploadTokenReq{
		Files:        make([]dto.GetUploadTokenReq, 0, len(fileInfos)),
		UploadSource: dto.UploadSourceServer, // ✨ 服务端上传，使用 server_endpoint
	}

	for _, info := range fileInfos {
		batchTokenReq.Files = append(batchTokenReq.Files, dto.GetUploadTokenReq{
			Router:       c.msg.GetFullRouter(),
			FileName:     info.FileName,
			ContentType:  info.ContentType,
			FileSize:     info.FileSize,
			Hash:         info.Hash,
			UploadSource: dto.UploadSourceServer, // 服务端上传，使用 server_endpoint
		})
	}

	// 使用 apicall.NewContext 创建包含 token 和 traceId 的 context
	ctx := apicall.NewContext(c.token, c.msg.TraceId)

	credsResp, err := apicall.BatchGetUploadToken(ctx, batchTokenReq)
	if err != nil {
		logger.Errorf(c, "[batchUploadFiles] Failed to get batch upload tokens: %v", err)
		return ""
	}

	if len(credsResp.Tokens) != len(fileInfos) {
		logger.Warnf(c, "[batchUploadFiles] Token count mismatch: expected %d, got %d", len(fileInfos), len(credsResp.Tokens))
	}

	// 3. 创建上传器工厂
	factory := storage.GetDefaultFactory()
	storageType := "" // 从第一个token获取（假设所有文件使用相同的存储引擎）

	// 4. 并发上传所有文件
	type uploadResult struct {
		fileInfo *FileInfo
		cred     *dto.GetUploadTokenResp
		result   *storage.UploadResult
		err      error
	}

	uploadResults := make([]uploadResult, len(fileInfos))
	var wg sync.WaitGroup

	for i, info := range fileInfos {
		if i >= len(credsResp.Tokens) {
			uploadResults[i] = uploadResult{
				fileInfo: info,
				err:      fmt.Errorf("缺少上传凭证"),
			}
			continue
		}

		cred := &credsResp.Tokens[i]
		if storageType == "" {
			storageType = cred.Storage
		}

		wg.Add(1)
		go func(idx int, fileInfo *FileInfo, cred *dto.GetUploadTokenResp) {
			defer wg.Done()

			// 创建上传器
			uploader, err := factory.NewUploader(cred.Storage)
			if err != nil {
				uploadResults[idx] = uploadResult{
					fileInfo: fileInfo,
					err:      fmt.Errorf("创建上传器失败: %w", err),
				}
				return
			}

			// 重置文件指针
			if _, err := fileInfo.File.Seek(0, 0); err != nil {
				uploadResults[idx] = uploadResult{
					fileInfo: fileInfo,
					err:      fmt.Errorf("重置文件指针失败: %w", err),
				}
				return
			}

			// 执行上传
			result, err := uploader.Upload(c, cred, fileInfo.File, fileInfo.FileSize, fileInfo.Hash)
			uploadResults[idx] = uploadResult{
				fileInfo: fileInfo,
				cred:     cred,
				result:   result,
				err:      err,
			}
		}(i, info, cred)
	}

	wg.Wait()

	// 5. 构建批量完成通知请求
	completeItems := make([]dto.BatchUploadCompleteItem, 0, len(uploadResults))
	uploadResultMap := make(map[string]*uploadResult) // key -> uploadResult，用于后续回填 ref

	for _, uploadRes := range uploadResults {
		if uploadRes.err != nil {
			// 上传失败
			logger.Errorf(c, "[batchUploadFiles] Upload failed for file %s: %v", uploadRes.fileInfo.Path, uploadRes.err)
			if uploadRes.cred != nil {
				completeItems = append(completeItems, dto.BatchUploadCompleteItem{
					Key:     uploadRes.cred.Key,
					Bucket:  uploadRes.cred.Bucket,
					Success: false,
					Error:   uploadRes.err.Error(),
					Router:  c.msg.GetFullRouter(),
				})
			}
			continue
		}

		// 上传成功，保存映射关系（用于后续更新DownloadURL）
		uploadResultMap[uploadRes.result.Key] = &uploadRes

		// 添加到完成通知列表
		completeItems = append(completeItems, dto.BatchUploadCompleteItem{
			Key:         uploadRes.result.Key,
			Bucket:      uploadRes.cred.Bucket,
			Success:     true,
			Router:      c.msg.GetFullRouter(),
			FileName:    uploadRes.fileInfo.FileName,
			FileSize:    uploadRes.fileInfo.FileSize,
			ContentType: uploadRes.fileInfo.ContentType,
			Hash:        uploadRes.fileInfo.Hash,
		})
	}

	// 6. 批量通知上传完成（最终只返回稳定 ref 字符串）
	successRefs := make([]string, 0, len(uploadResultMap))
	if len(completeItems) > 0 {
		// 分批通知（每批最多100个）
		const batchSize = 100
		for i := 0; i < len(completeItems); i += batchSize {
			end := i + batchSize
			if end > len(completeItems) {
				end = len(completeItems)
			}

			batchReq := &dto.BatchUploadCompleteReq{
				Items: completeItems[i:end],
			}

			// 使用 apicall.NewContext 创建包含 token 和 traceId 的 context
			ctx := apicall.NewContext(c.token, c.msg.TraceId)

			completeResp, err := apicall.BatchUploadComplete(ctx, batchReq)
			if err != nil {
				logger.Warnf(c, "[batchUploadFiles] Failed to notify batch upload complete (batch %d-%d): %v", i, end-1, err)
				// 如果通知失败，使用上传 token 中的 ref
				for _, item := range completeItems[i:end] {
					if item.Success {
						if uploadRes, ok := uploadResultMap[item.Key]; ok {
							if uploadRes.cred.Ref != "" {
								successRefs = append(successRefs, uploadRes.cred.Ref)
							} else {
								successRefs = append(successRefs, types.JoinRef(uploadRes.cred.Bucket, uploadRes.cred.Key))
							}
						}
					}
				}
				continue
			}

			// 使用批量完成接口返回的 ref 更新文件对象
			if completeResp != nil && len(completeResp.Results) > 0 {
				for _, result := range completeResp.Results {
					if result.Status == "completed" {
						if uploadRes, ok := uploadResultMap[result.Key]; ok {
							if result.Ref != "" {
								successRefs = append(successRefs, result.Ref)
							} else if uploadRes.cred.Ref != "" {
								successRefs = append(successRefs, uploadRes.cred.Ref)
							} else {
								successRefs = append(successRefs, types.JoinRef(uploadRes.cred.Bucket, uploadRes.cred.Key))
							}
						}
					}
				}
			}
		}
	}

	// 7. 关闭所有文件
	for _, info := range fileInfos {
		if info.File != nil {
			info.File.Close()
		}
	}

	// 8. 构建返回结果
	return types.JoinFileRefs(successRefs)
}

// collectFileInfos 收集文件信息（并行计算hash）
func (c *Context) collectFileInfos(filePaths []string) ([]*FileInfo, error) {
	type fileInfoResult struct {
		info *FileInfo
		err  error
	}

	results := make([]fileInfoResult, len(filePaths))
	var wg sync.WaitGroup

	for i, path := range filePaths {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()

			info, err := c.collectSingleFileInfo(filePath)
			results[idx] = fileInfoResult{
				info: info,
				err:  err,
			}
		}(i, path)
	}

	wg.Wait()

	// 过滤掉失败的文件
	fileInfos := make([]*FileInfo, 0, len(results))
	for i, result := range results {
		if result.err != nil {
			if i < len(filePaths) {
				logger.Errorf(c, "[collectFileInfos] Failed to collect file info for %s: %v", filePaths[i], result.err)
			} else {
				logger.Errorf(c, "[collectFileInfos] Failed to collect file info: %v", result.err)
			}
			continue
		}
		if result.info != nil {
			fileInfos = append(fileInfos, result.info)
		}
	}

	return fileInfos, nil
}

// collectSingleFileInfo 收集单个文件信息
func (c *Context) collectSingleFileInfo(filePath string) (*FileInfo, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	// 获取MIME类型
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 计算hash
	hash, err := calculateSHA256(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("计算hash失败: %w", err)
	}

	return &FileInfo{
		Path:        filePath,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		Hash:        hash,
		File:        file,
	}, nil
}
