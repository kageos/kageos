package server

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kageos/kageos/core/app-runtime/model"
	"gorm.io/gorm"
)

func TestCollectRuntimePlatformStats(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AppDatabase{}); err != nil {
		t.Fatal(err)
	}
	for _, database := range []model.AppDatabase{
		{User: "alice", App: "one", PackagePath: "_root", FullCodePath: "/alice/one", DatabaseName: "one", DatabaseUser: "one", MigrationDatabaseUser: "one_m", Status: "active"},
		{User: "alice", App: "two", PackagePath: "_root", FullCodePath: "/alice/two", DatabaseName: "two", DatabaseUser: "two", MigrationDatabaseUser: "two_m", Status: "pending"},
	} {
		if err := db.Create(&database).Error; err != nil {
			t.Fatal(err)
		}
	}
	stats, err := collectRuntimePlatformStats(db)
	if err != nil {
		t.Fatal(err)
	}
	if stats.AppDatabasesTotal != 1 {
		t.Fatalf("active app database count = %d, want 1", stats.AppDatabasesTotal)
	}
}
