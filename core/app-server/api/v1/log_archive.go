package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type LogArchive struct{ service *service.LogArchiveService }

func NewLogArchive(archiveService *service.LogArchiveService) *LogArchive {
	return &LogArchive{service: archiveService}
}

func (h *LogArchive) List(c *gin.Context) {
	if contextx.GetRequestUser(c) != service.SystemUsername {
		response.FailWithMessage(c, "仅 system 超管可查看日志归档")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	rows, total, err := h.service.List(contextx.ToContext(c), page, pageSize)
	if err != nil {
		response.FailWithMessage(c, "查询日志归档失败: "+err.Error())
		return
	}
	cfg := h.service.Config()
	response.OkWithData(c, gin.H{"list": rows, "total": total, "retention_days": cfg.RetentionDays, "cron_expr": cfg.CronExpr, "timezone": cfg.Timezone})
}
