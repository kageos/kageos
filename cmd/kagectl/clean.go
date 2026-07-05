package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const defaultCleanKeepVersions = 3

type cleanTarget struct {
	Kind    string
	App     string
	Version string
	Path    string
	Size    int64
	Reason  string
	IsDir   bool
}

type cleanVersionInfo struct {
	Version string `json:"version"`
}

type cleanVersionData struct {
	User           string             `json:"user"`
	App            string             `json:"app"`
	CurrentVersion string             `json:"current_version"`
	LatestVersion  string             `json:"latest_version"`
	Versions       []cleanVersionInfo `json:"versions"`
}

func cmdClean(paths Paths, args []string) error {
	opts, err := parseCleanFlags(args)
	if err != nil {
		return err
	}

	var targets []cleanTarget
	switch opts.Target {
	case "runtime":
		namespaceRoot, err := cleanNamespaceRoot(paths, opts.NamespacePath)
		if err != nil {
			return err
		}
		targets, err = planRuntimeCleanups(namespaceRoot, opts.Keep)
		if err != nil {
			return err
		}
		fmt.Printf("Clean runtime namespace: %s\n", namespaceRoot)
	case "logs":
		namespaceRoot, err := cleanNamespaceRoot(paths, opts.NamespacePath)
		if err != nil {
			return err
		}
		targets, err = planSourceLogCleanups(paths.RepoRoot, namespaceRoot)
		if err != nil {
			return err
		}
		fmt.Printf("Clean source logs: repo=%s namespace=%s\n", paths.RepoRoot, namespaceRoot)
	default:
		return fmt.Errorf("clean supports runtime or logs, got %q", opts.Target)
	}

	return applyCleanTargets(targets, opts.Execute)
}

func cleanNamespaceRoot(paths Paths, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return resolveRelativePath(paths.RepoRoot, override), nil
	}

	repoNamespace := filepath.Join(paths.RepoRoot, "namespace")
	if dirExists(repoNamespace) {
		return repoNamespace, nil
	}

	devNamespace := filepath.Join(paths.RepoRoot, ".kageos", "dev", "namespace")
	if currentWorkspaceMode(paths) == workspaceModeDev && dirExists(devNamespace) {
		return devNamespace, nil
	}

	if cfg, err := loadConfig(paths); err == nil && strings.TrimSpace(cfg.Storage.Root) != "" {
		return filepath.Join(cfg.Storage.Root, "namespace"), nil
	}

	return repoNamespace, nil
}

func planRuntimeCleanups(namespaceRoot string, keepLatest int) ([]cleanTarget, error) {
	if keepLatest <= 0 {
		keepLatest = defaultCleanKeepVersions
	}
	if !dirExists(namespaceRoot) {
		return nil, fmt.Errorf("namespace directory not found: %s", namespaceRoot)
	}

	var targets []cleanTarget
	err := filepath.WalkDir(namespaceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) != "version.json" || filepath.Base(filepath.Dir(path)) != "metadata" {
			return nil
		}

		appDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
		versionData, err := readCleanVersionData(path)
		if err != nil {
			return err
		}
		user, app := cleanAppIdentity(namespaceRoot, appDir, versionData)
		keepVersions := cleanVersionsToKeep(versionData, keepLatest)

		binaryTargets, err := planKnownReleaseFiles(
			filepath.Join(appDir, "workplace", "bin", "releases"),
			knownCleanBinaryVersions(user, app, versionData),
			keepVersions,
			"release binary",
			user,
			app,
		)
		if err != nil {
			return err
		}
		targets = append(targets, binaryTargets...)

		logTargets, err := planKnownReleaseFiles(
			filepath.Join(appDir, "workplace", "logs"),
			knownCleanLogVersions(user, app, versionData),
			keepVersions,
			"release log",
			user,
			app,
		)
		if err != nil {
			return err
		}
		targets = append(targets, logTargets...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortCleanTargets(targets)
	return targets, nil
}

func readCleanVersionData(path string) (*cleanVersionData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var versionData cleanVersionData
	if err := json.Unmarshal(data, &versionData); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &versionData, nil
}

func cleanAppIdentity(namespaceRoot, appDir string, versionData *cleanVersionData) (string, string) {
	user := strings.TrimSpace(versionData.User)
	app := strings.TrimSpace(versionData.App)
	if user != "" && app != "" {
		return user, app
	}

	rel, err := filepath.Rel(namespaceRoot, appDir)
	if err != nil {
		return user, app
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if user == "" && len(parts) >= 1 {
		user = parts[0]
	}
	if app == "" && len(parts) >= 2 {
		app = parts[1]
	}
	return user, app
}

func planKnownReleaseFiles(dir string, knownVersions map[string]string, keepVersions map[string]struct{}, kind, user, app string) ([]cleanTarget, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var targets []cleanTarget
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		version, ok := knownVersions[entry.Name()]
		if !ok && kind == "release log" {
			version, ok = matchRotatedCleanLog(entry.Name(), knownVersions)
		}
		if !ok {
			continue
		}
		if _, keep := keepVersions[version]; keep {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		targets = append(targets, cleanTarget{
			Kind:    kind,
			App:     user + "/" + app,
			Version: version,
			Path:    filepath.Join(dir, entry.Name()),
			Size:    info.Size(),
			Reason:  "old app runtime version",
		})
	}
	return targets, nil
}

func matchRotatedCleanLog(name string, knownVersions map[string]string) (string, bool) {
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

func knownCleanBinaryVersions(user, app string, versionData *cleanVersionData) map[string]string {
	known := make(map[string]string)
	for _, version := range allCleanVersions(versionData) {
		known[cleanBinaryName(user, app, version)] = version
	}
	return known
}

func knownCleanLogVersions(user, app string, versionData *cleanVersionData) map[string]string {
	known := make(map[string]string)
	for _, version := range allCleanVersions(versionData) {
		base := cleanLogName(user, app, version)
		known[base] = version
		rotatedPrefix := strings.TrimSuffix(base, ".log") + "-"
		known[rotatedPrefix] = version
	}
	return known
}

func allCleanVersions(versionData *cleanVersionData) []string {
	if versionData == nil {
		return nil
	}
	versions := make([]string, 0, len(versionData.Versions)+2)
	add := func(version string) {
		version = strings.TrimSpace(version)
		if version != "" {
			versions = append(versions, version)
		}
	}
	add(versionData.CurrentVersion)
	add(versionData.LatestVersion)
	for _, info := range versionData.Versions {
		add(info.Version)
	}
	return uniqueNonEmptyStrings(versions)
}

func cleanVersionsToKeep(versionData *cleanVersionData, keepLatest int) map[string]struct{} {
	keep := make(map[string]struct{})
	if versionData == nil {
		return keep
	}
	if keepLatest <= 0 {
		keepLatest = defaultCleanKeepVersions
	}

	add := func(version string) {
		version = strings.TrimSpace(version)
		if version != "" {
			keep[version] = struct{}{}
		}
	}
	add(versionData.CurrentVersion)
	add(versionData.LatestVersion)

	versions := make([]string, 0, len(versionData.Versions))
	for _, info := range versionData.Versions {
		version := strings.TrimSpace(info.Version)
		if version != "" {
			versions = append(versions, version)
		}
	}
	versions = uniqueNonEmptyStrings(versions)
	sort.Slice(versions, func(i, j int) bool {
		left := cleanVersionNumber(versions[i])
		right := cleanVersionNumber(versions[j])
		if left != right {
			return left > right
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

func cleanVersionNumber(version string) int {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	n, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return n
}

func cleanBinaryName(user, app, version string) string {
	return fmt.Sprintf("%s_%s_%s", user, app, version)
}

func cleanLogName(user, app, version string) string {
	return fmt.Sprintf("%s_%s_%s.log", user, app, version)
}

func planSourceLogCleanups(repoRoot, namespaceRoot string) ([]cleanTarget, error) {
	var targets []cleanTarget
	for _, root := range []string{
		filepath.Join(repoRoot, ".kageos"),
		filepath.Join(repoRoot, "core"),
		filepath.Join(repoRoot, "pkg"),
		filepath.Join(repoRoot, "sdk", "agent-app"),
		filepath.Join(repoRoot, "web", ".npm-cache", "_logs"),
	} {
		rootTargets, err := planSourceLogsUnderRoot(root, repoRoot, sourceLogInRepo)
		if err != nil {
			return nil, err
		}
		targets = append(targets, rootTargets...)
	}

	namespaceTargets, err := planSourceLogsUnderRoot(namespaceRoot, namespaceRoot, sourceLogInNamespace)
	if err != nil {
		return nil, err
	}
	targets = append(targets, namespaceTargets...)

	emptyDirs, err := planEmptyLogDirCleanups(repoRoot, namespaceRoot)
	if err != nil {
		return nil, err
	}
	targets = append(targets, emptyDirs...)

	sortCleanTargets(targets)
	return targets, nil
}

func planSourceLogsUnderRoot(root, relRoot string, match func(string) bool) ([]cleanTarget, error) {
	if !dirExists(root) {
		return nil, nil
	}

	var targets []cleanTarget
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(relRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !match(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		targets = append(targets, cleanTarget{
			Kind:   "source log",
			Path:   path,
			Size:   info.Size(),
			Reason: "historical default logger/package-manager log",
		})
		return nil
	})
	return targets, err
}

func planEmptyLogDirCleanups(repoRoot, namespaceRoot string) ([]cleanTarget, error) {
	var targets []cleanTarget
	for _, root := range []string{
		filepath.Join(repoRoot, ".kageos"),
		filepath.Join(repoRoot, "core"),
		filepath.Join(repoRoot, "pkg"),
		filepath.Join(repoRoot, "sdk", "agent-app"),
		namespaceRoot,
	} {
		rootTargets, err := planEmptyLogDirsUnderRoot(root)
		if err != nil {
			return nil, err
		}
		targets = append(targets, rootTargets...)
	}
	return targets, nil
}

func planEmptyLogDirsUnderRoot(root string) ([]cleanTarget, error) {
	if !dirExists(root) {
		return nil, nil
	}

	var targets []cleanTarget
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		switch entry.Name() {
		case ".git", "node_modules", "dist":
			return filepath.SkipDir
		case "logs":
			empty, err := isEmptyDir(path)
			if err != nil {
				return err
			}
			if empty {
				targets = append(targets, cleanTarget{
					Kind:   "empty log dir",
					Path:   path,
					Reason: "historical empty logger directory",
					IsDir:  true,
				})
				return filepath.SkipDir
			}
		}
		return nil
	})
	return targets, err
}

func isEmptyDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func sourceLogInRepo(rel string) bool {
	if strings.HasPrefix(rel, "web/.npm-cache/_logs/") && strings.HasSuffix(rel, ".log") {
		return true
	}
	return strings.HasSuffix(rel, "/logs/app.log") &&
		(strings.HasPrefix(rel, "core/") ||
			strings.HasPrefix(rel, "pkg/") ||
			strings.HasPrefix(rel, "sdk/agent-app/"))
}

func sourceLogInNamespace(rel string) bool {
	if strings.Contains(rel, "/workplace/") {
		return false
	}
	if strings.HasSuffix(rel, "/logs/app.log") && strings.Contains(rel, "/code/api/") {
		return true
	}
	parts := strings.Split(rel, "/")
	return len(parts) == 4 && parts[2] == "logs" && parts[3] == "app.log"
}

func applyCleanTargets(targets []cleanTarget, execute bool) error {
	totalSize := int64(0)
	fileCount := 0
	dirCount := 0
	for _, target := range targets {
		totalSize += target.Size
		if target.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}

	mode := "dry-run"
	if execute {
		mode = "execute"
	}
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Planned removals: files=%d dirs=%d size=%s\n", fileCount, dirCount, humanBytes(totalSize))
	if len(targets) == 0 {
		fmt.Println("Nothing to clean.")
		return nil
	}

	for _, target := range targets {
		prefix := "[dry-run]"
		if execute {
			prefix = "remove"
		}
		detail := target.Kind
		if target.App != "" || target.Version != "" {
			detail = strings.TrimSpace(fmt.Sprintf("%s %s %s", target.Kind, target.App, target.Version))
		}
		fmt.Printf("%s %s: %s (%s)\n", prefix, detail, target.Path, humanBytes(target.Size))
		if execute {
			if err := os.Remove(target.Path); err != nil && !(target.IsDir && os.IsNotExist(err)) {
				return err
			}
		}
	}
	if !execute {
		fmt.Println("Preview only. Re-run with --execute to delete these files.")
	}
	return nil
}

func sortCleanTargets(targets []cleanTarget) {
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].Path < targets[j].Path
	})
}

func humanBytes(bytes int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
