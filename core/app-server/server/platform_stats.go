package server

import (
	"encoding/json"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
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
	return stats, nil
}
