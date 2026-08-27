package server

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

func TestCollectAppPlatformStats(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.App{Code: "one", Name: "One", User: "alice", Status: "enabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.App{Code: "two", Name: "Two", User: "alice", Status: "disabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ServiceTree{Type: model.ServiceTreeTypePackage}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ServiceTree{Type: model.ServiceTreeTypeFunction}).Error; err != nil {
		t.Fatal(err)
	}
	stats, err := collectAppPlatformStats(db)
	if err != nil {
		t.Fatal(err)
	}
	if stats.WorkspacesTotal != 2 || stats.WorkspacesEnabled != 1 || stats.ServiceDirectories != 1 || stats.FunctionsTotal != 1 {
		t.Fatalf("stats = %+v", stats)
	}
}
