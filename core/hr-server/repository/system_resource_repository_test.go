package repository

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

func TestCollectPlatformMetricsCountsSharedPlatformTables(t *testing.T) {
	db := openSystemResourceTestDB(t)
	statements := []string{
		`CREATE TABLE user (id INTEGER PRIMARY KEY, status TEXT, deleted_at DATETIME)`,
		`INSERT INTO user VALUES (1, 'active', NULL), (2, 'pending', NULL), (3, 'active', CURRENT_TIMESTAMP)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	metrics, err := NewSystemResourceRepository(db).CollectPlatformMetrics(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if metrics.UsersTotal != 2 || metrics.UsersActive != 1 || metrics.UsersPending != 1 {
		t.Fatalf("user metrics = %+v", metrics)
	}
}

func TestDeleteBeforeHardDeletesAllMonitoringHistory(t *testing.T) {
	db := openSystemResourceTestDB(t)
	if err := db.AutoMigrate(&model.SystemResourceSample{}, &model.SystemCapacitySnapshot{}, &model.SystemPlatformSnapshot{}); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-31 * 24 * time.Hour)
	if err := db.Create(&model.SystemResourceSample{CollectedAt: old}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemCapacitySnapshot{CollectedAt: old, PayloadJSON: `{}`}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemPlatformSnapshot{CollectedAt: old, PayloadJSON: `{}`}).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewSystemResourceRepository(db)
	if err := repo.DeleteBefore(time.Now().Add(-30 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"system_resource_samples", "system_capacity_snapshots", "system_platform_snapshots"} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
}

func openSystemResourceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
