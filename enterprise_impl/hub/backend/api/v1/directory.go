package v1

import (
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

	// 获取发布者用户名（从 context 或请求头）
	publisherUsername := contextx.GetRequestUser(c)
	if publisherUsername == "" {
		publisherUsername = "system" // 默认值
	}

	ctx := contextx.ToContext(c)
	resp, err := d.directoryService.PublishDirectory(ctx, &req, publisherUsername)
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

	// 获取发布者用户名（从 context 或请求头）
	publisherUsername := contextx.GetRequestUser(c)
	if publisherUsername == "" {
		publisherUsername = "system" // 默认值
	}

	ctx := contextx.ToContext(c)
	resp, err := d.directoryService.UpdateDirectory(ctx, &req, publisherUsername)
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

	ctx := contextx.ToContext(c)
	resp, err := d.directoryService.GetDirectoryList(ctx, req.Page, req.PageSize, req.Search, req.Category, req.PublisherUsername)
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
	resp, err := d.directoryService.GetDirectoryDetail(ctx, req.HubDirectoryID, req.FullCodePath, req.Version, req.IncludeTree)
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
