package server

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	msgmodel "github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/ginx/response"
	"gorm.io/gorm"
)

func (s *Server) listNotificationChannels(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	rows, err := s.messageRepo.ListNotificationChannels(c.Request.Context(), username)
	if err != nil {
		response.Internal(c, "获取通知配置失败: "+err.Error())
		return
	}
	list := make([]dto.MessageNotificationChannelInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, notificationChannelToInfo(row))
	}
	response.OkWithData(c, dto.MessageNotificationChannelListResp{List: list})
}

func (s *Server) upsertNotificationChannel(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	channel, err := normalizeNotificationChannelParam(firstNonEmptyStringForServer(c.Param("channel"), c.Query("channel")))
	if err != nil {
		response.Error(c, err)
		return
	}
	var req dto.UpsertMessageNotificationChannelReq
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
	if s.notificationVault == nil {
		response.Internal(c, "通知密钥服务未初始化")
		return
	}

	existing, err := s.messageRepo.GetNotificationChannel(c.Request.Context(), username, channel)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		response.Internal(c, "读取通知配置失败: "+err.Error())
		return
	}
	enabled := true
	deliveryType := "webhook"
	displayName := strings.TrimSpace(req.DisplayName)
	webhookCipher := ""
	secretCipher := ""
	metadata := marshalStringMetadataForServer(req.Metadata)
	if existing != nil {
		enabled = existing.Enabled
		deliveryType = firstNonEmptyStringForServer(existing.DeliveryType, deliveryType)
		if displayName == "" {
			displayName = existing.DisplayName
		}
		webhookCipher = existing.WebhookURLCipher
		secretCipher = existing.SecretCipher
		if metadata == "" {
			metadata = existing.Metadata
		}
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if strings.TrimSpace(req.DeliveryType) != "" {
		deliveryType = strings.TrimSpace(req.DeliveryType)
	}
	if deliveryType != "webhook" {
		response.Internal(c, "当前仅支持 webhook 投递类型")
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

	row, err := s.messageRepo.UpsertNotificationChannel(c.Request.Context(), &msgmodel.NotificationChannelSetting{
		OwnerUsername:    username,
		Channel:          channel,
		Enabled:          enabled,
		DeliveryType:     deliveryType,
		DisplayName:      displayName,
		WebhookURLCipher: webhookCipher,
		SecretCipher:     secretCipher,
		Metadata:         metadata,
	})
	if err != nil {
		response.Internal(c, "保存通知配置失败: "+err.Error())
		return
	}
	response.OkWithData(c, notificationChannelToInfo(row))
}

func (s *Server) deleteNotificationChannel(c *gin.Context) {
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
	if err := s.messageRepo.DeleteNotificationChannel(c.Request.Context(), username, channel); err != nil {
		response.Internal(c, "删除通知配置失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (s *Server) testNotificationChannel(c *gin.Context) {
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
	row, err := s.messageRepo.GetNotificationChannel(c.Request.Context(), username, channel)
	if err != nil || row == nil {
		response.NotFound(c, "通知配置不存在")
		return
	}
	if strings.TrimSpace(row.WebhookURLCipher) == "" {
		response.Internal(c, "通知配置缺少 webhook 地址")
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
		CreatedAt:          time.Now(),
		From:               "system",
		RequestUser:        username,
		FullCodePath:       "/" + username,
		SourcePath:         "/" + username,
		SourceTitle:        "Kageos 通知配置",
		SourceType:         "notification_test",
		Title:              "Kageos 通知测试",
		Content:            "如果你看到这张卡片，说明 Kageos 已经可以通过该渠道触达你。后续业务、定时任务和 Agent 会话通知会使用同一张标准卡片携带目录、任务和会话上下文。",
		ContentType:        "markdown",
		WorkspaceSessionID: "",
	}
	target := service.NotificationTarget{
		Kind:            service.NotificationTargetKindUser,
		Recipient:       service.ResolvedRecipient{Username: username},
		AuthorizedUsers: []string{username},
		Channel:         channel,
		WebhookURL:      strings.TrimSpace(webhookURL),
		Secret:          strings.TrimSpace(secret),
		Metadata:        parseStringMetadataForServer(row.Metadata),
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
		if recordErr := s.messageRepo.RecordNotificationChannelDeliveryFailure(c.Request.Context(), username, channel, err.Error(), true); recordErr != nil {
			response.Internal(c, "测试通知发送失败: "+err.Error()+"；记录投递状态失败: "+recordErr.Error())
			return
		}
		response.Internal(c, "测试通知发送失败: "+err.Error())
		return
	}
	if err := s.messageRepo.RecordNotificationChannelDeliverySuccess(c.Request.Context(), username, channel, true); err != nil {
		response.Internal(c, "测试通知已发送，但记录投递状态失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.TestMessageNotificationChannelResp{Message: "测试通知已发送", Channel: channel})
}
