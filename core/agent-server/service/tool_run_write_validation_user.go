package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/apicall"
)

type runWriteUserWidgetValidator struct {
	resolveUsers runWriteUserResolver
}

func (runWriteUserWidgetValidator) WidgetTypes() []string {
	return []string{widget.TypeUser, widget.TypeUsers}
}

func (v runWriteUserWidgetValidator) ValidateBatch(ctx context.Context, items []runWriteFieldValue) []runWriteValidationIssue {
	var issues []runWriteValidationIssue
	userRefs := make([]runWriteUserRef, 0)
	for _, item := range items {
		values, invalid := runWriteStringValues(item.Value)
		if invalid {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 是用户字段，必须传真实 username 字符串；多用户字段使用逗号分隔 username。", runWriteFieldDisplayName(item.Field), item.Path),
			})
			continue
		}
		if strings.TrimSpace(item.Field.Widget.Type) == widget.TypeUser && len(values) > 1 {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 是单用户字段，只能传一个真实 username。", runWriteFieldDisplayName(item.Field), item.Path),
			})
			continue
		}
		for _, username := range values {
			userRefs = append(userRefs, runWriteUserRef{Username: username, Path: item.Path, Field: item.Field})
		}
	}

	if len(userRefs) == 0 || v.resolveUsers == nil {
		return issues
	}
	usernames := uniqueRunWriteUsernames(userRefs)
	existing, err := v.resolveUsers(ctx, usernames)
	if err != nil {
		issues = append(issues, runWriteValidationIssue{
			Kind:    runWriteIssueUser,
			Message: "用户字段校验失败：无法查询平台用户: " + err.Error(),
		})
		return issues
	}
	for _, ref := range userRefs {
		if _, ok := existing[ref.Username]; !ok {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 包含不存在或当前企业不可见的用户 %q。", runWriteFieldDisplayName(ref.Field), ref.Path, ref.Username),
			})
		}
	}
	return issues
}

func resolveRunWriteUsers(ctx context.Context, usernames []string) (map[string]struct{}, error) {
	users, err := apicall.GetUsersByUsernames(ctx, usernames)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(users))
	for _, user := range users {
		username := strings.TrimSpace(user.Username)
		if username != "" {
			out[username] = struct{}{}
		}
	}
	return out, nil
}

func isRunWriteUserWidget(field *widget.Field) bool {
	switch strings.TrimSpace(field.Widget.Type) {
	case widget.TypeUser, widget.TypeUsers:
		return true
	default:
		return false
	}
}

func runWriteStringValues(value interface{}) ([]string, bool) {
	switch v := value.(type) {
	case string:
		return splitRunWriteCSV(v), false
	case []string:
		return cleanRunWriteStrings(v), false
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, true
			}
			values = append(values, s)
		}
		return cleanRunWriteStrings(values), false
	default:
		return nil, true
	}
}

func splitRunWriteCSV(value string) []string {
	return cleanRunWriteStrings(strings.Split(value, ","))
}

func cleanRunWriteStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func uniqueRunWriteUsernames(refs []runWriteUserRef) []string {
	seen := map[string]struct{}{}
	for _, ref := range refs {
		username := strings.TrimSpace(ref.Username)
		if username == "" {
			continue
		}
		seen[username] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for username := range seen {
		out = append(out, username)
	}
	sort.Strings(out)
	return out
}
