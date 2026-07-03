package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DefaultMessageActionTokenTTL = 72 * time.Hour
	messageActionTokenPrefix     = "kat_"
	messageActionDefaultAction   = "reply"
)

type CreateActionTokenInput struct {
	MessageID          int64
	RecipientUsername  string
	AuthorizedUsers    []string
	Channel            string
	AllowedActions     []string
	ExpiresAt          time.Time
	WorkspaceSessionID string
	ThreadKey          string
	SourcePath         string
	TraceID            string
}

func (r *MessageRepository) CreateActionToken(ctx context.Context, input CreateActionTokenInput) (string, *model.MessageActionToken, error) {
	if r == nil || r.db == nil {
		return "", nil, fmt.Errorf("message repository is nil")
	}
	if input.MessageID <= 0 {
		return "", nil, fmt.Errorf("message_id is required")
	}
	recipient := strings.TrimSpace(input.RecipientUsername)
	if recipient == "" {
		return "", nil, fmt.Errorf("recipient_username is required")
	}
	rawToken, err := newMessageActionToken()
	if err != nil {
		return "", nil, err
	}
	expiresAt := input.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(DefaultMessageActionTokenTTL)
	}
	allowedActions := normalizeAllowedActions(input.AllowedActions)
	authorizedUsers := normalizeActionAuthorizedUsers(append(input.AuthorizedUsers, recipient))
	row := &model.MessageActionToken{
		TokenHash:          hashMessageActionToken(rawToken),
		MessageID:          input.MessageID,
		RecipientUsername:  recipient,
		AuthorizedUsers:    strings.Join(authorizedUsers, ","),
		Channel:            strings.TrimSpace(input.Channel),
		AllowedActions:     strings.Join(allowedActions, ","),
		Status:             string(dto.MessageActionTokenStatusOpen),
		ExpiresAt:          expiresAt,
		WorkspaceSessionID: strings.TrimSpace(input.WorkspaceSessionID),
		ThreadKey:          strings.TrimSpace(input.ThreadKey),
		SourcePath:         strings.TrimSpace(input.SourcePath),
		TraceID:            strings.TrimSpace(input.TraceID),
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return "", nil, err
	}
	return rawToken, row, nil
}

func (r *MessageRepository) GetActionView(ctx context.Context, rawToken, mobileAskURL string, viewerUsername ...string) (*dto.MessageActionViewResp, error) {
	row, err := r.getActionToken(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	actingUser, err := authorizeMessageActionUser(row, firstActionViewer(viewerUsername...))
	if err != nil {
		return nil, err
	}
	item, err := r.GetInboxMessage(ctx, actingUser, row.MessageID)
	if err != nil {
		return nil, err
	}
	thread := []dto.MessageInboxItem{*item}
	if strings.TrimSpace(item.ThreadKey) != "" {
		list, _, err := r.ListInbox(ctx, actingUser, InboxListFilter{ThreadKey: item.ThreadKey}, 0, 30)
		if err == nil && len(list) > 0 {
			thread = list
		}
	}
	allowedActions := splitAllowedActions(row.AllowedActions)
	status := effectiveActionTokenStatus(row, time.Now())
	return &dto.MessageActionViewResp{
		TokenStatus:       status,
		RecipientUser:     actingUser,
		Channel:           strings.TrimSpace(row.Channel),
		AuthenticatedUser: strings.TrimSpace(firstActionViewer(viewerUsername...)),
		AllowedActions:    allowedActions,
		CanReply:          status == string(dto.MessageActionTokenStatusOpen) && containsAllowedAction(allowedActions, messageActionDefaultAction),
		ExpiresAt:         row.ExpiresAt,
		Message:           *item,
		Thread:            thread,
		MobileAskURL:      strings.TrimSpace(mobileAskURL),
		WorkspaceSession:  row.WorkspaceSessionID,
		SubmittedAt:       row.UsedAt,
		ReplyMessageID:    row.ReplyMessageID,
	}, nil
}

func (r *MessageRepository) SubmitActionReply(ctx context.Context, rawToken, content, action string, viewerUsername ...string) (*dto.MessageActionReplyResp, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("回复内容不能为空")
	}
	if len([]rune(content)) > 8000 {
		return nil, fmt.Errorf("回复内容过长")
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = messageActionDefaultAction
	}

	var resp dto.MessageActionReplyResp
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tokenRow model.MessageActionToken
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", hashMessageActionToken(rawToken)).
			First(&tokenRow)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				return fmt.Errorf("处理链接无效")
			}
			return result.Error
		}
		actingUser, err := authorizeMessageActionUser(&tokenRow, firstActionViewer(viewerUsername...))
		if err != nil {
			return err
		}
		if tokenRow.Status != string(dto.MessageActionTokenStatusOpen) {
			return fmt.Errorf("消息已处理或链接已失效")
		}
		now := time.Now()
		if !tokenRow.ExpiresAt.IsZero() && now.After(tokenRow.ExpiresAt) {
			if err := tx.Model(&model.MessageActionToken{}).
				Where("id = ?", tokenRow.ID).
				Update("status", string(dto.MessageActionTokenStatusExpired)).Error; err != nil {
				return err
			}
			return fmt.Errorf("处理链接已过期")
		}
		allowedActions := splitAllowedActions(tokenRow.AllowedActions)
		if !containsAllowedAction(allowedActions, action) {
			return fmt.Errorf("当前链接不允许执行该动作")
		}

		var original model.MessageEntry
		if err := tx.Where("id = ? AND deleted_at IS NULL", tokenRow.MessageID).First(&original).Error; err != nil {
			return fmt.Errorf("原始消息不存在")
		}
		if err := tx.Model(&model.MessageRecipient{}).
			Where("username = ? AND message_id = ?", actingUser, tokenRow.MessageID).
			Update("read_at", now).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.MessageActionToken{}).
			Where("id = ?", tokenRow.ID).
			Updates(map[string]interface{}{
				"status":           string(dto.MessageActionTokenStatusSubmitted),
				"used_at":          now,
				"reply_message_id": int64(0),
			}).Error; err != nil {
			return err
		}
		workspaceSessionID := firstNonEmptyStringForRepository(original.WorkspaceSessionID, tokenRow.WorkspaceSessionID)
		resp = dto.MessageActionReplyResp{
			Status:             string(dto.MessageActionTokenStatusSubmitted),
			SubmittedAt:        now,
			Channel:            strings.TrimSpace(tokenRow.Channel),
			SourcePath:         firstNonEmptyStringForRepository(original.SourcePath, original.FullCodePath, original.SourceParentPath),
			FullCodePath:       firstNonEmptyStringForRepository(original.SourcePath, original.FullCodePath, original.SourceParentPath),
			WorkspaceSessionID: workspaceSessionID,
			WorkstationDraft:   buildMobileWorkstationDraft(original, content, actingUser, tokenRow.Channel),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *MessageRepository) UpdateActionWorkspaceSession(ctx context.Context, rawToken, sessionID string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("message repository is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	result := r.db.WithContext(ctx).Model(&model.MessageActionToken{}).
		Where("token_hash = ?", hashMessageActionToken(rawToken)).
		Update("workspace_session_id", sessionID)
	return result.Error
}

func (r *MessageRepository) getActionToken(ctx context.Context, rawToken string) (*model.MessageActionToken, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("message repository is nil")
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, fmt.Errorf("处理链接缺少 token")
	}
	var row model.MessageActionToken
	result := r.db.WithContext(ctx).Where("token_hash = ?", hashMessageActionToken(rawToken)).First(&row)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("处理链接无效")
		}
		return nil, result.Error
	}
	return &row, nil
}

func newMessageActionToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate action token: %w", err)
	}
	return messageActionTokenPrefix + base64.RawURLEncoding.EncodeToString(buf[:]), nil
}

func hashMessageActionToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func normalizeAllowedActions(actions []string) []string {
	out := make([]string, 0, len(actions)+1)
	seen := map[string]struct{}{}
	for _, action := range actions {
		action = strings.ToLower(strings.TrimSpace(action))
		if action == "" {
			continue
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		out = append(out, action)
	}
	if len(out) == 0 {
		out = append(out, messageActionDefaultAction)
	}
	return out
}

func splitAllowedActions(raw string) []string {
	return normalizeAllowedActions(strings.Split(raw, ","))
}

func containsAllowedAction(actions []string, action string) bool {
	action = strings.ToLower(strings.TrimSpace(action))
	for _, candidate := range actions {
		if candidate == action {
			return true
		}
	}
	return false
}

func firstActionViewer(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func normalizeActionAuthorizedUsers(users []string) []string {
	out := make([]string, 0, len(users))
	seen := map[string]struct{}{}
	for _, user := range users {
		user = strings.TrimSpace(user)
		if user == "" {
			continue
		}
		if _, ok := seen[user]; ok {
			continue
		}
		seen[user] = struct{}{}
		out = append(out, user)
	}
	return out
}

func actionAuthorizedUsers(row *model.MessageActionToken) []string {
	if row == nil {
		return nil
	}
	users := strings.Split(row.AuthorizedUsers, ",")
	users = append(users, row.RecipientUsername)
	return normalizeActionAuthorizedUsers(users)
}

func authorizeMessageActionUser(row *model.MessageActionToken, viewerUsername string) (string, error) {
	if row == nil {
		return "", fmt.Errorf("处理链接无效")
	}
	viewerUsername = strings.TrimSpace(viewerUsername)
	if viewerUsername == "" {
		return "", fmt.Errorf("请先登录 kageos 后处理该消息")
	}
	for _, user := range actionAuthorizedUsers(row) {
		if user == viewerUsername {
			return viewerUsername, nil
		}
	}
	return "", fmt.Errorf("当前登录用户无权处理此消息")
}

func effectiveActionTokenStatus(row *model.MessageActionToken, now time.Time) string {
	if row == nil {
		return string(dto.MessageActionTokenStatusRevoked)
	}
	status := strings.TrimSpace(row.Status)
	if status == "" {
		status = string(dto.MessageActionTokenStatusOpen)
	}
	if status == string(dto.MessageActionTokenStatusOpen) && !row.ExpiresAt.IsZero() && now.After(row.ExpiresAt) {
		return string(dto.MessageActionTokenStatusExpired)
	}
	return status
}

func replyTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "移动端回复"
	}
	return "回复：" + title
}

func buildMobileWorkstationDraft(original model.MessageEntry, replyContent, recipientUser, channel string) string {
	var lines []string
	lines = append(lines,
		"【移动端消息处理上下文】",
		"入口：kageos Pocket",
	)
	if channel = strings.TrimSpace(channel); channel != "" {
		lines = append(lines, "通知渠道："+channel)
	}
	if recipientUser = strings.TrimSpace(recipientUser); recipientUser != "" {
		lines = append(lines, "处理用户："+recipientUser)
	}
	lines = append(lines,
		"移动端限制：用户通过移动端消息入口提交本次处理，不在电脑端工作台前，也看不到本轮工作台回复内容。",
		"触达限制：用户只能收到 send_notification 投递的消息通知；凡是需要让用户知道的处理结果、确认问题或下一步，都必须通过 send_notification 发送。",
		"最终触达动作：完成本次处理后，必须主动调用 send_notification 给处理用户发送一条简短结论；仅写工作台回复不算完成通知。",
		"send_notification 参数要求：to_users 必须填写处理用户，不要省略；正文只写业务结论、关键结果、下一步或需要用户继续确认的问题。",
		"输出格式：面向移动端用户的最终回复和 send_notification.message 必须使用 Markdown 格式；content_type 使用 markdown 或省略默认 markdown。",
		"Markdown 要简短适合手机阅读，可用短段落、列表和加粗关键结论；不要使用 HTML、富文本，也不要把整段回复包进代码块。",
		"通知正文禁止包含思考过程、工具调用过程、长日志、完整工作台输出。",
		"通知只发一次；除非发现高优先级异常，不要额外发送多条消息。",
		"工作台普通回复只用于简短留档，不能替代消息通知。",
		"",
		"我正在通过 kageos Pocket 处理一条业务消息，请根据上下文继续协助。",
	)
	if original.Title != "" {
		lines = append(lines, "", "原消息标题："+original.Title)
	}
	source := firstNonEmptyStringForRepository(original.SourceTitle, original.SourcePath, original.FullCodePath, original.SourceParentPath)
	if source != "" {
		lines = append(lines, "来源："+source)
	}
	if original.WorkspaceSessionID != "" {
		lines = append(lines, "关联会话："+original.WorkspaceSessionID)
	}
	if original.ThreadKey != "" {
		lines = append(lines, "消息线程："+original.ThreadKey)
	}
	if original.SourceRef != "" {
		lines = append(lines, "来源引用："+original.SourceRef)
	}
	if strings.TrimSpace(original.Content) != "" {
		lines = append(lines, "", "原消息内容：", strings.TrimSpace(original.Content))
	}
	lines = append(lines, "", "我的回复/处理意图：", strings.TrimSpace(replyContent))
	return strings.Join(lines, "\n")
}
