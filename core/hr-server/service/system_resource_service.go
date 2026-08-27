package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

const (
	runtimeSampleInterval     = 30 * time.Second
	runtimePersistInterval    = 10 * time.Minute
	platformSampleInterval    = time.Hour
	platformRetryInterval     = time.Minute
	resourceRetention         = 30 * 24 * time.Hour
	capacityRunHour           = 2
	capacityRunMinute         = 30
	capacityCollectionTimeout = 30 * time.Minute
	capacityRetryInterval     = 5 * time.Minute
)

type SystemResourceService struct {
	repo         *repository.SystemResourceRepository
	collector    systemResourceCollector
	natsConn     *nats.Conn
	mu           sync.Mutex
	capacityMu   sync.Mutex
	lastRuntime  *dto.SystemResourceSnapshot
	lastCapacity *dto.SystemResourceSnapshot
	lastPlatform *dto.SystemPlatformMetrics
	tasks        map[string]dto.SystemCollectionTaskStatus
	accumulator  runtimeAccumulator
	lastPersist  time.Time
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewSystemResourceService(repo *repository.SystemResourceRepository, connections ...*nats.Conn) *SystemResourceService {
	result := &SystemResourceService{
		repo: repo, collector: newLocalSystemResourceCollector(),
		tasks: map[string]dto.SystemCollectionTaskStatus{
			"runtime": {Key: "runtime", Status: "pending"}, "platform": {Key: "platform", Status: "pending"}, "capacity": {Key: "capacity", Status: "pending"},
		},
	}
	if len(connections) > 0 {
		result.natsConn = connections[0]
	}
	return result
}

func (s *SystemResourceService) Start(ctx context.Context) {
	if s == nil || s.cancel != nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(3)
	go s.runtimeLoop(workerCtx)
	go s.platformLoop(workerCtx)
	go s.capacityLoop(workerCtx)
}

func (s *SystemResourceService) Stop() {
	if s == nil || s.cancel == nil {
		return
	}
	s.cancel()
	s.wg.Wait()
	s.cancel = nil
}

func (s *SystemResourceService) runtimeLoop(ctx context.Context) {
	defer s.wg.Done()
	s.collectRuntime(ctx, true)
	ticker := time.NewTicker(runtimeSampleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collectRuntime(ctx, false)
		}
	}
}

func (s *SystemResourceService) platformLoop(ctx context.Context) {
	defer s.wg.Done()
	nextInterval := platformRetryInterval
	if s.collectPlatform(ctx) {
		nextInterval = platformSampleInterval
	}
	for {
		next := time.Now().Add(nextInterval)
		s.setTaskNextRun("platform", next)
		timer := time.NewTimer(nextInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if s.collectPlatform(ctx) {
				nextInterval = platformSampleInterval
			} else {
				nextInterval = platformRetryInterval
			}
		}
	}
}

func (s *SystemResourceService) capacityLoop(ctx context.Context) {
	defer s.wg.Done()
	if !s.collectCapacity(ctx) {
		retry := time.Now().Add(capacityRetryInterval)
		s.setTaskNextRun("capacity", retry)
		timer := time.NewTimer(capacityRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.collectCapacity(ctx)
		}
	}
	for {
		next := nextDailyRun(time.Now(), capacityRunHour, capacityRunMinute)
		s.setTaskNextRun("capacity", next)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.collectCapacity(ctx)
		}
	}
}

func (s *SystemResourceService) collectRuntime(ctx context.Context, forcePersist bool) {
	started := time.Now()
	s.markTaskStarted("runtime", started)
	snapshot, err := s.collector.CollectRuntime()
	if err != nil {
		s.markTaskFinished("runtime", started, err, started.Add(runtimeSampleInterval))
		logger.Warnf(ctx, "[SystemResourceMonitor] runtime collection failed: %v", err)
		return
	}
	s.mu.Lock()
	s.lastRuntime = &snapshot
	s.accumulator.Add(snapshot)
	shouldPersist := forcePersist || s.lastPersist.IsZero() || snapshot.CollectedAt.Sub(s.lastPersist) >= runtimePersistInterval
	var sample *model.SystemResourceSample
	if shouldPersist {
		value := s.accumulator.Sample()
		sample = &value
	}
	s.mu.Unlock()
	if sample != nil {
		if err := s.repo.Create(sample); err != nil {
			logger.Warnf(ctx, "[SystemResourceMonitor] persist runtime rollup failed: %v", err)
			s.markTaskPartial("runtime", started, "runtime metrics collected but history persistence failed", started.Add(runtimeSampleInterval))
			return
		}
		s.mu.Lock()
		s.accumulator.Reset()
		s.lastPersist = snapshot.CollectedAt
		s.mu.Unlock()
	}
	s.markTaskFinished("runtime", started, nil, started.Add(runtimeSampleInterval))
}

func (s *SystemResourceService) collectPlatform(ctx context.Context) bool {
	started := time.Now()
	s.markTaskStarted("platform", started)
	metrics, err := s.repo.CollectPlatformMetrics(started)
	if err == nil {
		if stats, ok := s.requestPlatformStats(subjects.PlatformAppStatsQuerySubject); ok {
			metrics.WorkspacesTotal, metrics.WorkspacesEnabled = stats.WorkspacesTotal, stats.WorkspacesEnabled
			metrics.ServiceDirectories, metrics.FunctionsTotal, metrics.AppStatsAvailable = stats.ServiceDirectories, stats.FunctionsTotal, true
		}
		if stats, ok := s.requestPlatformStats(subjects.PlatformRuntimeStatsQuerySubject); ok {
			metrics.AppDatabasesTotal, metrics.RuntimeStatsAvailable = stats.AppDatabasesTotal, true
		}
		if stats, ok := s.requestPlatformStats(subjects.PlatformTimerStatsQuerySubject); ok {
			metrics.ScheduledTasksTotal, metrics.ScheduledTasksActive, metrics.TimerStatsAvailable = stats.ScheduledTasksTotal, stats.ScheduledTasksActive, true
		}
	}
	if err == nil {
		err = s.repo.CreatePlatform(metrics)
	}
	if err != nil {
		s.markTaskFinished("platform", started, err, started.Add(platformRetryInterval))
		logger.Warnf(ctx, "[SystemResourceMonitor] platform collection failed: %v", err)
		return false
	}
	s.mu.Lock()
	s.lastPlatform = &metrics
	s.mu.Unlock()
	if !metrics.AppStatsAvailable || !metrics.RuntimeStatsAvailable || !metrics.TimerStatsAvailable {
		s.markTaskPartial("platform", started, "one or more platform metric sources are unavailable", started.Add(platformRetryInterval))
		return false
	}
	s.markTaskFinished("platform", started, nil, started.Add(platformSampleInterval))
	return true
}

func (s *SystemResourceService) collectCapacity(ctx context.Context) bool {
	s.capacityMu.Lock()
	defer s.capacityMu.Unlock()
	started := time.Now()
	s.markTaskStarted("capacity", started)
	capacityCtx, cancel := context.WithTimeout(ctx, capacityCollectionTimeout)
	defer cancel()
	snapshot, err := s.collector.CollectCapacity(capacityCtx)
	remoteDatabaseAvailable := false
	if err == nil {
		snapshot.DatabaseLogicalBytes, snapshot.LargestDatabases, snapshot.DatabaseSizeAvailable = s.repo.CollectDatabaseSizes(capacityCtx)
		if remote, ok := s.requestDatabaseStats(subjects.PlatformDatabaseStatsQuerySubject); ok {
			remoteDatabaseAvailable = true
			if remote.Available {
				snapshot.DatabaseSizeAvailable = true
				snapshot.DatabaseLogicalBytes += remote.TotalBytes
				snapshot.LargestDatabases = append(snapshot.LargestDatabases, remote.Databases...)
				sort.Slice(snapshot.LargestDatabases, func(i, j int) bool {
					return snapshot.LargestDatabases[i].UsedBytes > snapshot.LargestDatabases[j].UsedBytes
				})
				if len(snapshot.LargestDatabases) > 10 {
					snapshot.LargestDatabases = snapshot.LargestDatabases[:10]
				}
			}
		}
	}
	if err == nil {
		err = s.repo.CreateCapacity(snapshot)
	}
	cleanupErr := s.repo.DeleteBefore(time.Now().Add(-resourceRetention))
	if err == nil {
		err = cleanupErr
	}
	next := nextDailyRun(time.Now(), capacityRunHour, capacityRunMinute)
	if err != nil {
		s.markTaskFinished("capacity", started, err, next)
		logger.Warnf(ctx, "[SystemResourceMonitor] capacity collection failed: %v", err)
		return false
	}
	s.mu.Lock()
	s.lastCapacity = &snapshot
	s.mu.Unlock()
	if !remoteDatabaseAvailable {
		s.markTaskPartial("capacity", started, "application database metric source is unavailable", time.Now().Add(capacityRetryInterval))
		return false
	}
	s.markTaskFinished("capacity", started, nil, next)
	return true
}

func (s *SystemResourceService) Overview(historyHours int, includeHistory bool) (*dto.SystemResourceOverviewResp, error) {
	if historyHours < 24 {
		historyHours = 24
	}
	if historyHours > 24*30 {
		historyHours = 24 * 30
	}
	current, platform, tasks, err := s.currentState()
	if err != nil {
		return nil, err
	}
	history := []dto.SystemResourceHistoryPoint{}
	if includeHistory {
		samples, historyErr := s.repo.History(time.Now().Add(-time.Duration(historyHours)*time.Hour), historyHours*6+12)
		if historyErr != nil {
			return nil, fmt.Errorf("load resource history: %w", historyErr)
		}
		history = make([]dto.SystemResourceHistoryPoint, 0, len(samples)+1)
		for _, sample := range samples {
			history = append(history, historyPoint(sample))
		}
		if len(history) == 0 || current.CollectedAt.Sub(history[len(history)-1].CollectedAt) > runtimePersistInterval {
			history = append(history, dto.SystemResourceHistoryPoint{CollectedAt: current.CollectedAt, DiskUsedBytes: current.DiskUsedBytes, DiskUsedPercent: current.DiskUsedPercent, MemoryUsedPercent: current.MemoryUsedPercent, CPUUsedPercent: current.CPUUsedPercent, CPUMaxPercent: current.CPUUsedPercent, NetworkRxBytesPS: current.NetworkRxBytesPS, NetworkTxBytesPS: current.NetworkTxBytesPS, DiskReadBytesPS: current.DiskReadBytesPS, DiskWriteBytesPS: current.DiskWriteBytesPS, Load1: current.Load1})
		}
	}
	return &dto.SystemResourceOverviewResp{Current: *current, History: history, HistoryHours: historyHours, SampleIntervalMinutes: int(runtimePersistInterval / time.Minute), RuntimeIntervalSeconds: int(runtimeSampleInterval / time.Second), Forecast: buildStorageForecast(*current, history), Platform: platform, CollectionTasks: tasks}, nil
}

func (s *SystemResourceService) currentState() (*dto.SystemResourceSnapshot, dto.SystemPlatformMetrics, []dto.SystemCollectionTaskStatus, error) {
	s.mu.Lock()
	runtimeSnapshot, capacitySnapshot, platform := s.lastRuntime, s.lastCapacity, s.lastPlatform
	s.mu.Unlock()
	if runtimeSnapshot == nil {
		value, err := s.collector.CollectRuntime()
		if err != nil {
			return nil, dto.SystemPlatformMetrics{}, nil, fmt.Errorf("collect runtime resources: %w", err)
		}
		runtimeSnapshot = &value
	}
	if capacitySnapshot == nil {
		if stored, err := s.repo.LatestCapacity(); err == nil {
			capacitySnapshot = stored
		} else {
			// Capacity scanning can recursively walk large directories. Never make
			// an HTTP request wait for it; the background task will replace this
			// lightweight filesystem snapshot when its first scan completes.
			capacitySnapshot = runtimeSnapshot
		}
	}
	if platform == nil {
		if stored, err := s.repo.LatestPlatform(); err == nil {
			platform = stored
		} else if value, collectErr := s.repo.CollectPlatformMetrics(time.Now()); collectErr == nil {
			platform = &value
		} else {
			return nil, dto.SystemPlatformMetrics{}, nil, fmt.Errorf("collect platform metrics: %w", collectErr)
		}
	}
	current := *runtimeSnapshot
	mergeCapacitySnapshot(&current, *capacitySnapshot)
	return &current, *platform, s.taskStatuses(), nil
}

func (s *SystemResourceService) requestPlatformStats(subject string) (dto.SystemPlatformServiceStats, bool) {
	if s.natsConn == nil {
		return dto.SystemPlatformServiceStats{}, false
	}
	message, err := s.natsConn.Request(subject, nil, 2*time.Second)
	if err != nil {
		return dto.SystemPlatformServiceStats{}, false
	}
	response, ok := decodePlatformStats(message.Data)
	if !ok {
		return dto.SystemPlatformServiceStats{}, false
	}
	return response, true
}

func (s *SystemResourceService) requestDatabaseStats(subject string) (dto.SystemDatabaseCapacityStats, bool) {
	if s.natsConn == nil {
		return dto.SystemDatabaseCapacityStats{}, false
	}
	message, err := s.natsConn.Request(subject, nil, 15*time.Second)
	if err != nil {
		return dto.SystemDatabaseCapacityStats{}, false
	}
	return decodeDatabaseStats(message.Data)
}

func decodePlatformStats(data []byte) (dto.SystemPlatformServiceStats, bool) {
	var envelope struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Error != "" {
		return dto.SystemPlatformServiceStats{}, false
	}
	var response dto.SystemPlatformServiceStats
	if json.Unmarshal(data, &response) != nil {
		return dto.SystemPlatformServiceStats{}, false
	}
	return response, true
}

func decodeDatabaseStats(data []byte) (dto.SystemDatabaseCapacityStats, bool) {
	var envelope struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Error != "" {
		return dto.SystemDatabaseCapacityStats{}, false
	}
	var response dto.SystemDatabaseCapacityStats
	if json.Unmarshal(data, &response) != nil {
		return dto.SystemDatabaseCapacityStats{}, false
	}
	return response, true
}

func (s *SystemResourceService) markTaskStarted(key string, started time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tasks[key]
	status.Status, status.LastStartedAt, status.Error = "running", timePointer(started), ""
	s.tasks[key] = status
}
func (s *SystemResourceService) markTaskFinished(key string, started time.Time, err error, next time.Time) {
	finished := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tasks[key]
	status.DurationMillis, status.NextRunAt = finished.Sub(started).Milliseconds(), timePointer(next)
	if err != nil {
		status.Status, status.Error = "failed", err.Error()
	} else {
		status.Status, status.Error, status.LastSucceededAt = "success", "", timePointer(finished)
	}
	s.tasks[key] = status
}

func (s *SystemResourceService) markTaskPartial(key string, started time.Time, message string, next time.Time) {
	finished := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tasks[key]
	status.Status, status.Error = "partial", message
	status.DurationMillis, status.NextRunAt, status.LastSucceededAt = finished.Sub(started).Milliseconds(), timePointer(next), timePointer(finished)
	s.tasks[key] = status
}
func (s *SystemResourceService) setTaskNextRun(key string, next time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := s.tasks[key]
	status.NextRunAt = timePointer(next)
	s.tasks[key] = status
}
func (s *SystemResourceService) taskStatuses() []dto.SystemCollectionTaskStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]dto.SystemCollectionTaskStatus, 0, 3)
	for _, key := range []string{"runtime", "platform", "capacity"} {
		result = append(result, s.tasks[key])
	}
	return result
}
func timePointer(value time.Time) *time.Time { copy := value.UTC(); return &copy }
func nextDailyRun(now time.Time, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

type runtimeAccumulator struct {
	count                                                         int
	last                                                          dto.SystemResourceSnapshot
	cpuSum, cpuMax, netRxSum, netTxSum, diskReadSum, diskWriteSum float64
}

func (a *runtimeAccumulator) Add(snapshot dto.SystemResourceSnapshot) {
	a.count++
	a.last = snapshot
	a.cpuSum += snapshot.CPUUsedPercent
	a.netRxSum += snapshot.NetworkRxBytesPS
	a.netTxSum += snapshot.NetworkTxBytesPS
	a.diskReadSum += snapshot.DiskReadBytesPS
	a.diskWriteSum += snapshot.DiskWriteBytesPS
	if snapshot.CPUUsedPercent > a.cpuMax {
		a.cpuMax = snapshot.CPUUsedPercent
	}
}
func (a *runtimeAccumulator) Sample() model.SystemResourceSample {
	count := float64(a.count)
	if count == 0 {
		count = 1
	}
	return model.SystemResourceSample{CollectedAt: a.last.CollectedAt, DiskTotalBytes: a.last.DiskTotalBytes, DiskUsedBytes: a.last.DiskUsedBytes, DiskFreeBytes: a.last.DiskFreeBytes, DiskUsedPercent: a.last.DiskUsedPercent, MemoryTotalBytes: a.last.MemoryTotalBytes, MemoryUsedBytes: a.last.MemoryUsedBytes, MemoryUsedPercent: a.last.MemoryUsedPercent, CPUUsedPercent: a.cpuSum / count, CPUMaxPercent: a.cpuMax, NetworkRxBytesPS: a.netRxSum / count, NetworkTxBytesPS: a.netTxSum / count, DiskReadBytesPS: a.diskReadSum / count, DiskWriteBytesPS: a.diskWriteSum / count, Load1: a.last.Load1}
}
func (a *runtimeAccumulator) Reset() { *a = runtimeAccumulator{} }

func historyPoint(sample model.SystemResourceSample) dto.SystemResourceHistoryPoint {
	return dto.SystemResourceHistoryPoint{CollectedAt: sample.CollectedAt, DiskUsedBytes: sample.DiskUsedBytes, DiskUsedPercent: sample.DiskUsedPercent, MemoryUsedPercent: sample.MemoryUsedPercent, CPUUsedPercent: sample.CPUUsedPercent, CPUMaxPercent: sample.CPUMaxPercent, NetworkRxBytesPS: sample.NetworkRxBytesPS, NetworkTxBytesPS: sample.NetworkTxBytesPS, DiskReadBytesPS: sample.DiskReadBytesPS, DiskWriteBytesPS: sample.DiskWriteBytesPS, Load1: sample.Load1}
}

func buildStorageForecast(current dto.SystemResourceSnapshot, history []dto.SystemResourceHistoryPoint) dto.StorageExpansionForecast {
	poolKey, currentPercent := "primary", current.DiskUsedPercent
	for _, pool := range current.StoragePools {
		if pool.Available && pool.UsedPercent > currentPercent {
			poolKey, currentPercent = pool.Key, pool.UsedPercent
		}
	}
	forecast := dto.StorageExpansionForecast{Status: "healthy", PoolKey: poolKey, CurrentUsedPercent: currentPercent, TargetPercent: 85, Message: "Storage capacity is healthy"}
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
		forecast.Status, forecast.Message = "warning", "Storage is projected to reach the expansion threshold within 30 days"
	}
	return forecast
}
