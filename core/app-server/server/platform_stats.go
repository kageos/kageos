package server

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func (s *Server) startPlatformStatsResponder() error {
	if s.natsConn == nil || s.db == nil {
		return nil
	}
	sub, err := s.natsConn.QueueSubscribe(subjects.PlatformAppStatsQuerySubject, subjects.PlatformAppStatsQueueGroup, func(msg *nats.Msg) {
		stats, err := collectAppPlatformStats(s.db)
		if err != nil {
			logger.Warnf(s.ctx, "[PlatformStats] collect app statistics failed: %v", err)
			_ = msg.Respond([]byte(`{"error":"query failed"}`))
			return
		}
		data, _ := json.Marshal(stats)
		_ = msg.Respond(data)
	})
	if err != nil {
		return err
	}
	s.platformStatsSub = sub
	return s.natsConn.Flush()
}

func collectAppPlatformStats(db *gorm.DB) (dto.SystemPlatformServiceStats, error) {
	stats := dto.SystemPlatformServiceStats{}
	queries := []struct {
		model        any
		where, value string
		target       *int64
	}{
		{&model.App{}, "", "", &stats.WorkspacesTotal},
		{&model.App{}, "status = ?", "enabled", &stats.WorkspacesEnabled},
		{&model.ServiceTree{}, "type = ?", model.ServiceTreeTypePackage, &stats.ServiceDirectories},
		{&model.ServiceTree{}, "type = ?", model.ServiceTreeTypeFunction, &stats.FunctionsTotal},
	}
	for _, query := range queries {
		statement := db.Model(query.model)
		if query.where != "" {
			statement = statement.Where(query.where, query.value)
		}
		if err := statement.Count(query.target).Error; err != nil {
			return stats, err
		}
	}
	usage, err := collectAppUsageStats(db, time.Now())
	if err != nil {
		return stats, err
	}
	stats.Usage = usage
	return stats, nil
}

func collectAppUsageStats(db *gorm.DB, now time.Time) (dto.SystemUsageSnapshot, error) {
	localNow := now.In(time.Local)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.Local)
	yesterday := today.AddDate(0, 0, -1)
	sevenDays := today.AddDate(0, 0, -6)
	thirtyDays := today.AddDate(0, 0, -29)

	type operationCounts struct {
		Today           int64 `gorm:"column:today"`
		Yesterday       int64 `gorm:"column:yesterday"`
		Last7Days       int64 `gorm:"column:last_7_days"`
		Last30Days      int64 `gorm:"column:last_30_days"`
		FailedToday     int64 `gorm:"column:failed_today"`
		FailedYesterday int64 `gorm:"column:failed_yesterday"`
		FailedLast7     int64 `gorm:"column:failed_last_7"`
		FailedLast30    int64 `gorm:"column:failed_last_30"`
	}
	var counts operationCounts
	if err := db.Model(&model.OperateLog{}).Where("created_at >= ?", thirtyDays).Select(`
		COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0) AS today,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS yesterday,
		COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0) AS last_7_days,
		COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0) AS last_30_days,
		COALESCE(SUM(CASE WHEN created_at >= ? AND status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_today,
		COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? AND status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_yesterday,
		COALESCE(SUM(CASE WHEN created_at >= ? AND status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_last_7,
		COALESCE(SUM(CASE WHEN created_at >= ? AND status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_last_30`,
		today, yesterday, today, sevenDays, thirtyDays,
		today, yesterday, today, sevenDays, thirtyDays).Scan(&counts).Error; err != nil {
		return dto.SystemUsageSnapshot{}, err
	}

	var nodes []model.ServiceTree
	if err := db.Model(&model.ServiceTree{}).
		Where("type = ?", model.ServiceTreeTypeFunction).
		Select("name", "full_code_path", "template_type", "run_count").
		Order("full_code_path ASC").Find(&nodes).Error; err != nil {
		return dto.SystemUsageSnapshot{}, err
	}
	var directories []model.ServiceTree
	if err := db.Model(&model.ServiceTree{}).
		Where("type = ?", model.ServiceTreeTypePackage).
		Select("name", "full_code_path").Find(&directories).Error; err != nil {
		return dto.SystemUsageSnapshot{}, err
	}
	directoryNames := make(map[string]string, len(directories))
	for _, directory := range directories {
		directoryNames[strings.TrimRight(directory.FullCodePath, "/")] = directory.Name
	}
	functions := make([]dto.SystemFunctionUsageSnapshot, 0, len(nodes))
	for _, node := range nodes {
		path := strings.TrimRight(node.FullCodePath, "/")
		directoryPath := path
		if index := strings.LastIndex(path, "/"); index > 0 {
			directoryPath = path[:index]
		}
		functions = append(functions, dto.SystemFunctionUsageSnapshot{
			Path: path, Name: node.Name, DirectoryPath: directoryPath,
			DirectoryName: directoryNames[directoryPath], TemplateType: node.TemplateType, TotalCalls: int64(node.RunCount),
		})
	}
	return dto.SystemUsageSnapshot{
		Available: true, CollectedAt: now.UTC(), OperationDay: yesterday.Format("2006-01-02"),
		OperationsToday: counts.Today, OperationsYesterday: counts.Yesterday,
		OperationsLast7Days: counts.Last7Days, OperationsLast30Days: counts.Last30Days,
		FailedOperationsToday: counts.FailedToday, FailedOperationsYesterday: counts.FailedYesterday,
		FailedOperationsLast7Days: counts.FailedLast7, FailedOperationsLast30Days: counts.FailedLast30,
		Functions: functions,
	}, nil
}
