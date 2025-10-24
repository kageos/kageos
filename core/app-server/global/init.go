package global

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/subject"
	"github.com/ai-agent-os/ai-agent-os/pkg/waiter"
)

// Init 初始化所有全局组件
func Init() {
	// 初始化 NATS 连接
	err := InitNats()
	if err != nil {
		panic(err)
	}

	subject.SetResponseNotifier(waiter.GetDefaultWaiter())
}

// Close 关闭所有全局组件
func Close() {
	CloseNats()
}
