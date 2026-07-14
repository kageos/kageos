package v1

import (
	"context"
	"net/http"
	"time"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-storage/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *Storage) browserURLForRef(ctx context.Context, ref string) string {
	if ref == "" {
		return ""
	}
	bucket, key, err := s.storageService.ParseFileRef(ref)
	if err != nil {
		logger.Warnf(ctx, "Failed to parse file preview ref %q: %v", ref, err)
		return ""
	}
	browserURL, _, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
	if err != nil {
		logger.Warnf(ctx, "Failed to generate file preview URL for ref %q: %v", ref, err)
		return ""
	}
	return browserURL
}

// ResolveFileRefs 批量解析文件引用，返回元数据和直连 URL。
// 前端使用 download_url 直连对象存储；SDK/容器使用 server_download_url 直连内部对象存储。
func (s *Storage) ResolveFileRefs(c *gin.Context) {
	var req dto.ResolveFileRefsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	files, err := s.storageService.ResolveFileRefs(ctx, req.Refs, req.Audience)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OkWithData(c, dto.ResolveFileRefsResp{Files: files})
}

// UpdateFileDescription 更新文件描述
// @Summary 更新文件描述
// @Description 更新文件引用对应的描述元数据
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.UpdateFileDescriptionReq true "更新文件描述请求"
// @Success 200 {object} dto.UpdateFileDescriptionResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/files/description [post]
func (s *Storage) UpdateFileDescription(c *gin.Context) {
	var req dto.UpdateFileDescriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.storageService.UpdateFileDescription(ctx, req.Ref, req.Bucket, req.Key, req.Description)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OkWithData(c, resp)
}

// DownloadFile 获取文件（重定向到对象存储直连地址）
// @Summary 下载文件
// @Description 重定向到对象存储直连地址，避免服务端转发文件流
// @Tags 存储管理
// @Accept json
// @Produce application/octet-stream
// @Param key path string true "文件 Key"
// @Success 200 {file} file "文件内容"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/download/{key} [get]
func (s *Storage) DownloadFile(c *gin.Context) {
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.BadRequest(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 获取文件信息
	_, err := s.storageService.GetFileInfo(ctx, key)
	if err != nil {
		response.NotFound(c, "文件不存在或无法访问")
		return
	}

	// 记录下载（如果启用）
	requestUser := contextx.GetRequestUser(c)

	// 获取客户端 IP 和 User-Agent（规范化IP地址）
	ipAddress := normalizeIP(c.ClientIP())
	userAgent := c.GetHeader("User-Agent")

	// 创建下载记录（只记录 username，不记录 user_id）
	var usernameValue *string
	if requestUser != "" {
		usernameValue = &requestUser
	}

	downloadRecord := &model.FileDownload{
		FileKey:   key,
		UserID:    nil,
		Username:  usernameValue,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// 异步记录下载（不阻塞响应）
	persistCtx, cancelPersist := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	go func() {
		defer cancelPersist()
		if err := s.storageService.RecordDownload(persistCtx, downloadRecord); err != nil {
			logger.Errorf(persistCtx, "Failed to record download: %v", err)
			// 不影响下载流程，只记录错误
		}
	}()

	downloadURL, _, _, err := s.storageService.GetFileURLs(ctx, key)
	if err != nil || downloadURL == "" {
		logger.Errorf(c, "Failed to resolve download URL: %v", err)
		response.Internal(c, "生成下载链接失败")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURL)
}

// DeleteFile 删除文件
// @Summary 删除文件
// @Description 删除存储的文件
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param key path string true "文件 Key"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/files/{key} [delete]
func (s *Storage) DeleteFile(c *gin.Context) {
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.BadRequest(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	err := s.storageService.DeleteFile(ctx, key)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OkWithMessage(c, "文件删除成功")
}

// GetFileInfo 获取文件信息
// @Summary 获取文件信息
// @Description 获取文件的元数据信息
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param key path string true "文件 Key"
// @Success 200 {object} dto.GetFileInfoResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/files/{key}/info [get]
func (s *Storage) GetFileInfo(c *gin.Context) {
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.BadRequest(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	info, err := s.storageService.GetFileInfo(ctx, key)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := &dto.GetFileInfoResp{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified.Format(http.TimeFormat),
	}

	response.OkWithData(c, resp)
}

// GetStorageStats 获取存储统计信息
// @Summary 获取存储统计信息
// @Description 获取某个函数路径下的文件数量和总大小
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param router query string true "函数路径，例如：luobei/test88888/plugins/cashier_desk"
// @Success 200 {object} dto.GetStorageStatsResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/stats [get]
func (s *Storage) GetStorageStats(c *gin.Context) {
	var req dto.GetStorageStatsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	fileCount, totalSize, err := s.storageService.GetStorageStats(ctx, req.Router)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 转换为人类可读的大小
	sizeHuman := formatSize(totalSize)

	resp := &dto.GetStorageStatsResp{
		Router:    req.Router,
		FileCount: fileCount,
		TotalSize: totalSize,
		SizeHuman: sizeHuman,
	}

	response.OkWithData(c, resp)
}

// ListFiles 列举文件
// @Summary 列举文件
// @Description 列举某个函数路径下的所有文件
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param router query string true "函数路径"
// @Success 200 {object} dto.ListFilesResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/files [get]
func (s *Storage) ListFiles(c *gin.Context) {
	var req dto.ListFilesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	files, err := s.storageService.ListFilesByRouter(ctx, req.Router)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := &dto.ListFilesResp{
		Router: req.Router,
		Files:  files,
		Count:  len(files),
	}

	response.OkWithData(c, resp)
}

// DeleteFilesByRouter 删除函数路径下的所有文件
// @Summary 删除函数路径下的所有文件
// @Description 批量删除某个函数路径下的所有文件（危险操作）
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.DeleteFilesByRouterReq true "删除请求"
// @Success 200 {object} dto.DeleteFilesByRouterResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/batch-delete [post]
func (s *Storage) DeleteFilesByRouter(c *gin.Context) {
	var req dto.DeleteFilesByRouterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	deletedCount, err := s.storageService.DeleteFilesByRouter(ctx, req.Router)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := &dto.DeleteFilesByRouterResp{
		Router:       req.Router,
		DeletedCount: deletedCount,
	}

	response.OkWithData(c, resp)
}
