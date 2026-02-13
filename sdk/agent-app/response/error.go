package response

import "fmt"

// 错误格式约定（由 SDK handle 根据是否含 [系统错误] 区分，用于决定是否需要智能体介入）：
//
//   - 业务错误：直接 return fmt.Errorf("具体消息")，无需加标签。
//     前端/框架按 ErrCode=-1 展示即可，无需智能体介入，一般也不打详细日志。
//
//   - 系统错误：return fmt.Errorf("[系统错误]-[DoXxx]： 具体消息")。
//     加 [系统错误] 是为了区分「需要智能体介入处理」与「可以不管」的错误：
//     只有带此标签的错误会触发智能体排查；业务错误不触发。
//     系统错误必须在打日志时带上足够上下文（req、model 等用 %+v），
//     方便后续根据参数定位问题；日志内容应包含 "[系统错误]" 便于检索。

// BizErrorf 设置业务错误信息，返回 Form 接口以支持链式调用（如 .Build()）
func (r *RunFunctionResp) BizErrorf(format string, a ...any) Form {
	message := fmt.Sprintf(format, a...)
	r.BizError = message
	// 返回 Form 接口以支持链式调用
	return r
}
