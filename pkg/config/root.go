package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

const (
	// MarkerKageosRoot 仓库根目录标记文件（可选，便于无 go.mod 场景定位根）
	MarkerKageosRoot     = ".kageos-root"
	maxMarkerSearchDepth = 8
	maxMarkerSearchDirs  = 4096
)

var (
	kageosRoot     string
	kageosRootOnce sync.Once
)

// GetKageosRoot 返回 Kageos 项目根目录绝对路径，用于解析 .kageos、deploy/dev、deploy/prod 等。
// 查找顺序：
//  1. 从代码所在目录、当前工作目录开始，优先向上查找 `.kageos-root`
//  2. 若仍未找到，则从这些起点的祖先目录向下搜索 `.kageos-root`
//  3. 最后再退化为 .kageos、deploy/prod/config、go.mod 等弱特征目录
//
// 若均未找到则返回空字符串，配置解析将退化为仅相对 cwd 查找。
func GetKageosRoot() string {
	kageosRootOnce.Do(func() {
		if explicitRoot := strings.TrimSpace(os.Getenv("KAGEOS_ROOT")); explicitRoot != "" {
			if abs, err := filepath.Abs(explicitRoot); err == nil {
				kageosRoot = filepath.Clean(abs)
			} else {
				kageosRoot = filepath.Clean(explicitRoot)
			}
			return
		}
		if _, file, _, ok := runtime.Caller(0); ok {
			if root := discoverKageosRootFrom(filepath.Dir(file)); root != "" {
				kageosRoot = root
				return
			}
		}
		cwd, err := os.Getwd()
		if err != nil {
			return
		}
		kageosRoot = discoverKageosRootFrom(cwd)
	})
	return kageosRoot
}

func discoverKageosRootFrom(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}

	if root := discoverMarkerRootUpward(dir); root != "" {
		return root
	}
	for _, ancestor := range ancestorDirs(dir) {
		if root := discoverMarkerRootDownward(ancestor, maxMarkerSearchDepth, maxMarkerSearchDirs); root != "" {
			return root
		}
	}
	for _, ancestor := range ancestorDirs(dir) {
		if isWeakKageosRootDir(ancestor) {
			return ancestor
		}
	}
	return ""
}

func discoverMarkerRootUpward(start string) string {
	for _, dir := range ancestorDirs(start) {
		if hasKageosRootMarker(dir) {
			return dir
		}
	}
	return ""
}

func discoverMarkerRootDownward(start string, maxDepth int, maxDirs int) string {
	type node struct {
		path  string
		depth int
	}

	queue := []node{{path: filepath.Clean(start), depth: 0}}
	visited := 0

	for len(queue) > 0 && visited < maxDirs {
		current := queue[0]
		queue = queue[1:]
		visited++

		if hasKageosRootMarker(current.path) {
			return current.path
		}
		if current.depth >= maxDepth {
			continue
		}

		entries, err := os.ReadDir(current.path)
		if err != nil {
			continue
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			queue = append(queue, node{
				path:  filepath.Join(current.path, entry.Name()),
				depth: current.depth + 1,
			})
		}
	}
	return ""
}

func ancestorDirs(start string) []string {
	dirs := make([]string, 0, 16)
	for dir := filepath.Clean(start); dir != ""; {
		dirs = append(dirs, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}

func hasKageosRootMarker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, MarkerKageosRoot)); err == nil {
		return true
	}
	return false
}

func isWeakKageosRootDir(dir string) bool {
	if st, err := os.Stat(filepath.Join(dir, ".kageos")); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(dir, "deploy", "prod", "config")); err == nil && st.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	return false
}
