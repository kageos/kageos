package model

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitTablesRestoresOnlyIdentityQuarantinePause(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "timer.db")), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TimerTask{}, &TimerExecution{}, &TimerOutboxEvent{}); err != nil {
		t.Fatal(err)
	}

	quarantined := TimerTask{
		ExecutorKey:      "test",
		ScheduleType:     "every",
		Status:           "paused",
		LastErrorMessage: legacyIdentityQuarantineMessage,
	}
	userPaused := TimerTask{ExecutorKey: "test", ScheduleType: "every", Status: "paused"}
	otherFailure := TimerTask{
		ExecutorKey:      "test",
		ScheduleType:     "every",
		Status:           "paused",
		LastErrorMessage: "业务处理失败",
	}
	for _, task := range []*TimerTask{&quarantined, &userPaused, &otherFailure} {
		if err := db.Create(task).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := InitTables(db); err != nil {
		t.Fatal(err)
	}
	for _, task := range []*TimerTask{&quarantined, &userPaused, &otherFailure} {
		if err := db.First(task, task.ID).Error; err != nil {
			t.Fatal(err)
		}
	}
	if quarantined.Status != "pending" || quarantined.LastErrorMessage != "" {
		t.Fatalf("quarantined task was not restored: status=%q error=%q", quarantined.Status, quarantined.LastErrorMessage)
	}
	if userPaused.Status != "paused" || userPaused.LastErrorMessage != "" {
		t.Fatalf("user-paused task changed unexpectedly: status=%q error=%q", userPaused.Status, userPaused.LastErrorMessage)
	}
	if otherFailure.Status != "paused" || otherFailure.LastErrorMessage != "业务处理失败" {
		t.Fatalf("unrelated failed task changed unexpectedly: status=%q error=%q", otherFailure.Status, otherFailure.LastErrorMessage)
	}
}
