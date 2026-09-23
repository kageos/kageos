package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	timerservice "github.com/kageos/kageos/core/timer-scheduler/service"
	"github.com/kageos/kageos/dto"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTaskAuditNATSAcknowledgesOnlyPersistedEvents(t *testing.T) {
	binary, err := exec.LookPath("nats-server")
	if err != nil {
		t.Skip("nats-server is required for transport integration test")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	process := exec.Command(binary, "-a", "127.0.0.1", "-p", fmt.Sprint(port))
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = process.Process.Signal(os.Interrupt); _ = process.Wait() })
	var conn *nats.Conn
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port), nats.NoReconnect(), nats.Timeout(100*time.Millisecond))
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Cleanup(conn.Close)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "audit.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{db: db, natsConn: conn}
	if err := server.startTaskAuditConsumer(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.taskAuditSub.Unsubscribe() })
	event := dto.TaskAuditEvent{EventID: "transport-audit", OccurredAt: time.Now(), Actor: "alice", ResourcePath: "/alice/demo/task.form", Action: "timer.task.paused", Status: "success", TaskID: 1}
	payload, _ := json.Marshal(event)
	publisher := timerservice.NewNATSOutboxPublisher(conn)
	if err := publisher.Publish(context.Background(), dto.TaskAuditSubject, payload); err == nil {
		t.Fatal("missing database table must not acknowledge delivery")
	}
	if err := db.AutoMigrate(&model.OperateLog{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := publisher.Publish(context.Background(), dto.TaskAuditSubject, payload); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&model.OperateLog{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("retry produced %d rows", count)
	}
}
