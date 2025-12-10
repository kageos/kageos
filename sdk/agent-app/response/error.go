package response

import "fmt"

// BizErrorf 设置业务错误信息，返回 Form 接口以支持链式调用（如 .Build()）
// 业务错误不应该返回 err，系统错误直接 return err
func (r *RunFunctionResp) BizErrorf(format string, a ...any) Form {
	message := fmt.Sprintf(format, a...)
	r.BizError = message
	// 返回 Form 接口以支持链式调用
	return r
}
