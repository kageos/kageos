package model

import (
	"fmt"

	"gorm.io/gorm"
)

const legacyIdentityQuarantineMessage = "任务身份未验证，请删除后使用已登录客户端重新创建"

func InitTables(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&TimerTask{},
		&TimerExecution{},
		&TimerOutboxEvent{},
	); err != nil {
		return err
	}
	return restoreTasksPausedByIdentityQuarantine(db)
}

// restoreTasksPausedByIdentityQuarantine reverses only the exact pause written
// by the short-lived identity quarantine migration. User-paused and terminal
// tasks are intentionally untouched.
func restoreTasksPausedByIdentityQuarantine(db *gorm.DB) error {
	if err := db.Model(&TimerTask{}).
		Where("status = ? AND last_error_message = ?", "paused", legacyIdentityQuarantineMessage).
		Updates(map[string]interface{}{
			"status":             "pending",
			"last_error_message": "",
			"lease_owner":        "",
			"lease_until":        nil,
		}).Error; err != nil {
		return fmt.Errorf("restore tasks paused by identity quarantine: %w", err)
	}
	return nil
}
