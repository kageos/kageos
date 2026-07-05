package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
)

type releaseBinaryCleanupStats struct {
	scanned int
	kept    int
	removed int
	skipped int
}

func (s *AppManageService) releaseBinaryCleanup(ctx context.Context) {
	apps, err := s.getAllApps(ctx)
	if err != nil {
		logger.Errorf(ctx, "[ReleaseBinaryCleanup] 获取应用列表失败: %v", err)
		return
	}

	var total releaseBinaryCleanupStats
	for _, app := range apps {
		stats, err := s.cleanupReleaseBinariesForApp(ctx, app.User, app.App, maxKeepVersions)
		if err != nil {
			logger.Warnf(ctx, "[ReleaseBinaryCleanup] 清理失败: %s/%s err=%v", app.User, app.App, err)
			continue
		}
		total.scanned += stats.scanned
		total.kept += stats.kept
		total.removed += stats.removed
		total.skipped += stats.skipped
	}

	if total.scanned > 0 || total.removed > 0 {
		logger.Infof(ctx, "[ReleaseBinaryCleanup] 完成 | 扫描=%d 保留=%d 删除=%d 跳过=%d",
			total.scanned, total.kept, total.removed, total.skipped)
	}
}

func (s *AppManageService) cleanupReleaseBinariesForApp(ctx context.Context, user, app string, keepLatest int) (releaseBinaryCleanupStats, error) {
	var stats releaseBinaryCleanupStats
	if keepLatest <= 0 {
		keepLatest = maxKeepVersions
	}

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), user, app)
	versionData, err := s.readVersionData(appPaths.VersionJSONPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Debugf(ctx, "[ReleaseBinaryCleanup] version.json 不存在，跳过: %s/%s", user, app)
			return stats, nil
		}
		return stats, err
	}

	keepVersions := releaseVersionsToKeep(versionData, keepLatest)
	knownBinaries := s.knownReleaseBinaryVersions(user, app, versionData)

	binaryStats, err := s.cleanupKnownReleaseFiles(ctx, appPaths.BuildOutputDir(s.config.GetBuildOutputDir()), knownBinaries, keepVersions, user, app, "二进制")
	if err != nil {
		return stats, err
	}
	logStats, err := s.cleanupKnownReleaseFiles(ctx, appPaths.LogsDir(), knownReleaseLogVersions(appPaths, versionData), keepVersions, user, app, "日志")
	if err != nil {
		return stats, err
	}
	stats.add(binaryStats)
	stats.add(logStats)

	return stats, nil
}

func (s *AppManageService) cleanupKnownReleaseFiles(
	ctx context.Context,
	dir string,
	knownVersions map[string]string,
	keepVersions map[string]struct{},
	user, app, label string,
) (releaseBinaryCleanupStats, error) {
	var stats releaseBinaryCleanupStats
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return stats, nil
		}
		return stats, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			stats.skipped++
			continue
		}

		stats.scanned++
		version, ok := knownVersions[entry.Name()]
		if !ok && label == "日志" {
			version, ok = matchKnownReleaseLogBackup(entry.Name(), knownVersions)
		}
		if !ok {
			stats.skipped++
			continue
		}
		if _, keep := keepVersions[version]; keep {
			stats.kept++
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			return stats, err
		}
		stats.removed++
		logger.Infof(ctx, "[ReleaseBinaryCleanup] 已删除旧 release %s: %s/%s %s path=%s", label, user, app, version, path)
	}

	return stats, nil
}

func matchKnownReleaseLogBackup(name string, knownVersions map[string]string) (string, bool) {
	if !(strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".log.gz")) {
		return "", false
	}
	for marker, version := range knownVersions {
		if strings.HasSuffix(marker, "-") && strings.HasPrefix(name, marker) {
			return version, true
		}
	}
	return "", false
}

func (s *releaseBinaryCleanupStats) add(other releaseBinaryCleanupStats) {
	s.scanned += other.scanned
	s.kept += other.kept
	s.removed += other.removed
	s.skipped += other.skipped
}

func (s *AppManageService) knownReleaseBinaryVersions(user, app string, versionData *VersionData) map[string]string {
	known := make(map[string]string)
	if versionData == nil {
		return known
	}

	add := func(version string) {
		version = strings.TrimSpace(version)
		if version == "" {
			return
		}
		known[s.appBinaryName(user, app, version)] = version
	}

	add(versionData.CurrentVersion)
	add(versionData.LatestVersion)
	for _, info := range versionData.Versions {
		add(info.Version)
	}
	return known
}

func knownReleaseLogVersions(appPaths runtimeAppPaths, versionData *VersionData) map[string]string {
	known := make(map[string]string)
	if versionData == nil {
		return known
	}

	add := func(version string) {
		version = strings.TrimSpace(version)
		if version == "" {
			return
		}
		logFileName := appPaths.LogFileName(version)
		known[logFileName] = version
		known[strings.TrimSuffix(logFileName, ".log")+"-"] = version
	}

	add(versionData.CurrentVersion)
	add(versionData.LatestVersion)
	for _, info := range versionData.Versions {
		add(info.Version)
	}
	return known
}

func releaseVersionsToKeep(versionData *VersionData, keepLatest int) map[string]struct{} {
	keep := make(map[string]struct{})
	if versionData == nil {
		return keep
	}
	if keepLatest <= 0 {
		keepLatest = maxKeepVersions
	}

	add := func(version string) {
		version = strings.TrimSpace(version)
		if version != "" {
			keep[version] = struct{}{}
		}
	}
	add(versionData.CurrentVersion)
	add(versionData.LatestVersion)

	versions := uniqueReleaseVersions(versionData.Versions)
	sort.Slice(versions, func(i, j int) bool {
		leftNum := parseRuntimeVersionNum(versions[i])
		rightNum := parseRuntimeVersionNum(versions[j])
		if leftNum != rightNum {
			return leftNum > rightNum
		}
		return versions[i] > versions[j]
	})

	for i, version := range versions {
		if i >= keepLatest {
			break
		}
		add(version)
	}
	return keep
}

func uniqueReleaseVersions(infos []VersionInfo) []string {
	seen := make(map[string]struct{})
	versions := make([]string, 0, len(infos))
	for _, info := range infos {
		version := strings.TrimSpace(info.Version)
		if version == "" {
			continue
		}
		if _, ok := seen[version]; ok {
			continue
		}
		seen[version] = struct{}{}
		versions = append(versions, version)
	}
	return versions
}
