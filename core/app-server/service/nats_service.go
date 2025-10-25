package service

import (
	"fmt"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type NatsService struct {
	mu         sync.RWMutex
	natsIdMap  map[int64]*nats.Conn
	hostIdMap  map[int64]*nats.Conn
	subsByHost map[int64][]*nats.Subscription
	appRuntime *AppRuntime // 添加 AppRuntime 引用
}

// NewNatsServiceWithDB 使用指定的数据库连接创建 NATS 服务
func NewNatsServiceWithDB(db *gorm.DB, appRuntime *AppRuntime) *NatsService {
	hostRepo := repository.NewHostRepository(db)
	list, err := hostRepo.GetHostList()
	if err != nil {
		panic(err)
	}
	return newNatsServiceFromHostList(list, appRuntime)
}

// newNatsServiceFromHostList 从主机列表创建 NATS 服务
func newNatsServiceFromHostList(list []*model.Host, appRuntime *AppRuntime) *NatsService {
	natsIdMap := make(map[int64]*nats.Conn)
	hostIdMap := make(map[int64]*nats.Conn)
	subsByHost := make(map[int64][]*nats.Subscription)

	// 先创建 NatsService 实例
	natsService := &NatsService{natsIdMap: natsIdMap, hostIdMap: hostIdMap, subsByHost: subsByHost, appRuntime: appRuntime}

	for _, host := range list {
		url := host.Nats.URL()
		connect, err := nats.Connect(url,
			nats.Name(fmt.Sprintf("app-server-host-%d", host.ID)),
			// 说明：nats.go 客户端会在重连后自动恢复订阅，这里不再手动重复订阅，避免重复消费
		)
		if err != nil {
			panic(err)
		}
		// 初次连接时即在该连接上订阅固定主题
		subs, err := subscribeOnConn(connect, natsService)
		if err != nil {
			panic(err)
		}

		natsIdMap[host.NatsID] = connect
		hostIdMap[host.ID] = connect
		subsByHost[host.ID] = subs

	}

	return natsService
}

func (n *NatsService) GetNatsByHost(hostId int64) (*nats.Conn, error) {
	n.mu.RLock()
	conn := n.hostIdMap[hostId]
	n.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("nats host id %d not exist", hostId)
	}
	return conn, nil
}

func (n *NatsService) GetNatsByNatsId(natsId int64) (*nats.Conn, error) {
	n.mu.RLock()
	conn := n.natsIdMap[natsId]
	n.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("nats id %d not exist", natsId)
	}
	return conn, nil
}

// SetAppRuntime 设置 AppRuntime 引用
func (n *NatsService) SetAppRuntime(appRuntime *AppRuntime) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.appRuntime = appRuntime
}

// subscribeOnConn 在指定连接上注册当前服务需要监听的主题
func subscribeOnConn(conn *nats.Conn, natsService *NatsService) ([]*nats.Subscription, error) {
	// 示例：每个连接都监听相同的响应主题
	s1, err := conn.Subscribe(subjects.GetApp2FunctionServerResponseSubject(), func(msg *nats.Msg) {
		if natsService.appRuntime != nil {
			natsService.appRuntime.HandleApp2FunctionServerResponse(msg)
		}
	})
	if err != nil {
		return nil, err
	}
	return []*nats.Subscription{s1}, nil
}

func (n *NatsService) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	for hostID, conn := range n.hostIdMap {
		if subs, ok := n.subsByHost[hostID]; ok {
			for _, sub := range subs {
				if sub != nil {
					_ = sub.Unsubscribe()
				}
			}
		}
		if conn != nil {
			conn.Close()
		}
		delete(n.subsByHost, hostID)
		delete(n.hostIdMap, hostID)
	}
	for k := range n.natsIdMap {
		delete(n.natsIdMap, k)
	}
	return nil
}
