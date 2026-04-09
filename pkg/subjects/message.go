package subjects

import "time"

const (
	MessageTypeStatusShutdown    = "shutdown"
	MessageTypeStatusDiscovery   = "discovery"
	MessageTypeStatusStartup     = "startup"
	MessageTypeStatusClose       = "close"
	MessageTypeStatusOnAppUpdate = "onAppUpdate"

	MessageTypeUpdateCallbackRequest = "update_callback_request"
)

// Message 为 runtime / app 生命周期与控制链路共用的统一消息体。
type Message struct {
	ErrorMsg  string      `json:"error_msg"`
	Type      string      `json:"type"`
	User      string      `json:"user"`
	App       string      `json:"app"`
	Version   string      `json:"version"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
