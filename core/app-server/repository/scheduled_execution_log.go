package repository

import (
	"context"
	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Management actions (task edits, permissions, etc.) remain in operation history.
func IsScheduledExecution(row *model.OperateLog) bool {
	return row != nil && (row.ResourceType == "form" || row.ResourceType == "function" || row.ResourceType == "table") &&
		(row.Source == "scheduled_task" || row.SourceType == "scheduled_task" || row.ExecutorType == "scheduled_function")
}

// MoveLegacyScheduledLogs is called under the archive lock before serving requests.
// Each chunk commits the copy and deletion together. Existing in-flight archives
// retain ownership of their original IDs and resume through the legacy table.
func (r *LogArchiveRepository) MoveLegacyScheduledLogs(ctx context.Context) error {
	for {
		moved := 0
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var rows []model.OperateLog
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("resource_type IN ? AND (source = ? OR source_type = ? OR executor_type = ?)", []string{"form", "function", "table"}, "scheduled_task", "scheduled_task", "scheduled_function").Where(unclaimedArchiveLogs).Order("id").Limit(500).Find(&rows).Error; err != nil {
				return err
			}
			copiedRows := make([]model.ScheduledExecutionLog, 0, len(rows))
			legacyIDs := make([]int64, 0, len(rows))
			for _, row := range rows {
				oldID := row.ID
				copied := model.ScheduledExecutionLog(row)
				copied.ID = 0
				copied.OriginalLogID = &oldID
				copiedRows = append(copiedRows, copied)
				legacyIDs = append(legacyIDs, oldID)
			}
			if len(copiedRows) > 0 {
				if err := tx.CreateInBatches(&copiedRows, len(copiedRows)).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Where("id IN ?", legacyIDs).Delete(&model.OperateLog{}).Error; err != nil {
					return err
				}
			}
			moved = len(rows)
			return nil
		})
		if err != nil {
			return err
		}
		if moved < 500 {
			return nil
		}
	}
}
