package service

import (
	"fmt"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/natsx"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type NatsService struct {
	mu         sync.RWMutex
	natsIdMap  map[int64]*nats.Conn
	hostIdMap  map[int64]*nats.Conn
	subsByHost map[int64][]*nats.Subscription
}

// NewNatsServiceWithDB 使用指定的数据库连接创建 NATS 服务
func NewNatsServiceWithDB(db *gorm.DB) *NatsService {
	hostRepo := repository.NewHostRepository(db)
	list, err := hostRepo.GetHostList()
	if err != nil {
		panic(err)
	}
	return newNatsServiceFromHostList(list)
}

// newNatsServiceFromHostList 从主机列表创建 NATS 服务
func newNatsServiceFromHostList(list []*model.Host) *NatsService {
	natsIdMap := make(map[int64]*nats.Conn)
	hostIdMap := make(map[int64]*nats.Conn)
	subsByHost := make(map[int64][]*nats.Subscription)

	for _, host := range list {
		url := host.Nats.URL()
		connect, err := natsx.ConnectNamed(url, fmt.Sprintf("app-server-host-%d", host.ID))
		if err != nil {
			panic(err)
		}

		natsIdMap[host.NatsID] = connect
		hostIdMap[host.ID] = connect
		subsByHost[host.ID] = []*nats.Subscription{} // 暂时不订阅任何主题

	}

	return &NatsService{natsIdMap: natsIdMap, hostIdMap: hostIdMap, subsByHost: subsByHost}
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

// HostIds 返回所有 hostId 列表（供 appcall.Client 订阅响应主题用）
func (n *NatsService) HostIds() []int64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	ids := make([]int64, 0, len(n.hostIdMap))
	for id := range n.hostIdMap {
		ids = append(ids, id)
	}
	return ids
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
