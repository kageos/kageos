package scheduledsdk

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/auth"
)

// IssueToken 在 Worker 真正开始执行后签发本次运行使用的内存 Token。
// ExecutionRequestedEvent.Token 禁止 JSON 序列化，因此该 Token 不会进入 outbox。
func (e *ExecutionRequestedEvent) IssueToken(ttl time.Duration) error {
	if e == nil {
		return fmt.Errorf("定时任务事件不能为空")
	}
	if ttl <= 0 {
		ttl = auth.DefaultScheduledTokenTTL
	}
	token, err := auth.NewJWTService().GenerateScheduledTokenWithContextTTL(auth.UserTokenContext{
		Username:           strings.TrimSpace(e.RequestUser),
		Email:              strings.TrimSpace(e.Metadata[MetadataRequestEmail]),
		DepartmentFullPath: strings.TrimSpace(e.RequestUserDept),
		LeaderUsername:     strings.TrimSpace(e.Metadata[MetadataLeaderUsername]),
	}, e.TaskID, e.ExecutionID, ttl)
	if err != nil {
		return err
	}
	e.Token = token
	return nil
}
