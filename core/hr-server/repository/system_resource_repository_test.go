package repository

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/dto"
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

func TestPruneHistoryUsesIndependentRetentionWindows(t *testing.T) {
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
	now := time.Now()
	if err := repo.PruneHistory(now.Add(-30*24*time.Hour), now.Add(-90*24*time.Hour), now.Add(-365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int64{
		"system_resource_samples": 0, "system_platform_snapshots": 1, "system_capacity_snapshots": 1,
	} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil || count != want {
			t.Fatalf("%s count=%d want=%d err=%v", table, count, want, err)
		}
	}
}

func TestDailySnapshotsReplaceSameLocalDay(t *testing.T) {
	db := openSystemResourceTestDB(t)
	if err := db.AutoMigrate(&model.SystemCapacitySnapshot{}, &model.SystemPlatformSnapshot{}); err != nil {
		t.Fatal(err)
	}
	repo := NewSystemResourceRepository(db)
	day := time.Date(2026, 8, 31, 2, 30, 0, 0, time.Local)
	for index, used := range []uint64{100, 140} {
		if err := repo.CreateCapacity(dto.SystemResourceSnapshot{CollectedAt: day.Add(time.Duration(index) * time.Hour), DatabaseLogicalBytes: used}); err != nil {
			t.Fatal(err)
		}
	}
	history, err := repo.CapacityHistory(day.Add(-time.Hour), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].DatabaseLogicalBytes != 140 {
		t.Fatalf("capacity history = %#v, want one replacement snapshot", history)
	}
}

func TestPlatformDatabaseDefinitionsCoverEveryPlatformServiceDatabase(t *testing.T) {
	definitions := platformDatabaseDefinitions()
	for _, name := range []string{"agent-server", "app-server", "app-storage", "connector-server", "hr-server", "message-server", "timer-scheduler"} {
		definition, exists := definitions[name]
		if !exists {
			t.Fatalf("platform database %q is missing", name)
		}
		if definition.service == "" || definition.purpose == "" {
			t.Fatalf("platform database %q metadata is incomplete: %#v", name, definition)
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
