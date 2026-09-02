package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestEnsureOperateLogQueryIndexesCreatesUsageOverviewIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&OperateLog{}); err != nil {
		t.Fatal(err)
	}
	if err := ensureOperateLogQueryIndexes(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasIndex(&OperateLog{}, "idx_oplog_deleted_created_status") {
		t.Fatal("usage overview covering index was not created")
	}
}
