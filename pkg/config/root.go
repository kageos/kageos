package config

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// EnvAgentOSRoot 若设置且为有效目录，则作为项目根目录（配置位于 deploy/dev、deploy/prod 或兼容的 deploy/config 下）
	EnvAgentOSRoot = "AI_AGENT_OS_ROOT"
	// MarkerAgentOSRoot 仓库根目录标记文件（可选，便于无 go.mod 场景定位根）
	MarkerAgentOSRoot = ".ai-agent-os-root"
)

var (
	agentOSRoot     string
	agentOSRootOnce sync.Once
)

// GetAgentOSRoot 返回 AI Agent OS 项目根目录绝对路径，用于解析 deploy/dev、deploy/prod、deploy/config 等。
// 查找顺序：
//  1. 环境变量 AI_AGENT_OS_ROOT（非空且为目录则直接使用）
//  2. 从当前工作目录向上，找到第一个包含以下任一条件的目录：
//     - 存在 .ai-agent-os-root
//     - 存在 deploy/dev/config 或 deploy/prod/config
//     - 存在 deploy/config/prod 或 deploy/config/dev
//     - 存在 go.mod
//
// 若均未找到则返回空字符串，配置解析将退化为仅相对 cwd 查找。
func GetAgentOSRoot() string {
	agentOSRootOnce.Do(func() {
		if v := strings.TrimSpace(os.Getenv(EnvAgentOSRoot)); v != "" {
			if abs, err := filepath.Abs(v); err == nil {
				if st, err := os.Stat(abs); err == nil && st.IsDir() {
					agentOSRoot = abs
					return
				}
			}
		}
		cwd, err := os.Getwd()
		if err != nil {
			return
		}
		agentOSRoot = discoverAgentOSRootFrom(cwd)
	})
	return agentOSRoot
}

func discoverAgentOSRootFrom(start string) string {
	dir, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	for dir != "" {
		if isAgentOSRootDir(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func isAgentOSRootDir(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, MarkerAgentOSRoot)); err == nil {
		return true
	}
	if st, err := os.Stat(filepath.Join(dir, "deploy", "dev", "config")); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(dir, "deploy", "prod", "config")); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(dir, "deploy", "config", "prod")); err == nil && st.IsDir() {
		return true
	}
	if st, err := os.Stat(filepath.Join(dir, "deploy", "config", "dev")); err == nil && st.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	return false
}
