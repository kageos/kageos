package app

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/storage"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

type FS struct {
	ctx *Context
}

// UploadFile 上传文件（从文件路径）
// filePath: 本地文件路径
// 返回: types.File 对象，包含上传后的文件信息
func (fs *FS) UploadFile(filePath string) (*types.File, error) {
	return uploadFile(fs.ctx, filePath)
}

// uploadFile 内部上传实现
func uploadFile(ctx *Context, filePath string) (*types.File, error) {
	// 1. 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 2. 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	// 3. 获取文件MIME类型
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream" // 默认类型
	}

	// 4. 计算文件SHA256 hash（用于秒传和去重）
	hash, err := calculateSHA256(file)
	if err != nil {
		return nil, fmt.Errorf("计算文件hash失败: %w", err)
	}

	// 重置文件指针到开头（用于后续上传）
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 5. 构建Header（用于调用存储服务API）
	header := &apicall.Header{
		TraceID:     ctx.msg.TraceId,
		RequestUser: ctx.msg.RequestUser,
		Token:       ctx.token, // ✨ 使用Context中保存的token（透传前端token）
	}

	// 6. 获取上传凭证
	getTokenReq := &dto.GetUploadTokenReq{
		Router:       ctx.msg.GetFullRouter(), // 函数路径
		FileName:     fileName,
		ContentType:  contentType,
		FileSize:     fileSize,
		Hash:         hash,                   // 传递hash，用于秒传
		UploadSource: dto.UploadSourceServer, // ✨ 服务端上传，使用 server_endpoint
	}

	creds, err := apicall.GetUploadToken(header, getTokenReq)
	if err != nil {
		return nil, fmt.Errorf("获取上传凭证失败: %w", err)
	}

	logger.Infof(ctx, "[UploadFile] Got upload credentials: key=%s, method=%s, storage=%s", creds.Key, creds.Method, creds.Storage)

	// 7. 创建对应的上传器（根据storage字段）
	factory := storage.GetDefaultFactory()
	uploader, err := factory.NewUploader(creds.Storage)
	if err != nil {
		return nil, fmt.Errorf("创建上传器失败: %w", err)
	}

	// 8. 执行上传
	uploadResult, err := uploader.Upload(ctx, creds, file, fileSize, hash)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}

	logger.Infof(ctx, "[UploadFile] Upload successful: key=%s, etag=%s", uploadResult.Key, uploadResult.ETag)

	// 9. 通知存储服务上传完成
	completeReq := &dto.UploadCompleteReq{
		Key:         uploadResult.Key,
		Success:     true,
		Router:      ctx.msg.GetFullRouter(),
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		Hash:        hash,
	}

	_, err = apicall.UploadComplete(header, completeReq)
	if err != nil {
		// 上传已完成，通知失败不影响整体流程，只记录日志
		logger.Warnf(ctx, "[UploadFile] Failed to notify upload complete: %v", err)
	}

	// 10. 构建返回的File对象
	now := time.Now().Unix()
	fileObj := &types.File{
		Name:        fileName,                       // 文件名
		SourceName:  fileName,                       // 源文件名（与Name相同）
		Storage:     creds.Storage,                  // 存储引擎类型
		Description: "",                             // 描述（可选）
		Hash:        hash,                           // SHA256 hash
		Size:        fileSize,                       // 文件大小
		UploadTs:    now,                            // 上传时间戳
		LocalPath:   filePath,                       // 本地路径
		IsUploaded:  true,                           // 已上传到云端
		Url:         uploadResult.DownloadURL,       // ✨ 外部访问地址（前端下载使用）
		ServerUrl:   uploadResult.ServerDownloadURL, // ✨ 内部访问地址（服务端下载使用）
		Downloaded:  true,                           // 已下载到本地（因为是本地文件上传）
	}

	return fileObj, nil
}

// calculateSHA256 计算文件的SHA256 hash
func calculateSHA256(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (c *Context) GetFS() *FS {
	return &FS{c}
}

// FileInfo 文件信息（用于批量上传）
type FileInfo struct {
	Path        string   // 文件路径
	FileName    string   // 文件名
	FileSize    int64    // 文件大小
	ContentType string   // MIME类型
	Hash        string   // SHA256 hash
	File        *os.File // 文件句柄（用于上传）
}

// ResponseDirFiles 把指定文件夹下的所有文件都给上传了
func (c *FS) ResponseDirFiles(dir string) *types.Files {
	// 1. 读取目录下的所有文件
	files, err := readDirFiles(dir)
	if err != nil {
		logger.Errorf(c.ctx, "[ResponseDirFiles] Failed to read directory: %v", err)
		return &types.Files{
			Files:      []*types.File{},
			UploadUser: c.ctx.msg.RequestUser, // ✨ 记录上传用户
			Remark:     fmt.Sprintf("读取目录失败: %v", err),
			Metadata:   make(map[string]interface{}),
		}
	}

	// 2. 批量上传
	return c.ctx.batchUploadFiles(files)
}

// ResponseFiles 上传多个文件
func (c *FS) ResponseFiles(filePaths []string) *types.Files {
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
func (c *Context) batchUploadFiles(filePaths []string) *types.Files {
	if len(filePaths) == 0 {
		return &types.Files{
			Files:      []*types.File{},
			UploadUser: c.msg.RequestUser, // ✨ 记录上传用户
			Remark:     "没有文件需要上传",
			Metadata:   make(map[string]interface{}),
		}
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
		return &types.Files{
			Files:      []*types.File{},
			UploadUser: c.msg.RequestUser, // ✨ 记录上传用户
			Remark:     fmt.Sprintf("收集文件信息失败: %v", err),
			Metadata:   make(map[string]interface{}),
		}
	}

	if len(fileInfos) == 0 {
		return &types.Files{
			Files:      []*types.File{},
			UploadUser: c.msg.RequestUser, // ✨ 记录上传用户
			Remark:     "没有有效的文件",
			Metadata:   make(map[string]interface{}),
		}
	}

	// 2. 批量获取上传凭证
	header := &apicall.Header{
		TraceID:     c.msg.TraceId,
		RequestUser: c.msg.RequestUser,
		Token:       c.token,
	}

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
			UploadSource: dto.UploadSourceServer, // ✨ 服务端上传，使用 server_endpoint
		})
	}

	credsResp, err := apicall.BatchGetUploadToken(header, batchTokenReq)
	if err != nil {
		logger.Errorf(c, "[batchUploadFiles] Failed to get batch upload tokens: %v", err)
		return &types.Files{
			Files:      []*types.File{},
			UploadUser: c.msg.RequestUser, // ✨ 记录上传用户
			Remark:     fmt.Sprintf("获取上传凭证失败: %v", err),
			Metadata:   make(map[string]interface{}),
		}
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
	uploadResultMap := make(map[string]*uploadResult) // key -> uploadResult，用于后续更新DownloadURL
	now := time.Now().Unix()

	for _, uploadRes := range uploadResults {
		if uploadRes.err != nil {
			// 上传失败
			logger.Errorf(c, "[batchUploadFiles] Upload failed for file %s: %v", uploadRes.fileInfo.Path, uploadRes.err)
			if uploadRes.cred != nil {
				completeItems = append(completeItems, dto.BatchUploadCompleteItem{
					Key:     uploadRes.cred.Key,
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
			Success:     true,
			Router:      c.msg.GetFullRouter(),
			FileName:    uploadRes.fileInfo.FileName,
			FileSize:    uploadRes.fileInfo.FileSize,
			ContentType: uploadRes.fileInfo.ContentType,
			Hash:        uploadRes.fileInfo.Hash,
		})
	}

	// 6. 批量通知上传完成（并使用响应中的DownloadURL更新文件对象）
	successFiles := make([]*types.File, 0, len(uploadResultMap))
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

			completeResp, err := apicall.BatchUploadComplete(header, batchReq)
			if err != nil {
				logger.Warnf(c, "[batchUploadFiles] Failed to notify batch upload complete (batch %d-%d): %v", i, end-1, err)
				// 如果通知失败，使用上传时的DownloadURL
				for _, item := range completeItems[i:end] {
					if item.Success {
						if uploadRes, ok := uploadResultMap[item.Key]; ok {
							fileObj := &types.File{
								Name:        uploadRes.fileInfo.FileName,
								SourceName:  uploadRes.fileInfo.FileName,
								Storage:     uploadRes.cred.Storage,
								Description: "",
								Hash:        uploadRes.fileInfo.Hash,
								Size:        uploadRes.fileInfo.FileSize,
								UploadTs:    now,
								LocalPath:   uploadRes.fileInfo.Path,
								IsUploaded:  true,
								Url:         uploadRes.result.DownloadURL,       // ✨ 外部访问地址（前端下载使用）
								ServerUrl:   uploadRes.result.ServerDownloadURL, // ✨ 内部访问地址（服务端下载使用）
								Downloaded:  true,
							}
							successFiles = append(successFiles, fileObj)
						}
					}
				}
				continue
			}

			// ✨ 使用批量完成接口返回的DownloadURL更新文件对象
			if completeResp != nil && len(completeResp.Results) > 0 {
				for _, result := range completeResp.Results {
					if result.Status == "completed" {
						if uploadRes, ok := uploadResultMap[result.Key]; ok {
							// ✨ 使用批量完成接口返回的DownloadURL（更准确）
							downloadURL := result.DownloadURL
							if downloadURL == "" {
								// 如果响应中没有URL，使用上传时的URL作为fallback
								downloadURL = uploadRes.result.DownloadURL
							}

							// ✨ 使用批量完成接口返回的ServerDownloadURL（更准确）
							serverDownloadURL := result.ServerDownloadURL
							if serverDownloadURL == "" {
								// 如果响应中没有ServerURL，使用上传时的URL作为fallback
								serverDownloadURL = uploadRes.result.ServerDownloadURL
							}

							fileObj := &types.File{
								Name:        uploadRes.fileInfo.FileName,
								SourceName:  uploadRes.fileInfo.FileName,
								Storage:     uploadRes.cred.Storage,
								Description: "",
								Hash:        uploadRes.fileInfo.Hash,
								Size:        uploadRes.fileInfo.FileSize,
								UploadTs:    now,
								LocalPath:   uploadRes.fileInfo.Path,
								IsUploaded:  true,
								Url:         downloadURL,       // ✨ 外部访问地址（前端下载使用）
								ServerUrl:   serverDownloadURL, // ✨ 内部访问地址（服务端下载使用）
								Downloaded:  true,
							}
							successFiles = append(successFiles, fileObj)
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
	return &types.Files{
		Files:      successFiles,
		UploadUser: c.msg.RequestUser, // ✨ 记录上传用户
		Remark:     fmt.Sprintf("成功上传 %d/%d 个文件", len(successFiles), len(filePaths)),
		Metadata:   make(map[string]interface{}),
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
