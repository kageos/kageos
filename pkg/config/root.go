package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

const (
	// MarkerKageOSRoot 仓库根目录标记文件（可选，便于无 go.mod 场景定位根）
	MarkerKageOSRoot     = ".kageos-root"
	maxMarkerSearchDepth = 8
	maxMarkerSearchDirs  = 4096
)

var (
	kageOSRoot     string
	kageOSRootOnce sync.Once
)

// GetKageOSRoot 返回 Kageos 项目根目录绝对路径，用于解析 deploy/dev、deploy/prod 等。
// 查找顺序：
//  1. 从代码所在目录、当前工作目录开始，优先向上查找 `.kageos-root`
//  2. 若仍未找到，则从这些起点的祖先目录向下搜索 `.kageos-root`
//  3. 最后再退化为 deploy/dev/config、deploy/prod/config、go.mod 等弱特征目录
//
// 若均未找到则返回空字符串，配置解析将退化为仅相对 cwd 查找。
func GetKageOSRoot() string {
	kageOSRootOnce.Do(func() {
		if _, file, _, ok := runtime.Caller(0); ok {
			if root := discoverKageOSRootFrom(filepath.Dir(file)); root != "" {
				kageOSRoot = root
				return
			}
		}
		cwd, err := os.Getwd()
		if err != nil {
			return
		}
		kageOSRoot = discoverKageOSRootFrom(cwd)
	})
	return kageOSRoot
}

func discoverKageOSRootFrom(start string) string {
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
		if isWeakKageOSRootDir(ancestor) {
			return ancestor
		}
	}
	return ""
}

func discoverMarkerRootUpward(start string) string {
	for _, dir := range ancestorDirs(start) {
		if hasKageOSRootMarker(dir) {
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

		if hasKageOSRootMarker(current.path) {
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

func hasKageOSRootMarker(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, MarkerKageOSRoot)); err == nil {
		return true
	}
	return false
}

func isWeakKageOSRootDir(dir string) bool {
	if st, err := os.Stat(filepath.Join(dir, "deploy", "dev", "config")); err == nil && st.IsDir() {
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
