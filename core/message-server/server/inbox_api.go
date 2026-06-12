package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/ginx/response"
)

func (s *Server) listInboxMessages(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	status := strings.TrimSpace(c.Query("status"))
	threadKey := strings.TrimSpace(c.Query("thread_key"))
	offset := (page - 1) * pageSize

	list, total, err := s.messageRepo.ListInbox(c.Request.Context(), username, status, threadKey, offset, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取消息列表失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.MessageInboxListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (s *Server) listInboxThreads(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	status := strings.TrimSpace(c.Query("status"))
	offset := (page - 1) * pageSize

	list, total, err := s.messageRepo.ListInboxThreads(c.Request.Context(), username, status, offset, pageSize)
	if err != nil {
		response.FailWithMessage(c, "获取消息会话列表失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.MessageInboxThreadListResp{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (s *Server) getInboxMessage(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	messageID, err := parseMessageID(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	item, err := s.messageRepo.GetInboxMessage(c.Request.Context(), username, messageID)
	if err != nil {
		response.FailWithMessage(c, "消息不存在")
		return
	}
	response.OkWithData(c, item)
}

func (s *Server) getUnreadCount(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	count, err := s.messageRepo.CountUnread(c.Request.Context(), username)
	if err != nil {
		response.FailWithMessage(c, "获取未读数失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.MessageUnreadCountResp{UnreadCount: count})
}

func (s *Server) markInboxMessageRead(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	messageID, err := parseMessageID(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.messageRepo.MarkRead(c.Request.Context(), username, messageID); err != nil {
		response.FailWithMessage(c, "标记已读失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (s *Server) markAllInboxMessagesRead(c *gin.Context) {
	username, err := s.resolveInboxUsername(c)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	if err := s.messageRepo.MarkAllRead(c.Request.Context(), username); err != nil {
		response.FailWithMessage(c, "全部已读失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func (s *Server) resolveInboxUsername(c *gin.Context) (string, error) {
	_, meta, err := resolveRequestMessageMeta(c, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(meta.From) == "" {
		return "", fmt.Errorf("无法解析当前用户")
	}
	return strings.TrimSpace(meta.From), nil
}

func parseMessageID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("消息 ID 不正确")
	}
	return id, nil
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
