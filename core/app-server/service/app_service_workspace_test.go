package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUpdateWorkspaceRenamesAppAndRootTogether(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewAppRepository(db)
	app := &model.App{User: "alice", Code: "home", Name: "我的空间", Version: "v1"}
	if err := repo.CreateApp(app); err != nil {
		t.Fatal(err)
	}
	root := &model.ServiceTree{AppID: app.ID, RefID: app.ID, Name: app.Name, Code: app.Code, Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/home"}
	if err := db.Create(root).Error; err != nil {
		t.Fatal(err)
	}

	name := "客户交付"
	service := NewAppService(AppServiceDependencies{
		AppRuntimeClient:      &fakeAppRuntimeClient{},
		AppRepository:         repo,
		ServiceTreeRepository: repository.NewServiceTreeRepository(db),
	})
	resp, err := service.UpdateWorkspace(context.Background(), &dto.UpdateWorkspaceReq{ResourcePath: "/alice/home", Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != name {
		t.Fatalf("response name = %q", resp.Name)
	}
	var updatedApp model.App
	var updatedRoot model.ServiceTree
	_ = db.First(&updatedApp, app.ID).Error
	_ = db.First(&updatedRoot, root.ID).Error
	if updatedApp.Name != name || updatedApp.Code != "home" || updatedRoot.Name != name || updatedRoot.FullCodePath != "/alice/home" {
		t.Fatalf("rename changed identity or missed root: app=%#v root=%#v", updatedApp, updatedRoot)
	}
}
