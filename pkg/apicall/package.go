// Package apicall 提供内部服务间 HTTP API 的轻量封装。
//
// 分层约定：
//   - transport.go / methods.go 负责通用 HTTP 请求与响应解析。
//   - query.go / paths.go 负责复用型 query 与 path builder。
//   - workspace.go / hr.go / storage.go / form.go 提供领域化的 typed wrapper。
//
// 新增调用时，优先使用已有的语义化 helper，例如分页、full_code_path、status 等；
// 只有在没有合适语义 helper 时，才回退到基础 key/value builder，避免把样板参数拼装重新散回各个调用点。
package apicall
