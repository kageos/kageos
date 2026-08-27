package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestCollectStorageComponentsReportsKnownDirectories(t *testing.T) {
	root := t.TempDir()
	writeSizedFile(t, filepath.Join(root, "mysql", "database.bin"), 1024)
	writeSizedFile(t, filepath.Join(root, "namespace", "alice", "app", "release.bin"), 2048)

	components := collectLocalStorageComponents(context.Background(), root, "primary")
	byKey := make(map[string]dto.SystemResourceComponent, len(components))
	for _, component := range components {
		byKey[component.Key] = component
	}
	if got := byKey["mysql"]; !got.Available || got.UsedBytes != 1024 {
		t.Fatalf("mysql component = %+v, want available 1024 bytes", got)
	}
	if got := byKey["workspaces"]; !got.Available || got.UsedBytes != 2048 {
		t.Fatalf("workspaces component = %+v, want available 2048 bytes", got)
	}
	if got := byKey["object_storage"]; got.Available || got.UsedBytes != 0 {
		t.Fatalf("object storage component = %+v, want unavailable", got)
	}
}

func TestParseContainerVolumeSizes(t *testing.T) {
	output := `Images space usage:

Local Volumes space usage:

VOLUME NAME                       LINKS  SIZE
kageos-dev-mysql3318-data         1      21.81GB
kageos-infra-minio-data           1      4.961GB
`
	volumes := parseContainerVolumeSizes(output)
	if got := volumes["kageos-dev-mysql3318-data"]; got != 21810000000 {
		t.Fatalf("mysql volume size = %d, want 21810000000", got)
	}
	if got := volumes["kageos-infra-minio-data"]; got != 4961000000 {
		t.Fatalf("minio volume size = %d, want 4961000000", got)
	}
}

func TestAppendUnclassifiedComponentsKeepsStoragePoolsSeparate(t *testing.T) {
	snapshot := dto.SystemResourceSnapshot{
		StoragePools: []dto.SystemStoragePool{
			{Key: "primary", UsedBytes: 1000, Available: true},
			{Key: "container_engine", UsedBytes: 500, Available: true},
		},
		Components: []dto.SystemResourceComponent{
			{Key: "workspaces", PoolKey: "primary", UsedBytes: 200, Available: true},
			{Key: "mysql", PoolKey: "container_engine", UsedBytes: 100, Available: true},
		},
	}
	appendUnclassifiedComponents(&snapshot)
	if got := snapshot.Components[2]; got.Key != "other" || got.PoolKey != "primary" || got.UsedBytes != 800 {
		t.Fatalf("primary other = %+v", got)
	}
	if got := snapshot.Components[3]; got.Key != "container_other" || got.PoolKey != "container_engine" || got.UsedBytes != 400 {
		t.Fatalf("container other = %+v", got)
	}
}

func TestCollectContainerEngineStorageKeepsKnownUsageWhenCapacityIsUnavailable(t *testing.T) {
	runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "-v" {
			return []byte("Local Volumes space usage:\n\nVOLUME NAME LINKS SIZE\nkageos-dev-mysql3318-data 1 0B\n"), nil
		}
		if len(args) >= 3 && args[0] == "system" && args[1] == "df" && args[2] == "--format" {
			return []byte("{\"Type\":\"Images\",\"RawSize\":1024}\n{\"Type\":\"Containers\",\"RawSize\":256}\n"), nil
		}
		return nil, os.ErrNotExist
	}

	engine := collectContainerEngineStorage(context.Background(), "docker", runner)
	if !engine.available || engine.pool.Key != "container_engine" || engine.pool.Available {
		t.Fatalf("docker storage detection = %+v", engine)
	}
	if engine.pool.UsedBytes != 1280 {
		t.Fatalf("known docker usage = %d, want 1280", engine.pool.UsedBytes)
	}
	components := collectLocalStorageComponents(context.Background(), t.TempDir(), "primary")
	overlayEngineComponents(&components, engine)
	for _, component := range components {
		if component.Key == "mysql" && (!component.Available || component.PoolKey != "container_engine" || component.UsedBytes != 0) {
			t.Fatalf("zero-byte known mysql volume = %+v", component)
		}
	}
}

func TestBuildStorageForecastWarnsBeforeThreshold(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	day := uint64(1 << 30)
	history := []dto.SystemResourceHistoryPoint{
		{CollectedAt: start, DiskUsedBytes: 60 * day},
		{CollectedAt: start.Add(10 * 24 * time.Hour), DiskUsedBytes: 70 * day},
	}
	current := dto.SystemResourceSnapshot{DiskTotalBytes: 100 * day, DiskUsedBytes: 70 * day, DiskUsedPercent: 70}

	forecast := buildStorageForecast(current, history)
	if forecast.Status != "warning" {
		t.Fatalf("status = %q, want warning", forecast.Status)
	}
	if forecast.DaysToTarget == nil || *forecast.DaysToTarget != 15 {
		t.Fatalf("days_to_target = %v, want 15", forecast.DaysToTarget)
	}
	if forecast.DailyGrowthByte != float64(day) {
		t.Fatalf("daily_growth_bytes = %f, want %d", forecast.DailyGrowthByte, day)
	}
}

func TestBuildStorageForecastMarksCriticalUsage(t *testing.T) {
	forecast := buildStorageForecast(dto.SystemResourceSnapshot{DiskTotalBytes: 100, DiskUsedBytes: 91, DiskUsedPercent: 91}, nil)
	if forecast.Status != "critical" {
		t.Fatalf("status = %q, want critical", forecast.Status)
	}
}

func TestBuildStorageForecastUsesMostConstrainedStoragePool(t *testing.T) {
	forecast := buildStorageForecast(dto.SystemResourceSnapshot{
		DiskUsedPercent: 40,
		StoragePools: []dto.SystemStoragePool{
			{Key: "primary", UsedPercent: 40, Available: true},
			{Key: "container_engine", UsedPercent: 92, Available: true},
		},
	}, nil)
	if forecast.Status != "critical" || forecast.PoolKey != "container_engine" || forecast.CurrentUsedPercent != 92 {
		t.Fatalf("forecast = %+v, want critical container engine storage", forecast)
	}
}

func TestRuntimeAccumulatorBuildsTenMinuteRollup(t *testing.T) {
	var accumulator runtimeAccumulator
	accumulator.Add(dto.SystemResourceSnapshot{CollectedAt: time.Unix(1, 0), CPUUsedPercent: 20, NetworkRxBytesPS: 100, DiskReadBytesPS: 40})
	accumulator.Add(dto.SystemResourceSnapshot{CollectedAt: time.Unix(2, 0), CPUUsedPercent: 60, NetworkRxBytesPS: 300, DiskReadBytesPS: 80})
	sample := accumulator.Sample()
	if sample.CPUUsedPercent != 40 || sample.CPUMaxPercent != 60 {
		t.Fatalf("cpu rollup = avg %.1f max %.1f", sample.CPUUsedPercent, sample.CPUMaxPercent)
	}
	if sample.NetworkRxBytesPS != 200 || sample.DiskReadBytesPS != 60 {
		t.Fatalf("io rollup = network %.1f disk %.1f", sample.NetworkRxBytesPS, sample.DiskReadBytesPS)
	}
}

func TestNextDailyRunUsesNextLocal0230(t *testing.T) {
	location := time.FixedZone("test", 8*60*60)
	before := time.Date(2026, 8, 27, 1, 0, 0, 0, location)
	if got := nextDailyRun(before, 2, 30); !got.Equal(time.Date(2026, 8, 27, 2, 30, 0, 0, location)) {
		t.Fatalf("next run before schedule = %v", got)
	}
	after := time.Date(2026, 8, 27, 3, 0, 0, 0, location)
	if got := nextDailyRun(after, 2, 30); !got.Equal(time.Date(2026, 8, 28, 2, 30, 0, 0, location)) {
		t.Fatalf("next run after schedule = %v", got)
	}
}

func TestDecodePlatformStatsRejectsErrorEnvelope(t *testing.T) {
	if _, ok := decodePlatformStats([]byte(`{"error":"query failed"}`)); ok {
		t.Fatal("error response must not be accepted as platform stats")
	}
	stats, ok := decodePlatformStats([]byte(`{"workspaces_total":3,"functions_total":8}`))
	if !ok || stats.WorkspacesTotal != 3 || stats.FunctionsTotal != 8 {
		t.Fatalf("decoded platform stats = %+v, available=%v", stats, ok)
	}
}

func TestDecodeDatabaseStatsDistinguishesDisabledFromUnavailable(t *testing.T) {
	stats, ok := decodeDatabaseStats([]byte(`{"available":false,"total_bytes":0,"databases":[]}`))
	if !ok || stats.Available {
		t.Fatalf("disabled database stats = %+v, source available=%v", stats, ok)
	}
	if _, ok := decodeDatabaseStats([]byte(`{"error":"query failed"}`)); ok {
		t.Fatal("error response must not be accepted as database stats")
	}
}

func TestOverviewCanSkipHistoricalQueryForLiveRefresh(t *testing.T) {
	now := time.Now().UTC()
	service := &SystemResourceService{
		lastRuntime:  &dto.SystemResourceSnapshot{CollectedAt: now, DiskTotalBytes: 100, DiskUsedBytes: 40, DiskFreeBytes: 60, DiskUsedPercent: 40},
		lastCapacity: &dto.SystemResourceSnapshot{CollectedAt: now},
		lastPlatform: &dto.SystemPlatformMetrics{CollectedAt: now},
		tasks:        map[string]dto.SystemCollectionTaskStatus{},
	}
	overview, err := service.Overview(24*30, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.History) != 0 || overview.HistoryHours != 24*30 {
		t.Fatalf("lightweight overview history = %d points, %d hours", len(overview.History), overview.HistoryHours)
	}
}

func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}
