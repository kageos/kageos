package widget

import "errors"

func init() {
	RegisterWidgetValidator(TypeUser, validateUserWidget)
}

// User 用户选择器组件
//
// 功能：
// - 支持用户搜索和选择
// - 支持动态默认值函数：Me()（当前登录用户）、MyLeader()（当前用户的上级领导）
//
// 使用示例：
//
//	widget:"name:预约人;type:user;render_default:Me()"
//	widget:"name:审批人;type:user;render_default:MyLeader()"
//
// 动态默认值函数说明：
//   - Me(): 自动填充当前登录用户的用户名，用户无需手动选择
//     适用于：预约人、创建人、负责人等字段，大部分情况下默认是自己
//   - MyLeader(): 自动填充当前登录用户的上级领导用户名
//     适用于：审批人、抄送人、上级领导等字段，需要默认选择上级时使用
//
// 注意：
//   - render_default 参数支持函数调用（如 Me()、MyLeader()）
//   - 如果用户未登录，Me() 会返回 null
//   - 如果用户没有上级领导，MyLeader() 会返回 null
//   - disabled: 是否禁用（只读模式，Form 中展示但不可编辑）
//
// 校验规则：
// - 注册的是本文件的 validateUserWidget；
// - Go 字段必须是 string 或 *string；
// - disabled 必须显式写 true/false；
// - 字段值存储用户名/用户标识字符串；
// - render_default 里的 Me()/MyLeader() 在前端动态默认值逻辑中解析，SDK 只透传配置。
type User struct {
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持函数调用 Me()（当前登录用户）、MyLeader()（当前用户的上级领导）
	Disabled      bool   `json:"disabled,omitempty"`       // 是否禁用（只读模式）
}

func (u *User) Config() interface{} {
	return u
}

func (u *User) Type() string {
	return TypeUser
}

func (u *User) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"beiluo"`},
	}
	if u.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: u.RenderDefault})
	}
	if u.Disabled && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "disabled", Value: "true"})
	}
	return facts
}

// validateUserWidget 校验 user 的单用户字符串协议。
//
// 用户选择器存储的是用户名/用户标识，不允许绑定 int 或结构体。
func validateUserWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringLikeGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "disabled"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func newUser(widgetParsed map[string]string) *User {
	user := &User{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		user.RenderDefault = defaultValue
	}
	if disabled, exists := widgetParsed["disabled"]; exists {
		user.Disabled = disabled == "true"
	}

	return user
}
