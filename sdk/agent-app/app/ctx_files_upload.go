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

const maxUploadBatchSize = 100

type fileUploadResult struct {
	fileInfo *FileInfo
	cred     *dto.GetUploadTokenResp
	result   *storage.UploadResult
	err      error
}

// calculateSHA256 计算文件的SHA256 hash
func calculateSHA256(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// batchUploadFiles 批量上传文件（核心实现）
func (c *Context) batchUploadFiles(filePaths []string) string {
	if len(filePaths) == 0 {
		return ""
	}

	filePaths = c.limitUploadFilePaths(filePaths)
	fileInfos, err := c.collectFileInfos(filePaths)
	if err != nil {
		logger.Errorf(c, "[batchUploadFiles] Failed to collect file infos: %v", err)
		return ""
	}
	defer closeFileInfos(fileInfos)

	if len(fileInfos) == 0 {
		return ""
	}

	credsResp, err := c.fetchBatchUploadTokens(fileInfos)
	if err != nil {
		logger.Errorf(c, "[batchUploadFiles] Failed to get batch upload tokens: %v", err)
		return ""
	}
	if len(credsResp.Tokens) != len(fileInfos) {
		logger.Warnf(c, "[batchUploadFiles] Token count mismatch: expected %d, got %d", len(fileInfos), len(credsResp.Tokens))
	}

	uploadResults := c.uploadFilesWithCredentials(fileInfos, credsResp.Tokens)
	completeItems, uploadResultMap := c.buildUploadCompleteItems(uploadResults)
	successRefs := c.completeUploadedFiles(completeItems, uploadResultMap)
	return types.JoinFileRefs(successRefs)
}

func (c *Context) limitUploadFilePaths(filePaths []string) []string {
	if len(filePaths) <= maxUploadBatchSize {
		return filePaths
	}
	logger.Warnf(c, "[batchUploadFiles] 文件数量超过限制 (%d > %d)，只处理前 %d 个", len(filePaths), maxUploadBatchSize, maxUploadBatchSize)
	return filePaths[:maxUploadBatchSize]
}

func (c *Context) fetchBatchUploadTokens(fileInfos []*FileInfo) (*dto.BatchGetUploadTokenResp, error) {
	ctx := apicall.NewContext(c.token, c.msg.TraceId)
	return apicall.BatchGetUploadToken(ctx, c.buildBatchUploadTokenReq(fileInfos))
}

func (c *Context) buildBatchUploadTokenReq(fileInfos []*FileInfo) *dto.BatchGetUploadTokenReq {
	batchTokenReq := &dto.BatchGetUploadTokenReq{
		Files:        make([]dto.GetUploadTokenReq, 0, len(fileInfos)),
		UploadSource: dto.UploadSourceServer,
	}

	for _, info := range fileInfos {
		batchTokenReq.Files = append(batchTokenReq.Files, dto.GetUploadTokenReq{
			Router:       c.msg.GetFullRouter(),
			FileName:     info.FileName,
			ContentType:  info.ContentType,
			FileSize:     info.FileSize,
			Hash:         info.Hash,
			UploadSource: dto.UploadSourceServer,
		})
	}
	return batchTokenReq
}

func (c *Context) uploadFilesWithCredentials(fileInfos []*FileInfo, tokens []dto.GetUploadTokenResp) []fileUploadResult {
	uploadResults := make([]fileUploadResult, len(fileInfos))
	var wg sync.WaitGroup

	for i, info := range fileInfos {
		if i >= len(tokens) {
			uploadResults[i] = fileUploadResult{
				fileInfo: info,
				err:      fmt.Errorf("缺少上传凭证"),
			}
			continue
		}

		cred := &tokens[i]
		wg.Add(1)
		go func(idx int, fileInfo *FileInfo, cred *dto.GetUploadTokenResp) {
			defer wg.Done()
			uploadResults[idx] = c.uploadSingleFileWithCredential(fileInfo, cred)
		}(i, info, cred)
	}

	wg.Wait()
	return uploadResults
}

func (c *Context) uploadSingleFileWithCredential(fileInfo *FileInfo, cred *dto.GetUploadTokenResp) fileUploadResult {
	uploader, err := storage.NewUploader(cred.Storage)
	if err != nil {
		return fileUploadResult{
			fileInfo: fileInfo,
			err:      fmt.Errorf("创建上传器失败: %w", err),
		}
	}

	if _, err := fileInfo.File.Seek(0, 0); err != nil {
		return fileUploadResult{
			fileInfo: fileInfo,
			err:      fmt.Errorf("重置文件指针失败: %w", err),
		}
	}

	result, err := uploader.Upload(c, cred, fileInfo.File, fileInfo.FileSize, fileInfo.Hash)
	return fileUploadResult{
		fileInfo: fileInfo,
		cred:     cred,
		result:   result,
		err:      err,
	}
}

func (c *Context) buildUploadCompleteItems(uploadResults []fileUploadResult) ([]dto.BatchUploadCompleteItem, map[string]*fileUploadResult) {
	completeItems := make([]dto.BatchUploadCompleteItem, 0, len(uploadResults))
	uploadResultMap := make(map[string]*fileUploadResult)

	for i := range uploadResults {
		uploadRes := &uploadResults[i]
		if uploadRes.err != nil {
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

		uploadResultMap[uploadRes.result.Key] = uploadRes
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
	return completeItems, uploadResultMap
}

func (c *Context) completeUploadedFiles(completeItems []dto.BatchUploadCompleteItem, uploadResultMap map[string]*fileUploadResult) []string {
	successRefs := make([]string, 0, len(uploadResultMap))
	if len(completeItems) == 0 {
		return successRefs
	}

	const batchSize = 100
	for i := 0; i < len(completeItems); i += batchSize {
		end := i + batchSize
		if end > len(completeItems) {
			end = len(completeItems)
		}

		batch := completeItems[i:end]
		batchReq := &dto.BatchUploadCompleteReq{
			Items: batch,
		}
		ctx := apicall.NewContext(c.token, c.msg.TraceId)

		completeResp, err := apicall.BatchUploadComplete(ctx, batchReq)
		if err != nil {
			logger.Warnf(c, "[batchUploadFiles] Failed to notify batch upload complete (batch %d-%d): %v", i, end-1, err)
			successRefs = appendFallbackUploadRefs(successRefs, batch, uploadResultMap)
			continue
		}

		if completeResp != nil && len(completeResp.Results) > 0 {
			successRefs = appendCompletedUploadRefs(successRefs, completeResp.Results, uploadResultMap)
		}
	}
	return successRefs
}

func appendFallbackUploadRefs(successRefs []string, completeItems []dto.BatchUploadCompleteItem, uploadResultMap map[string]*fileUploadResult) []string {
	for _, item := range completeItems {
		if !item.Success {
			continue
		}
		if uploadRes, ok := uploadResultMap[item.Key]; ok {
			successRefs = append(successRefs, refFromUploadResult(uploadRes))
		}
	}
	return successRefs
}

func appendCompletedUploadRefs(successRefs []string, results []dto.BatchUploadCompleteResult, uploadResultMap map[string]*fileUploadResult) []string {
	for _, result := range results {
		if result.Status != "completed" {
			continue
		}
		if uploadRes, ok := uploadResultMap[result.Key]; ok {
			if result.Ref != "" {
				successRefs = append(successRefs, result.Ref)
			} else {
				successRefs = append(successRefs, refFromUploadResult(uploadRes))
			}
		}
	}
	return successRefs
}

func refFromUploadResult(uploadRes *fileUploadResult) string {
	if uploadRes.cred.Ref != "" {
		return uploadRes.cred.Ref
	}
	return types.JoinRef(uploadRes.cred.Bucket, uploadRes.cred.Key)
}

func closeFileInfos(fileInfos []*FileInfo) {
	for _, info := range fileInfos {
		if info.File != nil {
			info.File.Close()
		}
	}
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
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

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
