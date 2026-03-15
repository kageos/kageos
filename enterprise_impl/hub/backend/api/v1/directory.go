package v1

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/hub/backend/dto"
	"github.com/ai-agent-os/hub/backend/service"
	"github.com/gin-gonic/gin"
)

type Directory struct {
	directoryService *service.HubDirectoryService
}

// NewDirectory 创建 Directory 处理器（依赖注入）
func NewDirectory(directoryService *service.HubDirectoryService) *Directory {
	return &Directory{
		directoryService: directoryService,
	}
}

// PublishDirectory 发布目录到 Hub
// @Summary 发布目录到 Hub
// @Description 发布目录及其所有子目录到 Hub
// @Tags Hub目录管理
// @Accept json
// @Produce json
// @Param request body dto.PublishHubDirectoryRequest true "发布目录请求"
// @Success 200 {object} dto.PublishHubDirectoryResponse "发布成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/hub/directories/publish [post]
func (d *Directory) PublishDirectory(c *gin.Context) {
	var req dto.PublishHubDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := d.directoryService.PublishDirectory(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "[Directory] 发布目录失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// UpdateDirectory 更新目录到 Hub（用于 push）
// @Summary 更新目录到 Hub
// @Description 更新已发布的目录到 Hub（类似 git push）
// @Tags Hub目录管理
// @Accept json
// @Produce json
// @Param request body dto.UpdateHubDirectoryRequest true "更新目录请求"
// @Success 200 {object} dto.UpdateHubDirectoryResponse "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/hub/directories/update [put]
func (d *Directory) UpdateDirectory(c *gin.Context) {
	var req dto.UpdateHubDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := d.directoryService.UpdateDirectory(ctx, &req)
	if err != nil {
		logger.Errorf(ctx, "[Directory] 更新目录失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDirectoryList 获取目录列表
// @Summary 获取目录列表
// @Description 获取 Hub 目录列表（分页）
// @Tags Hub目录管理
// @Accept json
// @Produce json
// @Param request query dto.GetHubDirectoryListRequest false "查询参数"
// @Success 200 {object} dto.HubDirectoryListResponse "获取成功"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/hub/directories [get]
func (d *Directory) GetDirectoryList(c *gin.Context) {
	var req dto.GetHubDirectoryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 限制分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 最大100条
	}

	// 只看自己：从请求上下文取当前用户，未登录则 401
	publisherUsername := req.PublisherUsername
	if req.MineOnly {
		currentUser := contextx.GetRequestUser(c)
		if currentUser == "" {
			response.NoAuth(c, "请先登录后再查看「只看我的」")
			return
		}
		publisherUsername = currentUser
	}

	ctx := contextx.ToContext(c)
	// copy_url 使用主站 host:port：优先配置文件 public_host（主站地址），否则请求头
	host := config.GetHubConfig().GetPublicHost()
	if host == "" {
		host = contextx.GetPresignHost(c)
	}
	if host == "" {
		host = c.Request.Host
	}
	resp, err := d.directoryService.GetDirectoryList(ctx, req.Page, req.PageSize, req.Search, req.Category, publisherUsername, req.FeeType, req.OrderBy, host)
	if err != nil {
		logger.Errorf(ctx, "[Directory] 获取目录列表失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDirectoryDetail 获取目录详情
// @Summary 获取目录详情
// @Description 获取 Hub 目录详情（可选包含目录树和文件列表）
// @Tags Hub目录管理
// @Accept json
// @Produce json
// @Param request query dto.GetHubDirectoryDetailRequest true "查询参数"
// @Success 200 {object} dto.HubDirectoryDetailDTO "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 404 {string} string "目录不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/hub/directories/detail [get]
func (d *Directory) GetDirectoryDetail(c *gin.Context) {
	var req dto.GetHubDirectoryDetailRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证参数：必须提供 hub_directory_id 或 full_code_path 之一
	if req.HubDirectoryID == 0 && req.FullCodePath == "" {
		response.FailWithMessage(c, "必须提供 hub_directory_id 或 full_code_path 之一")
		return
	}

	ctx := contextx.ToContext(c)
	host := config.GetHubConfig().GetPublicHost()
	if host == "" {
		host = contextx.GetPresignHost(c)
	}
	if host == "" {
		host = c.Request.Host
	}
	resp, err := d.directoryService.GetDirectoryDetail(ctx, req.HubDirectoryID, req.FullCodePath, req.Version, req.IncludeTree, host)
	if err != nil {
		logger.Errorf(ctx, "[Directory] 获取目录详情失败: %v", err)
		if err.Error() == "record not found" {
			response.FailWithMessage(c, "目录不存在")
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDirectoryVersions 获取目录版本列表
// @Summary 获取目录版本列表
// @Description 获取指定 Hub 目录的所有历史版本（用于详情页右侧展示）
// @Tags Hub目录管理
// @Accept json
// @Produce json
// @Param hub_directory_id query int false "Hub 目录ID（与 full_code_path 二选一）"
// @Param full_code_path query string false "目录完整路径（与 hub_directory_id 二选一）"
// @Success 200 {object} dto.GetHubDirectoryVersionsResponse "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/hub/directories/versions [get]
func (d *Directory) GetDirectoryVersions(c *gin.Context) {
	var req dto.GetHubDirectoryVersionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	ctx := contextx.ToContext(c)
	hubHost := config.GetHubConfig().GetPublicHost()
	if hubHost == "" {
		hubHost = contextx.GetPresignHost(c)
	}
	if hubHost == "" {
		hubHost = c.Request.Host
	}
	hubDirectoryID := req.HubDirectoryID
	if hubDirectoryID <= 0 && req.FullCodePath != "" {
		dir, err := d.directoryService.GetDirectoryDetail(ctx, 0, req.FullCodePath, "", false, hubHost)
		if err != nil || dir == nil {
			response.FailWithMessage(c, "根据 full_code_path 获取目录失败")
			return
		}
		hubDirectoryID = dir.ID
	}
	if hubDirectoryID <= 0 {
		response.FailWithMessage(c, "必须提供 hub_directory_id 或 full_code_path 之一")
		return
	}
	resp, err := d.directoryService.ListDirectoryVersions(ctx, hubDirectoryID)
	if err != nil {
		logger.Errorf(ctx, "[Directory] 获取目录版本列表失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// IncrementDownloadCount 复制/下载时增加下载次数（公开接口，供 app-server 在 copy 成功后调用）
func (d *Directory) IncrementDownloadCount(c *gin.Context) {
	var req struct {
		FullCodePath string `json:"full_code_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "缺少 full_code_path")
		return
	}
	ctx := contextx.ToContext(c)
	if err := d.directoryService.IncrementDownloadCount(ctx, req.FullCodePath); err != nil {
		logger.Errorf(ctx, "[Directory] 增加下载次数失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"message": "ok"})
}

// Star 为目录加星（类似 GitHub star）
func (d *Directory) Star(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.FailWithMessage(c, "缺少目录 ID")
		return
	}
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的目录 ID")
		return
	}
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	if err := d.directoryService.Star(ctx, id, username); err != nil {
		logger.Errorf(ctx, "[Directory] 加星失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"message": "ok"})
}

// Unstar 取消星星
func (d *Directory) Unstar(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.FailWithMessage(c, "缺少目录 ID")
		return
	}
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的目录 ID")
		return
	}
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	if err := d.directoryService.Unstar(ctx, id, username); err != nil {
		logger.Errorf(ctx, "[Directory] 取消星星失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"message": "ok"})
}

// DeleteDirectory 删除应用（软删除：只改状态，数据保留，通过链接仍可访问；仅发布者可操作）
func (d *Directory) DeleteDirectory(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		response.FailWithMessage(c, "缺少目录 ID")
		return
	}
	var id int64
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的目录 ID")
		return
	}
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "请先登录")
		return
	}
	ctx := contextx.ToContext(c)
	if err := d.directoryService.DeleteDirectory(ctx, id, username); err != nil {
		logger.Errorf(ctx, "[Directory] 删除目录失败: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, gin.H{"message": "已下架"})
}
