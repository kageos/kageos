package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRemoveRetiredOperateLogCompanyColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:retired-operate-log-company?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE `operate_logs` (`id` INTEGER PRIMARY KEY, `company_code` TEXT)").Error; err != nil {
		t.Fatal(err)
	}

	if err := removeRetiredOperateLogCompanyColumn(db); err != nil {
		t.Fatal(err)
	}
	if db.Migrator().HasColumn(&OperateLog{}, "company_code") {
		t.Fatal("operate_logs.company_code should be removed")
	}
}
