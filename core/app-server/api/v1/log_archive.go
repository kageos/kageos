package v1

import (
	"fmt"
	"github.com/kageos/kageos/pkg/access"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type LogArchive struct {
	service     *service.LogArchiveService
	permissions *service.PermissionService
}

func NewLogArchive(archiveService *service.LogArchiveService, permissions ...*service.PermissionService) *LogArchive {
	h := &LogArchive{service: archiveService}
	if len(permissions) > 0 {
		h.permissions = permissions[0]
	}
	return h
}

func (h *LogArchive) List(c *gin.Context) {
	path := access.NormalizeResourcePath(c.Query("resource_path"))
	if err := h.authorize(c, path); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	rows, total, err := h.service.List(contextx.ToContext(c), page, pageSize, path)
	if err != nil {
		response.FailWithMessage(c, "查询日志归档失败: "+err.Error())
		return
	}
	cfg := h.service.Config()
	response.OkWithData(c, gin.H{"progress_supported": true, "list": rows, "total": total, "min_records": cfg.MinRecords, "scheduled_retention_days": cfg.ScheduledSuccessRetentionDays, "retention_days": cfg.RetentionDays, "scheduled_success_retention_days": cfg.ScheduledSuccessRetentionDays, "cron_expr": cfg.CronExpr, "timezone": cfg.Timezone})
}

func (h *LogArchive) Retry(c *gin.Context) {
	if contextx.GetRequestUser(c) != service.SystemUsername && h.permissions == nil {
		response.FailWithMessage(c, "需要归档所属目录或函数的管理权限")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的归档批次")
		return
	}
	if err := h.authorizeBatch(c, id); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := h.service.Retry(contextx.ToContext(c), id); err != nil {
		response.FailWithMessage(c, "重试归档未完成，可从已保存的阶段继续: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"id": id})
}

func (h *LogArchive) Progress(c *gin.Context) {
	if contextx.GetRequestUser(c) != service.SystemUsername {
		response.FailWithMessage(c, "仅 system 超管可查看归档进度")
		return
	}
	id, err := strconv.ParseInt(c.Query("execution_id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效执行编号")
		return
	}
	row, err := h.service.Progress(contextx.ToContext(c), id)
	if err != nil {
		response.FailWithMessage(c, "查询归档进度失败")
		return
	}
	response.OkWithData(c, gin.H{"progress": row})
}

func (h *LogArchive) Download(c *gin.Context) {
	if contextx.GetRequestUser(c) != service.SystemUsername && h.permissions == nil {
		response.FailWithMessage(c, "需要归档所属目录或函数的管理权限")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.FailWithMessage(c, "无效的归档批次")
		return
	}
	if err := h.authorizeBatch(c, id); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	url, err := h.service.DownloadURL(contextx.ToContext(c), id)
	if err != nil {
		response.FailWithMessage(c, "下载日志归档失败: "+err.Error())
		return
	}
	response.OkWithData(c, gin.H{"download_url": url})
}

func (h *LogArchive) authorize(c *gin.Context, path string) error {
	if contextx.GetRequestUser(c) == service.SystemUsername {
		return nil
	}
	if path == "" || h.permissions == nil {
		return fmt.Errorf("需要归档所属目录或函数的管理权限")
	}
	return requireAccess(c, h.permissions, path, access.ActionAdmin)
}
func (h *LogArchive) authorizeBatch(c *gin.Context, id int64) error {
	path, err := h.service.ArchiveResourcePath(contextx.ToContext(c), id)
	if err != nil {
		return fmt.Errorf("归档不存在或不可访问")
	}
	selected := access.NormalizeResourcePath(c.Query("resource_path"))
	if selected != "" && path != selected && !strings.HasPrefix(path, selected+"/") {
		return fmt.Errorf("归档不属于当前目录或函数")
	}
	return h.authorize(c, path)
}
