package repository

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/connector-server/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetOwnedConnectionsReturnsRequestedActiveRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:connector-batch?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.ConnectorConnection{}); err != nil {
		t.Fatal(err)
	}
	rows := []*model.ConnectorConnection{
		{ConnectionID: "a", OwnerUsername: "alice", Provider: "github", DisplayName: "A", Status: model.ConnectorStatusActive},
		{ConnectionID: "b", OwnerUsername: "alice", Provider: "notion", DisplayName: "B", Status: model.ConnectorStatusActive},
		{ConnectionID: "c", OwnerUsername: "bob", Provider: "github", DisplayName: "C", Status: model.ConnectorStatusActive},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	got, err := NewConnectorRepository(db).GetOwnedConnections(context.Background(), "alice", []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["a"] == nil || got["b"] == nil || got["c"] != nil {
		t.Fatalf("unexpected connections: %#v", got)
	}
}
