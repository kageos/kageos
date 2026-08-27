package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

var safeAppDatabaseIdentifier = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// PurgeSoftDeletedRows permanently removes explicitly selected soft-deleted
// rows through the runtime-owned admin connection.
func (s *AppDatabaseService) PurgeSoftDeletedRows(ctx context.Context, req *dto.AppDBPurgeRowsReq) (*dto.AppDBPurgeRowsResp, error) {
	if s == nil || !s.IsEnabled() {
		return nil, ErrAppDatabaseDisabled
	}
	if s.db == nil {
		return nil, fmt.Errorf("app database registry is unavailable")
	}
	if req == nil || strings.TrimSpace(req.User) == "" || strings.TrimSpace(req.App) == "" || strings.TrimSpace(req.PackagePath) == "" {
		return nil, fmt.Errorf("app database purge scope is incomplete")
	}
	if !safeAppDatabaseIdentifier.MatchString(req.Table) || len(req.IDs) == 0 || len(req.IDs) > 100 {
		return nil, fmt.Errorf("invalid app database purge target")
	}
	ids := make([]int64, 0, len(req.IDs))
	seen := make(map[int64]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if id <= 0 {
			return nil, fmt.Errorf("purge ids must be positive integers")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	var database model.AppDatabase
	if err := s.db.Where("user = ? AND app = ? AND package_path = ? AND status = ?", req.User, req.App, req.PackagePath, appDBStatusActive).First(&database).Error; err != nil {
		return nil, fmt.Errorf("resolve app database purge target: %w", err)
	}
	adminDB, err := s.openAdminDB()
	if err != nil {
		return nil, err
	}
	defer closeGORM(adminDB)
	qualifiedTable := quoteMySQLIdentifier(database.DatabaseName) + "." + quoteMySQLIdentifier(req.Table)
	var hasDeletedAt int64
	if err := adminDB.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = 'deleted_at'", database.DatabaseName, req.Table).Scan(&hasDeletedAt).Error; err != nil || hasDeletedAt == 0 {
		return nil, fmt.Errorf("target table does not support soft delete")
	}
	rows := make([]map[string]interface{}, 0)
	if err := adminDB.Raw("SELECT * FROM "+qualifiedTable+" WHERE id IN ? AND deleted_at IS NOT NULL", ids).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("read purge snapshots: %w", err)
	}
	if len(rows) != len(ids) {
		return nil, fmt.Errorf("some rows do not exist or are not in recycle bin")
	}
	result := adminDB.Exec("DELETE FROM "+qualifiedTable+" WHERE id IN ? AND deleted_at IS NOT NULL", ids)
	if result.Error != nil {
		return nil, fmt.Errorf("purge rows: %w", result.Error)
	}
	logger.Infof(ctx, "[AppDatabaseCleanup] manually purged user=%s app=%s package=%s table=%s count=%d", req.User, req.App, req.PackagePath, req.Table, result.RowsAffected)
	return &dto.AppDBPurgeRowsResp{Rows: rows, Purged: result.RowsAffected}, nil
}

func (s *AppDatabaseService) GetSoftDeleteCleanupPolicy(ctx context.Context, req *dto.AppDBCleanupPolicyReq) (*dto.AppDBCleanupPolicyResp, error) {
	database, err := s.resolveCleanupPolicyDatabase(ctx, req)
	if err != nil {
		return nil, err
	}
	if !safeAppDatabaseIdentifier.MatchString(req.Table) {
		return nil, fmt.Errorf("invalid cleanup policy table")
	}

	resp := s.defaultCleanupPolicyResponse()
	var policy model.AppDatabaseCleanupPolicy
	err = s.db.WithContext(ctx).
		Where("app_database_id = ? AND table_name = ?", database.ID, req.Table).
		First(&policy).Error
	if err == nil {
		resp.Enabled = policy.Enabled
		resp.Mode = policy.Mode
		resp.RetentionDays = policy.RetentionDays
		resp.Source = "table"
		resp.UpdatedBy = policy.UpdatedBy
		return resp, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("read cleanup policy: %w", err)
	}
	return resp, nil
}

func (s *AppDatabaseService) UpdateSoftDeleteCleanupPolicy(ctx context.Context, req *dto.AppDBCleanupPolicyUpdateReq) (*dto.AppDBCleanupPolicyResp, error) {
	if req == nil {
		return nil, fmt.Errorf("cleanup policy request is nil")
	}
	database, err := s.resolveCleanupPolicyDatabase(ctx, &req.AppDBCleanupPolicyReq)
	if err != nil {
		return nil, err
	}
	if !safeAppDatabaseIdentifier.MatchString(req.Table) {
		return nil, fmt.Errorf("invalid cleanup policy table")
	}
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != "dry_run" && mode != "purge" {
		return nil, fmt.Errorf("cleanup policy mode must be dry_run or purge")
	}
	if req.RetentionDays < 1 || req.RetentionDays > 3650 {
		return nil, fmt.Errorf("cleanup retention days must be between 1 and 3650")
	}

	policy := model.AppDatabaseCleanupPolicy{
		AppDatabaseID: database.ID,
		TargetTable:   req.Table,
	}
	err = s.db.WithContext(ctx).
		Where("app_database_id = ? AND table_name = ?", database.ID, req.Table).
		Assign(model.AppDatabaseCleanupPolicy{
			Enabled: req.Enabled, Mode: mode, RetentionDays: req.RetentionDays,
			UpdatedBy: strings.TrimSpace(req.RequestUser),
		}).FirstOrCreate(&policy).Error
	if err != nil {
		return nil, fmt.Errorf("save cleanup policy: %w", err)
	}
	return s.GetSoftDeleteCleanupPolicy(ctx, &req.AppDBCleanupPolicyReq)
}

func (s *AppDatabaseService) resolveCleanupPolicyDatabase(ctx context.Context, req *dto.AppDBCleanupPolicyReq) (*model.AppDatabase, error) {
	if s == nil || !s.IsEnabled() {
		return nil, ErrAppDatabaseDisabled
	}
	if s.db == nil || req == nil || strings.TrimSpace(req.User) == "" || strings.TrimSpace(req.App) == "" || strings.TrimSpace(req.PackagePath) == "" {
		return nil, fmt.Errorf("cleanup policy scope is incomplete")
	}
	var database model.AppDatabase
	err := s.db.WithContext(ctx).
		Where("user = ? AND app = ? AND package_path = ? AND status = ?", req.User, req.App, req.PackagePath, appDBStatusActive).
		First(&database).Error
	if err != nil {
		return nil, fmt.Errorf("resolve cleanup policy target: %w", err)
	}
	return &database, nil
}

func (s *AppDatabaseService) defaultCleanupPolicyResponse() *dto.AppDBCleanupPolicyResp {
	cfg := s.cfg.SoftDeleteCleanup
	return &dto.AppDBCleanupPolicyResp{
		Enabled: cfg.Enabled, Mode: cfg.Mode, RetentionDays: cfg.RetentionDays,
		IntervalMinutes: cfg.IntervalMinutes, BatchSize: cfg.BatchSize, Source: "deployment",
	}
}

// SoftDeleteCleanupReport summarizes one platform-owned cleanup pass.
type SoftDeleteCleanupReport struct {
	Mode          string
	Cutoff        time.Time
	Databases     int
	Tables        int
	CandidateRows int64
	PurgedRows    int64
	FailedTables  int
}

type softDeleteTableName struct {
	TableName string `gorm:"column:table_name"`
}

// StartSoftDeleteCleanup runs the configured cleanup loop until ctx is done.
// Deployment cleanup is disabled by default; table overrides may opt in, and
// every dry_run policy remains read-only.
func (s *AppDatabaseService) StartSoftDeleteCleanup(ctx context.Context) {
	if s == nil || !s.IsEnabled() {
		return
	}
	interval := time.Duration(s.cfg.SoftDeleteCleanup.IntervalMinutes) * time.Minute
	logger.Infof(ctx, "[AppDatabaseCleanup] started mode=%s retention_days=%d interval=%s batch_size=%d",
		s.cfg.SoftDeleteCleanup.Mode,
		s.cfg.SoftDeleteCleanup.RetentionDays,
		interval,
		s.cfg.SoftDeleteCleanup.BatchSize,
	)
	run := func() {
		report, err := s.RunSoftDeleteCleanup(ctx)
		if err != nil {
			logger.Errorf(ctx, "[AppDatabaseCleanup] cleanup pass failed: %v", err)
			return
		}
		logger.Infof(ctx, "[AppDatabaseCleanup] completed mode=%s cutoff=%s databases=%d tables=%d candidates=%d purged=%d failed_tables=%d",
			report.Mode, report.Cutoff.Format(time.RFC3339), report.Databases, report.Tables,
			report.CandidateRows, report.PurgedRows, report.FailedTables)
	}

	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// RunSoftDeleteCleanup scans runtime-managed app databases for tables that
// explicitly contain a deleted_at column. Ordinary app credentials are not
// used and retain their no-DELETE privilege boundary.
func (s *AppDatabaseService) RunSoftDeleteCleanup(ctx context.Context) (*SoftDeleteCleanupReport, error) {
	if s == nil || !s.IsEnabled() {
		return nil, ErrAppDatabaseDisabled
	}
	cfg := s.cfg.SoftDeleteCleanup
	cutoff := time.Now().UTC().AddDate(0, 0, -cfg.RetentionDays)
	report := &SoftDeleteCleanupReport{Mode: cfg.Mode, Cutoff: cutoff}
	if !cfg.Enabled {
		var enabledOverrides int64
		if err := s.db.WithContext(ctx).Model(&model.AppDatabaseCleanupPolicy{}).
			Where("enabled = ?", true).Count(&enabledOverrides).Error; err != nil {
			return nil, fmt.Errorf("count enabled cleanup policy overrides: %w", err)
		}
		if enabledOverrides == 0 {
			return report, nil
		}
		report.Mode = "table_override"
	}
	var databases []model.AppDatabase
	query := s.db.WithContext(ctx).Where("status = ?", appDBStatusActive)
	if clusterKey := strings.TrimSpace(s.cfg.ClusterKey); clusterKey != "" {
		query = query.Where("cluster_key = ?", clusterKey)
	}
	if err := query.Find(&databases).Error; err != nil {
		return nil, fmt.Errorf("list managed app databases: %w", err)
	}
	report.Databases = len(databases)
	if len(databases) == 0 {
		return report, nil
	}

	adminDB, err := s.openAdminDB()
	if err != nil {
		return nil, fmt.Errorf("open cleanup admin database: %w", err)
	}
	if sqlDB, dbErr := adminDB.DB(); dbErr == nil {
		defer sqlDB.Close()
	}

	for _, database := range databases {
		var overrides []model.AppDatabaseCleanupPolicy
		if err := s.db.WithContext(ctx).Where("app_database_id = ?", database.ID).Find(&overrides).Error; err != nil {
			return nil, fmt.Errorf("list cleanup policy overrides: %w", err)
		}
		overrideByTable := make(map[string]model.AppDatabaseCleanupPolicy, len(overrides))
		for _, override := range overrides {
			overrideByTable[override.TargetTable] = override
		}
		var tables []softDeleteTableName
		if err := adminDB.WithContext(ctx).Raw(
			"SELECT TABLE_NAME AS table_name FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = ? AND COLUMN_NAME = 'deleted_at' ORDER BY TABLE_NAME",
			database.DatabaseName,
		).Scan(&tables).Error; err != nil {
			report.FailedTables++
			logger.Warnf(ctx, "[AppDatabaseCleanup] discover tables failed database=%s: %v", database.DatabaseName, err)
			continue
		}

		for _, table := range tables {
			report.Tables++
			enabled := cfg.Enabled
			mode := cfg.Mode
			retentionDays := cfg.RetentionDays
			if override, ok := overrideByTable[table.TableName]; ok {
				enabled = override.Enabled
				mode = override.Mode
				retentionDays = override.RetentionDays
			}
			if !enabled {
				continue
			}
			cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
			qualifiedTable := quoteMySQLIdentifier(database.DatabaseName) + "." + quoteMySQLIdentifier(table.TableName)
			var candidates int64
			countSQL := "SELECT COUNT(*) FROM " + qualifiedTable + " WHERE deleted_at IS NOT NULL AND deleted_at < ?"
			if err := adminDB.WithContext(ctx).Raw(countSQL, cutoff).Scan(&candidates).Error; err != nil {
				report.FailedTables++
				logger.Warnf(ctx, "[AppDatabaseCleanup] count failed database=%s table=%s: %v", database.DatabaseName, table.TableName, err)
				continue
			}
			report.CandidateRows += candidates
			if candidates == 0 || mode != "purge" {
				continue
			}

			deleteSQL := "DELETE FROM " + qualifiedTable + " WHERE deleted_at IS NOT NULL AND deleted_at < ? ORDER BY deleted_at ASC LIMIT ?"
			for {
				result := adminDB.WithContext(ctx).Exec(deleteSQL, cutoff, cfg.BatchSize)
				if result.Error != nil {
					report.FailedTables++
					logger.Warnf(ctx, "[AppDatabaseCleanup] purge failed database=%s table=%s: %v", database.DatabaseName, table.TableName, result.Error)
					break
				}
				report.PurgedRows += result.RowsAffected
				if result.RowsAffected < int64(cfg.BatchSize) {
					break
				}
				select {
				case <-ctx.Done():
					return report, ctx.Err()
				default:
				}
			}
		}
	}
	return report, nil
}
