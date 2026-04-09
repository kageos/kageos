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

// NATSConnPool 按 hostID / natsID 持有 app-server 需要访问的远端 NATS 连接。
// 这层只负责连接复用和关闭，不负责订阅生命周期。
type NATSConnPool struct {
	mu           sync.RWMutex
	natsIDToConn map[int64]*nats.Conn
	hostIDToConn map[int64]*nats.Conn
}

// NewNATSConnPoolWithDB 使用指定的数据库连接创建 NATS 连接池。
func NewNATSConnPoolWithDB(db *gorm.DB) *NATSConnPool {
	hostRepo := repository.NewHostRepository(db)
	list, err := hostRepo.GetHostList()
	if err != nil {
		panic(err)
	}
	return newNATSConnPoolFromHostList(list)
}

// newNATSConnPoolFromHostList 从主机列表创建 NATS 连接池。
func newNATSConnPoolFromHostList(list []*model.Host) *NATSConnPool {
	natsIDToConn := make(map[int64]*nats.Conn)
	hostIDToConn := make(map[int64]*nats.Conn)

	for _, host := range list {
		url := host.Nats.URL()
		conn, err := natsx.ConnectNamed(url, fmt.Sprintf("app-server-host-%d", host.ID))
		if err != nil {
			panic(err)
		}

		natsIDToConn[host.NatsID] = conn
		hostIDToConn[host.ID] = conn
	}

	return &NATSConnPool{
		natsIDToConn: natsIDToConn,
		hostIDToConn: hostIDToConn,
	}
}

func (p *NATSConnPool) GetConnByHost(hostID int64) (*nats.Conn, error) {
	p.mu.RLock()
	conn := p.hostIDToConn[hostID]
	p.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("nats host id %d not exist", hostID)
	}
	return conn, nil
}

func (p *NATSConnPool) GetConnByNATSID(natsID int64) (*nats.Conn, error) {
	p.mu.RLock()
	conn := p.natsIDToConn[natsID]
	p.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("nats id %d not exist", natsID)
	}
	return conn, nil
}

// HostIDs 返回所有 hostID 列表（供 appcall.Client 初始化响应主题订阅用）。
func (p *NATSConnPool) HostIDs() []int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ids := make([]int64, 0, len(p.hostIDToConn))
	for id := range p.hostIDToConn {
		ids = append(ids, id)
	}
	return ids
}

func (p *NATSConnPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for hostID, conn := range p.hostIDToConn {
		if conn != nil {
			conn.Close()
		}
		delete(p.hostIDToConn, hostID)
	}
	for natsID := range p.natsIDToConn {
		delete(p.natsIDToConn, natsID)
	}
	return nil
}
