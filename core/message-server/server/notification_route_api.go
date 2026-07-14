package server

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/ginx/response"
	"gorm.io/gorm"
)

func (s *Server) listNotificationRoutes(c *gin.Context) {
	scopePath := msgrepo.NormalizeNotificationScopePath(c.Query("scope_path"))
	rows, err := s.messageRepo.ListNotificationRoutes(c.Request.Context(), scopePath)
	if err != nil {
		response.Internal(c, "获取目录通知路由失败: "+err.Error())
		return
	}
	list := make([]dto.MessageNotificationRouteInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, notificationRouteToInfo(row))
	}
	response.OkWithData(c, dto.MessageNotificationRouteListResp{List: list})
}

func (s *Server) listNotificationRouteSummary(c *gin.Context) {
	rootScopePath := msgrepo.NormalizeNotificationScopePath(c.Query("root_scope_path"))
	if rootScopePath == "" {
		response.BadRequest(c, "root_scope_path 不能为空")
		return
	}
	rows, err := s.messageRepo.ListNotificationRoutesByRoot(c.Request.Context(), rootScopePath)
	if err != nil {
		response.Internal(c, "获取目录通知路由摘要失败: "+err.Error())
		return
	}
	summaries := make(map[string]dto.MessageNotificationRoutePathSummary, len(rows))
	for _, row := range rows {
		info := notificationRouteToInfo(row)
		summary := summaries[info.ScopePath]
		if summary.ScopePath == "" {
			summary.ScopePath = info.ScopePath
		}
		summary.Routes = append(summary.Routes, info)
		summaries[info.ScopePath] = summary
	}
	response.OkWithData(c, dto.MessageNotificationRouteSummaryResp{Routes: summaries})
}

func (s *Server) upsertNotificationRoute(c *gin.Context) {
	if _, err := s.resolveInboxUsername(c); err != nil {
		response.Error(c, err)
		return
	}
	channel, err := normalizeNotificationChannelParam(firstNonEmptyStringForServer(c.Param("channel"), c.Query("channel")))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req dto.UpsertMessageNotificationRouteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if bodyChannel := strings.TrimSpace(req.Channel); bodyChannel != "" {
		normalized, err := normalizeNotificationChannelParam(bodyChannel)
		if err != nil {
			response.Error(c, err)
			return
		}
		if normalized != channel {
			response.BadRequest(c, "路径 channel 与请求体 channel 不一致")
			return
		}
	}
	scopePath := msgrepo.NormalizeNotificationScopePath(req.ScopePath)
	if scopePath == "" {
		response.BadRequest(c, "scope_path 不能为空")
		return
	}
	if s.notificationVault == nil {
		response.Internal(c, "通知密钥服务未初始化")
		return
	}

	existing, err := s.messageRepo.GetNotificationRoute(c.Request.Context(), scopePath, channel)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		response.Internal(c, "读取目录通知路由失败: "+err.Error())
		return
	}
	enabled := true
	deliveryType := "webhook"
	displayName := strings.TrimSpace(req.DisplayName)
	remark := ""
	webhookCipher := ""
	secretCipher := ""
	metadata := marshalStringMetadataForServer(req.Metadata)
	if existing != nil {
		enabled = existing.Enabled
		deliveryType = firstNonEmptyStringForServer(existing.DeliveryType, deliveryType)
		if displayName == "" {
			displayName = existing.DisplayName
		}
		remark = existing.Remark
		webhookCipher = existing.WebhookURLCipher
		secretCipher = existing.SecretCipher
		if metadata == "" {
			metadata = existing.Metadata
		}
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Remark != nil {
		remark = strings.TrimSpace(*req.Remark)
	}
	if strings.TrimSpace(req.DeliveryType) != "" {
		deliveryType = strings.TrimSpace(req.DeliveryType)
	}
	if deliveryType != "webhook" {
		response.BadRequest(c, "当前仅支持 webhook 投递类型")
		return
	}
	if req.ClearWebhookURL {
		webhookCipher = ""
	}
	if strings.TrimSpace(req.WebhookURL) != "" {
		webhookURL := strings.TrimSpace(req.WebhookURL)
		if err := validateNotificationWebhookURLForServer(channel, webhookURL); err != nil {
			response.Error(c, err)
			return
		}
		webhookCipher, err = s.notificationVault.Seal(webhookURL)
		if err != nil {
			response.Internal(c, "加密 webhook 地址失败: "+err.Error())
			return
		}
	}
	if req.ClearSecret {
		secretCipher = ""
	}
	if strings.TrimSpace(req.Secret) != "" {
		secretCipher, err = s.notificationVault.Seal(strings.TrimSpace(req.Secret))
		if err != nil {
			response.Internal(c, "加密 secret 失败: "+err.Error())
			return
		}
	}

	row, err := s.messageRepo.UpsertNotificationRoute(c.Request.Context(), &msgmodel.NotificationRouteSetting{
		ScopePath:        scopePath,
		ScopeType:        strings.TrimSpace(req.ScopeType),
		Channel:          channel,
		Enabled:          enabled,
		DeliveryType:     deliveryType,
		DisplayName:      displayName,
		Remark:           remark,
		WebhookURLCipher: webhookCipher,
		SecretCipher:     secretCipher,
		Metadata:         metadata,
	})
	if err != nil {
		response.Internal(c, "保存目录通知路由失败: "+err.Error())
		return
	}
	response.OkWithData(c, notificationRouteToInfo(row))
}

func (s *Server) deleteNotificationRoute(c *gin.Context) {
	if _, err := s.resolveInboxUsername(c); err != nil {
		response.Error(c, err)
		return
	}
	channel, err := normalizeNotificationChannelParam(c.Param("channel"))
	if err != nil {
		response.Error(c, err)
		return
	}
	scopePath := msgrepo.NormalizeNotificationScopePath(c.Query("scope_path"))
	if scopePath == "" {
		response.BadRequest(c, "scope_path 不能为空")
		return
	}
	if err := s.messageRepo.DeleteNotificationRoute(c.Request.Context(), scopePath, channel); err != nil {
		response.Internal(c, "删除目录通知路由失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (s *Server) testNotificationRoute(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	channel, err := normalizeNotificationChannelParam(c.Param("channel"))
	if err != nil {
		response.Error(c, err)
		return
	}
	scopePath := msgrepo.NormalizeNotificationScopePath(c.Query("scope_path"))
	if scopePath == "" {
		var req dto.UpsertMessageNotificationRouteReq
		if bindErr := c.ShouldBindJSON(&req); bindErr == nil {
			scopePath = msgrepo.NormalizeNotificationScopePath(req.ScopePath)
		}
	}
	if scopePath == "" {
		response.BadRequest(c, "scope_path 不能为空")
		return
	}
	row, err := s.messageRepo.GetNotificationRoute(c.Request.Context(), scopePath, channel)
	if err != nil || row == nil {
		response.NotFound(c, "目录通知路由不存在")
		return
	}
	if strings.TrimSpace(row.WebhookURLCipher) == "" {
		response.Conflict(c, "目录通知路由缺少 webhook 地址")
		return
	}
	webhookURL, err := s.notificationVault.Open(row.WebhookURLCipher)
	if err != nil {
		response.Internal(c, "解密 webhook 地址失败: "+err.Error())
		return
	}
	secret, err := s.notificationVault.Open(row.SecretCipher)
	if err != nil {
		response.Internal(c, "解密 secret 失败: "+err.Error())
		return
	}
	entry := &msgmodel.MessageEntry{
		CreatedAt:    time.Now(),
		From:         "system",
		RequestUser:  username,
		FullCodePath: scopePath,
		SourcePath:   scopePath,
		SourceTitle:  "Kageos 目录通知路由",
		SourceType:   "notification_route_test",
		Title:        "Kageos 目录通知测试",
		Content:      "如果你看到这张卡片，说明该服务树目录已经可以通过独立通知路由触达。后续来自该目录及其子目录的通知会优先使用这个路由。",
		ContentType:  "markdown",
		ThreadKey:    "notification_route_test:" + scopePath,
	}
	target := service.NotificationTarget{
		Kind:            service.NotificationTargetKindRoute,
		Recipient:       service.ResolvedRecipient{Username: username},
		AuthorizedUsers: []string{username},
		Channel:         channel,
		WebhookURL:      strings.TrimSpace(webhookURL),
		Secret:          strings.TrimSpace(secret),
		Metadata:        parseStringMetadataForServer(row.Metadata),
		RouteID:         row.ID,
		ScopePath:       row.ScopePath,
		ScopeType:       row.ScopeType,
	}
	card := service.DefaultNotificationCardBuilder{}.BuildNotificationCard(
		c.Request.Context(),
		entry,
		dto.MessageSendPayload{},
		target,
		service.NotificationCardBuildOptions{BaseURL: config.GetPublicSiteBaseURL()},
	)
	provider, err := notificationTestProvider(channel, s.cfg.GetNotificationWebhookTimeout())
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := provider.Deliver(c.Request.Context(), target, card); err != nil {
		if recordErr := s.messageRepo.RecordNotificationRouteDeliveryFailure(c.Request.Context(), row.ID, err.Error(), true); recordErr != nil {
			response.Internal(c, "测试通知发送失败: "+err.Error()+"；记录投递状态失败: "+recordErr.Error())
			return
		}
		response.Internal(c, "测试通知发送失败: "+err.Error())
		return
	}
	if err := s.messageRepo.RecordNotificationRouteDeliverySuccess(c.Request.Context(), row.ID, true); err != nil {
		response.Internal(c, "测试通知已发送，但记录投递状态失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.TestMessageNotificationChannelResp{Message: "测试通知已发送", Channel: channel})
}
