package global

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/nats-io/nats.go"
)

// InitNats 初始化 NATS 连接
func InitNats() error {
	// 获取配置
	cfg := config.GetAppServerConfig()

	connect, err := nats.Connect(cfg.Nats.URL)
	if err != nil {
		return err
	}
	NatsConn = connect

	return nil
}

// CloseNats 关闭 NATS 连接
func CloseNats() {
	if NatsConn != nil {
		NatsConn.Close()
	}
}
