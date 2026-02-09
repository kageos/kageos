// Package v1 提供 app-runtime 的 NATS 处理层（类似 Gin 的 api/v1）。
// 本层只负责：解码 NATS 消息 -> 调用 service -> 编码响应，业务逻辑在 service 层。
package v1
