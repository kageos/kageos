package dto

// MessageSendPayload 消息发送载荷（SDK 发到 NATS，消费方解析后按渠道投递）
// 发送方只关心：谁发的、发给谁（用户/部门）、来源目录、内容；渠道由消息服务内部决定
type MessageSendPayload struct {
	From            string `json:"from"`              // 发送人（如 request_user 或 system，定时任务等无用户场景可为空）
	FullCodePath    string `json:"full_code_path"`   // 来源目录/函数路径，如 /luobei/example/inventory/stock_in，便于知道是哪个目录发的
	ToUsers         string `json:"to_users"`         // 接收用户，逗号分隔，如 "zhangsan,lisi"
	ToDepartments   string `json:"to_departments"`   // 接收部门（full_code_path），逗号分隔
	Title           string `json:"title"`            // 标题/摘要（可选）
	Content         string `json:"content"`           // 正文
	ContentType     string `json:"content_type"`      // 内容类型：text | html | markdown
}
