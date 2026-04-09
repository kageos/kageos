package store

import (
	"context"
	"errors"
	"fmt"

	backupmodel "github.com/ai-agent-os/ai-agent-os/core/backup-service/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/dbx"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(path string) (*Store, error) {
	db, err := dbx.OpenSQLite(path, dbx.OpenOptions{})
	if err != nil {
		return nil, fmt.Errorf("open backup sqlite: %w", err)
	}
	if err := backupmodel.InitTables(db); err != nil {
		return nil, fmt.Errorf("init backup tables: %w", err)
	}

	s := &Store{db: db}
	if err := s.ensureSystemState(context.Background()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) ensureSystemState(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var state backupmodel.SystemState
		err := tx.First(&state, backupmodel.DefaultSystemStateID).Error
		if err == nil {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		state.ID = backupmodel.DefaultSystemStateID
		return tx.Create(&state).Error
	})
}

func (s *Store) GetSystemState(ctx context.Context) (*backupmodel.SystemState, error) {
	var state backupmodel.SystemState
	if err := s.db.WithContext(ctx).First(&state, backupmodel.DefaultSystemStateID).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) UpdateSystemState(ctx context.Context, mutate func(*backupmodel.SystemState) error) (*backupmodel.SystemState, error) {
	var updated backupmodel.SystemState

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var state backupmodel.SystemState
		if err := tx.First(&state, backupmodel.DefaultSystemStateID).Error; err != nil {
			return err
		}
		if err := mutate(&state); err != nil {
			return err
		}
		if err := tx.Save(&state).Error; err != nil {
			return err
		}
		updated = state
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *Store) CreateTask(ctx context.Context, task *backupmodel.Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

func (s *Store) SaveTask(ctx context.Context, task *backupmodel.Task) error {
	return s.db.WithContext(ctx).Save(task).Error
}

func (s *Store) GetTask(ctx context.Context, id int64) (*backupmodel.Task, error) {
	var task backupmodel.Task
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) ListTasks(ctx context.Context, limit int) ([]backupmodel.Task, error) {
	if limit <= 0 {
		limit = 20
	}

	var tasks []backupmodel.Task
	err := s.db.WithContext(ctx).
		Order("id desc").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

func (s *Store) CreateSnapshot(ctx context.Context, snapshot *backupmodel.Snapshot) error {
	return s.db.WithContext(ctx).Create(snapshot).Error
}

func (s *Store) SaveSnapshot(ctx context.Context, snapshot *backupmodel.Snapshot) error {
	return s.db.WithContext(ctx).Save(snapshot).Error
}

func (s *Store) GetSnapshot(ctx context.Context, id int64) (*backupmodel.Snapshot, error) {
	var snapshot backupmodel.Snapshot
	if err := s.db.WithContext(ctx).First(&snapshot, id).Error; err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *Store) DeleteSnapshot(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&backupmodel.Snapshot{}, id).Error
}

func (s *Store) ListSnapshots(ctx context.Context, resourceType string, limit int) ([]backupmodel.Snapshot, error) {
	if limit <= 0 {
		limit = 20
	}

	query := s.db.WithContext(ctx).Order("id desc").Limit(limit)
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}

	var snapshots []backupmodel.Snapshot
	err := query.Find(&snapshots).Error
	return snapshots, err
}
