package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/msgx"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

type PermissionNotification struct {
	ToUser       string
	Actor        string
	TenantUser   string
	App          string
	ResourcePath string
	Title        string
	Message      string
}

type PermissionNotifier interface {
	Notify(ctx context.Context, notification PermissionNotification) error
}

type permissionMessagePublisher interface {
	PublishMsg(msg *nats.Msg) error
}

type NATSPermissionNotifier struct {
	publisher permissionMessagePublisher
}

func NewNATSPermissionNotifier(publisher permissionMessagePublisher) *NATSPermissionNotifier {
	return &NATSPermissionNotifier{publisher: publisher}
}

func (n *NATSPermissionNotifier) Notify(ctx context.Context, notification PermissionNotification) error {
	if n == nil || n.publisher == nil {
		return fmt.Errorf("permission notification publisher is not initialized")
	}
	notification.ToUser = strings.ToLower(strings.TrimSpace(notification.ToUser))
	notification.Actor = strings.ToLower(strings.TrimSpace(notification.Actor))
	notification.ResourcePath = access.NormalizeResourcePath(notification.ResourcePath)
	notification.Title = strings.TrimSpace(notification.Title)
	notification.Message = strings.TrimSpace(notification.Message)
	if notification.ToUser == "" || notification.Title == "" || notification.Message == "" {
		return fmt.Errorf("permission notification recipient, title and message are required")
	}

	envelope := &dto.MessageSendEnvelope{
		Meta: dto.MessageSendMeta{
			From:         notification.Actor,
			RequestUser:  notification.Actor,
			FullCodePath: notification.ResourcePath,
			SourceType:   "permission",
			SourceRef:    notification.ResourcePath,
			SourcePath:   notification.ResourcePath,
			SourceTitle:  notification.Title,
			ThreadKey:    permissionNotificationThreadKey(notification),
		},
		Message: dto.MessageSendPayload{
			ToUsers:     notification.ToUser,
			Title:       notification.Title,
			Content:     notification.Message,
			ContentType: "markdown",
		},
	}
	msg, err := msgx.BuildJSONRequest(ctx, subjects.MessageSendCommandSubject, envelope)
	if err != nil {
		return err
	}
	return n.publisher.PublishMsg(msg)
}

func permissionNotificationThreadKey(notification PermissionNotification) string {
	return strings.Join([]string{
		"permission",
		strings.TrimSpace(notification.TenantUser),
		strings.TrimSpace(notification.App),
		strings.ToLower(strings.TrimSpace(notification.ToUser)),
	}, ":")
}

func (s *PermissionService) notifyPermissionRequestDecision(
	ctx context.Context,
	request *model.WorkspacePermissionRequest,
) {
	if request == nil {
		return
	}
	roleName := permissionRoleName(access.NormalizeRoleCode(access.RoleCode(request.RequestedRole)))
	expiresAt := permissionExpiresAtLabel(request.RequestedExpires)
	comment := strings.TrimSpace(request.ReviewComment)
	if comment == "" {
		comment = "无"
	}

	title := "权限申请已驳回"
	result := "未通过"
	actionHint := "如仍需使用，请根据审批意见调整后重新申请。"
	if request.Status == model.PermissionRequestStatusApproved {
		title = "权限申请已通过"
		result = "已通过"
		actionHint = "你现在可以进入对应目录使用相关功能。"
	}
	message := fmt.Sprintf(
		"你的权限申请%s。\n\n- 资源：`%s`\n- 角色：%s\n- 审批人：`%s`\n- 有效期：%s\n- 审批意见：%s\n\n%s",
		result,
		access.NormalizeResourcePath(request.ResourcePath),
		roleName,
		strings.TrimSpace(request.ReviewedBy),
		expiresAt,
		comment,
		actionHint,
	)
	s.sendPermissionNotification(ctx, PermissionNotification{
		ToUser:       request.Requester,
		Actor:        request.ReviewedBy,
		TenantUser:   request.TenantUser,
		App:          request.App,
		ResourcePath: request.ResourcePath,
		Title:        title,
		Message:      message,
	})
}

func (s *PermissionService) notifyRoleGranted(ctx context.Context, req access.GrantRoleRequest) {
	principal := access.NormalizePrincipal(req.Principal)
	if principal.Type != access.PrincipalUser {
		return
	}
	message := fmt.Sprintf(
		"`%s` 已为你授予新的访问权限。\n\n- 资源：`%s`\n- 角色：%s\n- 有效期：%s\n\n你现在可以进入对应目录使用相关功能。",
		strings.TrimSpace(req.CreatedBy),
		access.NormalizeResourcePath(req.ResourcePath),
		permissionRoleName(access.NormalizeRoleCode(req.RoleCode)),
		permissionExpiresAtLabel(req.ExpiresAt),
	)
	s.sendPermissionNotification(ctx, PermissionNotification{
		ToUser:       principal.Key,
		Actor:        req.CreatedBy,
		TenantUser:   req.TenantUser,
		App:          req.App,
		ResourcePath: req.ResourcePath,
		Title:        "你已获得新的权限",
		Message:      message,
	})
}

func (s *PermissionService) notifyBatchRolesGranted(ctx context.Context, grants []access.GrantRoleRequest) {
	byUser := make(map[string][]access.GrantRoleRequest)
	for _, grant := range grants {
		principal := access.NormalizePrincipal(grant.Principal)
		if principal.Type != access.PrincipalUser || principal.Key == "" {
			continue
		}
		byUser[principal.Key] = append(byUser[principal.Key], grant)
	}
	users := make([]string, 0, len(byUser))
	for username := range byUser {
		users = append(users, username)
	}
	sort.Strings(users)
	for _, username := range users {
		userGrants := byUser[username]
		if len(userGrants) == 0 {
			continue
		}
		lines := make([]string, 0, len(userGrants))
		seen := make(map[string]struct{}, len(userGrants))
		for _, grant := range userGrants {
			line := fmt.Sprintf(
				"- `%s`：%s（%s）",
				access.NormalizeResourcePath(grant.ResourcePath),
				permissionRoleName(access.NormalizeRoleCode(grant.RoleCode)),
				permissionExpiresAtLabel(grant.ExpiresAt),
			)
			if _, exists := seen[line]; exists {
				continue
			}
			seen[line] = struct{}{}
			lines = append(lines, line)
		}
		first := userGrants[0]
		message := fmt.Sprintf(
			"`%s` 已为你授予以下访问权限：\n\n%s\n\n你现在可以进入对应目录使用相关功能。",
			strings.TrimSpace(first.CreatedBy),
			strings.Join(lines, "\n"),
		)
		s.sendPermissionNotification(ctx, PermissionNotification{
			ToUser:       username,
			Actor:        first.CreatedBy,
			TenantUser:   first.TenantUser,
			App:          first.App,
			ResourcePath: first.ResourcePath,
			Title:        "你已获得新的权限",
			Message:      message,
		})
	}
}

func (s *PermissionService) sendPermissionNotification(ctx context.Context, notification PermissionNotification) {
	if s == nil || s.permissionNotifier == nil {
		return
	}
	if err := s.permissionNotifier.Notify(ctx, notification); err != nil {
		logger.Warnf(
			ctx,
			"[Permission] send notification failed: to=%s resource=%s title=%s err=%v",
			notification.ToUser,
			notification.ResourcePath,
			notification.Title,
			err,
		)
	}
}

func permissionRoleName(role access.RoleCode) string {
	switch role {
	case access.RoleViewer:
		return "查看者"
	case access.RoleMember:
		return "成员"
	case access.RoleAdmin:
		return "管理员"
	case access.RoleOwner:
		return "拥有者"
	default:
		return string(role)
	}
}

func permissionExpiresAtLabel(expiresAt *time.Time) string {
	if expiresAt == nil {
		return "永久"
	}
	return expiresAt.Local().Format("2006-01-02 15:04")
}
