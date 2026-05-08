# 修改类型：新增消息通知

优先使用 SDK 的 `ctx.SendMessage(...)` 或平台 `/system/openapi/message` 能力。业务代码只放业务触发点，不自建消息表或通知系统。发送失败是否阻断主流程要在方案中说明。
