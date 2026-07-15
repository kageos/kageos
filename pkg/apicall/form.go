package apicall

import (
	"context"
)

// CallFormAPI 调用 Form API（泛型版本，支持任意请求和响应类型）
// TReq: 请求类型（可以是值类型或指针类型）
// TResp: 响应类型（指针类型，如 *MyFormResp）
// ctx: 上下文（从 ctx 中提取 token、trace_id、request_user）
// formPath: Form 函数路径（full-code-path），例如：/system/tools/table/inspect.form
// req: Form 提交请求（任意类型）
// 返回: TResp（指针类型）
func CallFormAPI[TReq, TResp any](ctx context.Context, formPath string, req TReq) (TResp, error) {
	path := buildWorkspaceFunctionPath("/workspace/api/v1/form/submit", formPath)
	return PostAPI[TReq, TResp](ctx, path, req)
}
