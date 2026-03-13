package natsx

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// Connect 创建带自动重连的 NATS 连接
// 休眠/网络中断后会自动重连，避免手动重启服务。
func Connect(url string) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1), // 无限重连
		nats.ReconnectWait(2 * time.Second),
		nats.ReconnectBufSize(8 * 1024 * 1024), // 8MB 重连缓冲

		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warnf(nil, "[NATS] Disconnected: %v, will auto-reconnect...", err)
			} else {
				logger.Warnf(nil, "[NATS] Disconnected (graceful), will auto-reconnect...")
			}
		}),

		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infof(nil, "[NATS] Reconnected to %s", nc.ConnectedUrl())
		}),

		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Warnf(nil, "[NATS] Connection closed permanently")
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, err
	}

	logger.Infof(nil, "[NATS] Connected to %s (auto-reconnect enabled)", url)
	return conn, nil
}
