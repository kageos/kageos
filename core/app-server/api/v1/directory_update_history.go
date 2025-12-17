package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// DirectoryUpdateHistory 目录更新历史相关API
type DirectoryUpdateHistory struct {
	directoryUpdateHistoryService *service.DirectoryUpdateHistoryService
}

// NewDirectoryUpdateHistory 创建目录更新历史API（依赖注入）
func NewDirectoryUpdateHistory(directoryUpdateHistoryService *service.DirectoryUpdateHistoryService) *DirectoryUpdateHistory {
	return &DirectoryUpdateHistory{
		directoryUpdateHistoryService: directoryUpdateHistoryService,
	}
}

// GetAppVersionUpdateHistory 获取应用版本更新历史（App视角）
// @Summary 获取应用版本更新历史
// @Description 查看整个应用在某个版本的所有目录变更，方便从app侧回滚。返回二维数组结构：一个app有多个版本，每个版本里有数组列举了多个目录的变更
// @Tags 目录更新历史
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app_id query int true "应用ID"
// @Param app_version query string false "应用版本号，如 v101 (可选，如果为空则返回所有版本)"
// @Success 200 {object} dto.GetAppVersionUpdateHistoryResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/directory_update_history/app_version [get]
func (d *DirectoryUpdateHistory) GetAppVersionUpdateHistory(c *gin.Context) {
	var resp *dto.GetAppVersionUpdateHistoryResp
	var err error

	// 从query参数获取app_id
	appIDStr := c.Query("app_id")
	if appIDStr == "" {
		response.FailWithMessage(c, "缺少app_id参数")
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的app_id参数")
		return
	}

	// 从query参数获取app_version（可选，如果为空则返回所有版本）
	appVersion := c.Query("app_version")

	defer func() {
		logger.Infof(c, "GetAppVersionUpdateHistory app_id:%d app_version:%s resp:%+v err:%v", appID, appVersion, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = d.directoryUpdateHistoryService.GetAppVersionUpdateHistory(ctx, appID, appVersion)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// GetDirectoryUpdateHistory 获取目录更新历史（目录视角）
// @Summary 获取目录更新历史
// @Description 查看单个目录的变更历史，方便仅回滚某个目录
// @Tags 目录更新历史
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app_id query int true "应用ID"
// @Param full_code_path query string true "目录完整路径，如 /luobei/app/servercenter/erp"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} dto.GetDirectoryUpdateHistoryResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/directory_update_history/directory [get]
func (d *DirectoryUpdateHistory) GetDirectoryUpdateHistory(c *gin.Context) {
	var resp *dto.GetDirectoryUpdateHistoryResp
	var err error

	// 从query参数获取app_id
	appIDStr := c.Query("app_id")
	if appIDStr == "" {
		response.FailWithMessage(c, "缺少app_id参数")
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的app_id参数")
		return
	}

	// 从query参数获取full_code_path
	fullCodePath := c.Query("full_code_path")
	if fullCodePath == "" {
		response.FailWithMessage(c, "缺少full_code_path参数")
		return
	}

	// 获取分页参数
	page := 1
	pageSize := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	defer func() {
		logger.Infof(c, "GetDirectoryUpdateHistory app_id:%d full_code_path:%s page:%d page_size:%d resp:%+v err:%v", appID, fullCodePath, page, pageSize, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = d.directoryUpdateHistoryService.GetDirectoryUpdateHistory(ctx, appID, fullCodePath, page, pageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}
