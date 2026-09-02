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
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}, &model.OperateLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.App{Code: "one", Name: "One", User: "alice", Status: "enabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.App{Code: "two", Name: "Two", User: "alice", Status: "disabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ServiceTree{Type: model.ServiceTreeTypePackage, Name: "Tools", FullCodePath: "/alice/one/tools"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ServiceTree{Type: model.ServiceTreeTypeFunction, Name: "Convert", FullCodePath: "/alice/one/tools/convert.form", TemplateType: "form", RunCount: 12}).Error; err != nil {
		t.Fatal(err)
	}
	for _, log := range []model.OperateLog{
		{TenantUser: "alice", App: "one", ActorUser: "alice", Action: "form_submit", ResourcePath: "/alice/one/tools/convert.form", Status: "success"},
		{TenantUser: "alice", App: "one", ActorUser: "alice", Action: "form_submit", ResourcePath: "/alice/one/tools/convert.form", Status: "failed"},
	} {
		if err := db.Create(&log).Error; err != nil {
			t.Fatal(err)
		}
	}
	stats, err := collectAppPlatformStats(db)
	if err != nil {
		t.Fatal(err)
	}
	if stats.WorkspacesTotal != 2 || stats.WorkspacesEnabled != 1 || stats.ServiceDirectories != 1 || stats.FunctionsTotal != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if !stats.Usage.Available || stats.Usage.OperationsToday != 2 || stats.Usage.FailedOperationsToday != 1 {
		t.Fatalf("usage = %+v", stats.Usage)
	}
	if len(stats.Usage.Functions) != 1 || stats.Usage.Functions[0].TotalCalls != 12 || stats.Usage.Functions[0].DirectoryName != "Tools" {
		t.Fatalf("function usage = %+v", stats.Usage.Functions)
	}
}
