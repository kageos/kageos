package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPackageDocsTestService(t *testing.T) (*AppService, *repository.DocRepository, *repository.ServiceTreeRepository, *model.App) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.App{}, &model.ServiceTree{}, &model.Docs{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	appModel := &model.App{User: "alice", Code: "demo", Name: "Demo", Version: "v3"}
	if err := db.Create(appModel).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	serviceTreeRepo := repository.NewServiceTreeRepository(db)
	for _, tree := range []*model.ServiceTree{
		{
			AppID:        appModel.ID,
			Type:         model.ServiceTreeTypePackage,
			Code:         "demo",
			Name:         "Demo",
			FullCodePath: "/alice/demo",
			RefID:        appModel.ID,
		},
		{
			AppID:        appModel.ID,
			Type:         model.ServiceTreeTypePackage,
			Code:         "followup",
			Name:         "物流节点跟进",
			FullCodePath: "/alice/demo/followup",
		},
	} {
		if err := serviceTreeRepo.Create(tree); err != nil {
			t.Fatalf("create service tree %s: %v", tree.FullCodePath, err)
		}
	}
	docRepo := repository.NewDocRepository(db)
	svc := NewAppService(nil, nil, nil, serviceTreeRepo, nil)
	svc.SetDocService(NewDocService(docRepo, serviceTreeRepo, nil, nil))
	return svc, docRepo, serviceTreeRepo, appModel
}

func TestReconcilePackageDocsCreatesMissingRunbook(t *testing.T) {
	svc, docRepo, serviceTreeRepo, appModel := newPackageDocsTestService(t)

	err := svc.reconcilePackageDocs(context.Background(), &appMetadataSyncState{
		app:               appModel,
		currentVersionNum: 3,
		requestUser:       "alice",
	}, []*dto.PackageInfo{
		{
			FullPath: "/alice/demo/followup",
			Docs: []dto.DocSeedConfig{
				{
					Code:    "runbook",
					Name:    "运行手册",
					Content: "# 物流节点跟进运行手册\n",
					Format:  "markdown",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := serviceTreeRepo.GetServiceTreeByFullPath("/alice/demo/followup/runbook.docs")
	if err != nil {
		t.Fatalf("get runbook tree: %v", err)
	}
	if tree.Type != model.ServiceTreeTypeDocs || tree.Name != "运行手册" || tree.AddVersionNum != 3 {
		t.Fatalf("unexpected runbook tree: %#v", tree)
	}
	doc, err := docRepo.GetByTreeID(tree.ID)
	if err != nil {
		t.Fatalf("get runbook doc: %v", err)
	}
	if doc.Content != "# 物流节点跟进运行手册\n" || doc.Format != "markdown" || doc.FullCodePath != tree.FullCodePath {
		t.Fatalf("unexpected runbook doc: %#v", doc)
	}
}

func TestReconcilePackageDocsDoesNotOverwriteExistingDoc(t *testing.T) {
	svc, docRepo, serviceTreeRepo, appModel := newPackageDocsTestService(t)
	tree := &model.ServiceTree{
		AppID:        appModel.ID,
		Type:         model.ServiceTreeTypeDocs,
		Code:         "runbook.docs",
		Name:         "运行手册",
		FullCodePath: "/alice/demo/followup/runbook.docs",
	}
	if err := serviceTreeRepo.Create(tree); err != nil {
		t.Fatalf("create runbook tree: %v", err)
	}
	doc := &model.Docs{
		Name:         "运行手册",
		Content:      "# 已人工更新的运行手册\n",
		Format:       "markdown",
		AppID:        appModel.ID,
		TreeID:       tree.ID,
		FullCodePath: tree.FullCodePath,
	}
	if err := docRepo.Create(doc); err != nil {
		t.Fatalf("create runbook doc: %v", err)
	}

	err := svc.reconcilePackageDocs(context.Background(), &appMetadataSyncState{
		app:               appModel,
		currentVersionNum: 3,
		requestUser:       "alice",
	}, []*dto.PackageInfo{
		{
			FullPath: "/alice/demo/followup",
			Docs: []dto.DocSeedConfig{
				{Code: "runbook", Name: "运行手册", Content: "# Seed 旧内容\n"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := docRepo.GetByTreeID(tree.ID)
	if err != nil {
		t.Fatalf("get runbook doc: %v", err)
	}
	if got.Content != "# 已人工更新的运行手册\n" {
		t.Fatalf("existing runbook was overwritten: %#v", got)
	}
}
