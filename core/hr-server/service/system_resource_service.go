package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

const (
	resourceSampleInterval = 5 * time.Minute
	resourceRetention      = 90 * 24 * time.Hour
	resourceFreshness      = resourceSampleInterval
)

type SystemResourceService struct {
	repo      *repository.SystemResourceRepository
	collector systemResourceCollector
	mu        sync.Mutex
	last      *dto.SystemResourceSnapshot
	cancel    context.CancelFunc
	done      chan struct{}
}

func NewSystemResourceService(repo *repository.SystemResourceRepository) *SystemResourceService {
	return &SystemResourceService{repo: repo, collector: newLocalSystemResourceCollector()}
}

func (s *SystemResourceService) Start(ctx context.Context) {
	if s == nil || s.cancel != nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.collectAndStore(workerCtx)
		ticker := time.NewTicker(resourceSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				s.collectAndStore(workerCtx)
			}
		}
	}()
}

func (s *SystemResourceService) Stop() {
	if s == nil || s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
	s.cancel = nil
}

func (s *SystemResourceService) Overview(historyHours int) (*dto.SystemResourceOverviewResp, error) {
	if historyHours < 24 {
		historyHours = 24
	}
	if historyHours > 24*30 {
		historyHours = 24 * 30
	}
	current, err := s.currentSnapshot()
	if err != nil {
		return nil, err
	}
	samples, err := s.repo.History(time.Now().Add(-time.Duration(historyHours)*time.Hour), historyHours*12+12)
	if err != nil {
		return nil, fmt.Errorf("load resource history: %w", err)
	}
	history := make([]dto.SystemResourceHistoryPoint, 0, len(samples)+1)
	for _, sample := range samples {
		history = append(history, historyPoint(sample))
	}
	if len(history) == 0 || current.CollectedAt.Sub(history[len(history)-1].CollectedAt) > resourceFreshness {
		history = append(history, dto.SystemResourceHistoryPoint{
			CollectedAt: current.CollectedAt, DiskUsedBytes: current.DiskUsedBytes,
			DiskUsedPercent: current.DiskUsedPercent, MemoryUsedPercent: current.MemoryUsedPercent, Load1: current.Load1,
		})
	}
	return &dto.SystemResourceOverviewResp{
		Current: *current, History: history, HistoryHours: historyHours,
		SampleIntervalMinutes: int(resourceSampleInterval / time.Minute),
		Forecast:              buildStorageForecast(*current, history),
	}, nil
}

func (s *SystemResourceService) currentSnapshot() (*dto.SystemResourceSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.last != nil && time.Since(s.last.CollectedAt) < resourceFreshness {
		copy := *s.last
		return &copy, nil
	}
	snapshot, err := s.collector.Collect()
	if err != nil {
		return nil, fmt.Errorf("collect system resources: %w", err)
	}
	s.last = &snapshot
	return &snapshot, nil
}

func (s *SystemResourceService) collectAndStore(ctx context.Context) {
	snapshot, err := s.collector.Collect()
	if err != nil {
		logger.Warnf(ctx, "[SystemResourceMonitor] collect failed: %v", err)
		return
	}
	components, err := json.Marshal(snapshot.Components)
	if err != nil {
		logger.Warnf(ctx, "[SystemResourceMonitor] encode components failed: %v", err)
		return
	}
	sample := &model.SystemResourceSample{
		CollectedAt: snapshot.CollectedAt, DiskTotalBytes: snapshot.DiskTotalBytes,
		DiskUsedBytes: snapshot.DiskUsedBytes, DiskFreeBytes: snapshot.DiskFreeBytes,
		DiskUsedPercent: snapshot.DiskUsedPercent, MemoryTotalBytes: snapshot.MemoryTotalBytes,
		MemoryUsedBytes: snapshot.MemoryUsedBytes, MemoryUsedPercent: snapshot.MemoryUsedPercent,
		Load1: snapshot.Load1, ComponentsJSON: string(components),
	}
	if err := s.repo.Create(sample); err != nil {
		logger.Warnf(ctx, "[SystemResourceMonitor] persist sample failed: %v", err)
		return
	}
	s.mu.Lock()
	s.last = &snapshot
	s.mu.Unlock()
	if err := s.repo.DeleteBefore(time.Now().Add(-resourceRetention)); err != nil {
		logger.Warnf(ctx, "[SystemResourceMonitor] delete expired samples failed: %v", err)
	}
}

func historyPoint(sample model.SystemResourceSample) dto.SystemResourceHistoryPoint {
	return dto.SystemResourceHistoryPoint{
		CollectedAt: sample.CollectedAt, DiskUsedBytes: sample.DiskUsedBytes,
		DiskUsedPercent: sample.DiskUsedPercent, MemoryUsedPercent: sample.MemoryUsedPercent, Load1: sample.Load1,
	}
}

func buildStorageForecast(current dto.SystemResourceSnapshot, history []dto.SystemResourceHistoryPoint) dto.StorageExpansionForecast {
	poolKey, currentPercent := "primary", current.DiskUsedPercent
	for _, pool := range current.StoragePools {
		if pool.Available && pool.UsedPercent > currentPercent {
			poolKey, currentPercent = pool.Key, pool.UsedPercent
		}
	}
	forecast := dto.StorageExpansionForecast{
		Status: "healthy", PoolKey: poolKey, CurrentUsedPercent: currentPercent,
		TargetPercent: 85, Message: "Storage capacity is healthy",
	}
	if currentPercent >= 90 {
		forecast.Status, forecast.Message = "critical", "Storage usage is critical; expand capacity now"
		return forecast
	} else if currentPercent >= 80 {
		forecast.Status, forecast.Message = "warning", "Storage usage is high; prepare capacity expansion"
		return forecast
	}
	if len(history) < 2 || current.DiskTotalBytes == 0 {
		return forecast
	}
	first, last := history[0], history[len(history)-1]
	days := last.CollectedAt.Sub(first.CollectedAt).Hours() / 24
	// Avoid extrapolating a capacity date from a few noisy samples collected
	// immediately after installation.
	if days < 1 || last.DiskUsedBytes <= first.DiskUsedBytes {
		return forecast
	}
	forecast.DailyGrowthByte = float64(last.DiskUsedBytes-first.DiskUsedBytes) / days
	targetBytes := float64(current.DiskTotalBytes) * forecast.TargetPercent / 100
	remaining := targetBytes - float64(current.DiskUsedBytes)
	if remaining <= 0 {
		zero := 0
		forecast.DaysToTarget = &zero
		return forecast
	}
	daysToTarget := int(math.Ceil(remaining / forecast.DailyGrowthByte))
	forecast.DaysToTarget = &daysToTarget
	if daysToTarget <= 30 && forecast.Status == "healthy" {
		forecast.Status = "warning"
		forecast.Message = "Storage is projected to reach the expansion threshold within 30 days"
	}
	return forecast
}
