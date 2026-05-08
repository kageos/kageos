package widget

import (
	"errors"
	"fmt"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeUsers, validateUsersWidget)
}

// Users 多用户选择器组件
//
// 功能：
// - 支持多个用户搜索和选择
// - 支持动态默认值函数：Me()（当前登录用户）、MyLeader()（当前用户的上级领导）
// - 值使用逗号分隔的字符串格式存储（如 "user1,user2,user3"）
//
// 使用示例：
//
//	widget:"name:审核人;type:users;render_default:Me()"
//	widget:"name:抄送人;type:users;render_default:MyLeader()"
//	widget:"name:管理员;type:users;max_count:5"
//
// 动态默认值函数说明：
//   - Me(): 自动填充当前登录用户的用户名，用户无需手动选择
//   - MyLeader(): 自动填充当前登录用户的上级领导用户名
//   - 支持多个默认值，用逗号分隔：Me(),MyLeader(),user2
//
// 注意：
//   - render_default 参数支持函数调用（如 Me()、MyLeader()）
//   - 如果用户未登录，Me() 会返回 null
//   - 如果用户没有上级领导，MyLeader() 会返回 null
//   - 值存储格式：逗号分隔的字符串（如 "user1,user2"），便于存储到数据库
//   - 前端会自动处理字符串和数组之间的转换
//
// 校验规则：
// - Go 字段可以是 string/*string，也可以是 []string/[N]string；
// - max_count 必须是非负整数；
// - 不允许 []int/[]struct，因为用户标识协议是字符串；
// - render_default 里的 Me()/MyLeader() 由前端动态默认值逻辑解析。
type Users struct {
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持函数调用 Me()、MyLeader()，多个值用逗号分隔
	MaxCount      int    `json:"max_count,omitempty"`      // 最大选择数量，0表示不限制
}

func (u *Users) Config() interface{} {
	return u
}

func (u *Users) Type() string {
	return TypeUsers
}

func (u *Users) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"beiluo,zhangsan"`},
	}
	if u.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: u.RenderDefault})
	}
	if u.MaxCount > 0 && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max_count", Value: fmt.Sprintf("%d", u.MaxCount)})
	}
	return facts
}

func newUsers(widgetParsed map[string]string) *Users {
	users := &Users{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		users.RenderDefault = defaultValue
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		// 解析最大选择数量，支持 "0" 或 "" 表示不限制
		if maxCount == "0" || maxCount == "" {
			users.MaxCount = 0 // 0表示不限制
		} else if val, err := strconv.Atoi(maxCount); err == nil && val > 0 {
			users.MaxCount = val
		}
	}

	return users
}

// validateUsersWidget 校验多用户组件的字符串协议。
//
// 推荐落库字段使用 string，值为逗号分隔用户名；如果只是接口 DTO，也可以使用 []string。
func validateUsersWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringOrStringSliceGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateNonNegativeIntTag(ctx, "max_count"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
