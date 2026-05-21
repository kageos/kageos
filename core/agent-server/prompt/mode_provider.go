package prompt

import (
	"encoding/json"
	"io/fs"
	"strings"
	"sync"

	workspaceroles "github.com/kageos/kageos/core/agent-server/workspace/roles"
)

// ModeConfig 模式目录下的 config.json 结构
type ModeConfig struct {
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	ToolNames               []string `json:"tool_names"`
	SystemPromptFile        string   `json:"system_prompt_file"`
	SystemPromptAppendFiles []string `json:"system_prompt_append_files"`
}

// WorkspaceModePromptProvider 工作台「模式提示词提供者」多态接口；每种模式一个实现，参数封装在内部
type WorkspaceModePromptProvider interface {
	Code() string
	SystemPrompt(env *WorkspaceEnvData) string
	ToolNames() []string
}

// modeProvider 从本地 seed 的 prompt/system/prompt/mode/<code>/ 加载，内部持有所需内容。
type modeProvider struct {
	code         string
	systemPrompt string
	toolNames    []string
}

func (p *modeProvider) Code() string { return p.code }

func (p *modeProvider) SystemPrompt(_ *WorkspaceEnvData) string {
	return strings.TrimSpace(p.systemPrompt)
}

func (p *modeProvider) ToolNames() []string {
	return p.toolNames
}

func loadSeedModeConfig(code string) *ModeConfig {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	cfgPath := "system/prompt/mode/" + code + "/config.json"
	data, err := fs.ReadFile(promptFS, cfgPath)
	if err != nil {
		return nil
	}
	var cfg ModeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

var (
	registry   = make(map[string]WorkspaceModePromptProvider)
	registryMu sync.RWMutex
)

func init() {
	for _, code := range []string{"dev"} {
		if p := loadModeProvider(code); p != nil {
			registry[code] = p
		}
	}
}

// GetModeProvider 根据模式 code 返回提供者；未注册返回 nil（调用方退化为 DB 的 fragment + tool_names）
func GetModeProvider(code string) WorkspaceModePromptProvider {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[code]
}

// loadModeProvider 从 embed 的 system/prompt/mode/<code>/ 加载 config + md，封装为 modeProvider。
func loadModeProvider(code string) *modeProvider {
	cfg := loadSeedModeConfig(code)
	if cfg == nil {
		return nil
	}
	prefix := "system/prompt/mode/" + code + "/"
	systemPrompt, _ := readModeFile(prefix + cfg.SystemPromptFile)
	systemPrompt = appendModeSystemPrompt(systemPrompt, loadSeedPromptAppendFiles(code, modeSystemPromptAppendFiles(code, cfg.SystemPromptAppendFiles)))
	systemPrompt = finalizeModeSystemPrompt(systemPrompt)
	toolNames := cfg.ToolNames
	if toolNames == nil {
		toolNames = []string{}
	}
	return &modeProvider{
		code:         code,
		systemPrompt: systemPrompt,
		toolNames:    toolNames,
	}
}

func finalizeModeSystemPrompt(systemPrompt string) string {
	return workspaceroles.ApplyRoutingMarkdown(systemPrompt)
}

func readModeFile(path string) (string, error) {
	data, err := fs.ReadFile(promptFS, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func modeSystemPromptAppendFiles(code string, configured []string) []string {
	return append([]string{}, configured...)
}

func loadSeedPromptAppendFiles(code string, fileNames []string) []string {
	contents := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		content := readSeedPromptAppendFile(code, fileName)
		if strings.TrimSpace(content) == "" {
			continue
		}
		contents = append(contents, content)
	}
	return contents
}

func readSeedPromptAppendFile(code, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}
	if strings.HasPrefix(fileName, SystemPromptRootPath+"/") || fileName == SystemPromptRootPath {
		_, content := getSeedPromptDocContent(fileName)
		return content
	}
	if strings.HasPrefix(fileName, systemPromptSeedRoot+"/") {
		content, _ := readModeFile(fileName)
		return content
	}
	prefix := "system/prompt/mode/" + strings.TrimSpace(code) + "/"
	content, _ := readModeFile(prefix + fileName)
	return content
}

func appendModeSystemPrompt(base string, appendContents []string) string {
	parts := make([]string, 0, 1+len(appendContents))
	if strings.TrimSpace(base) != "" {
		parts = append(parts, strings.TrimSpace(base))
	}
	for _, content := range appendContents {
		if strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, strings.TrimSpace(content))
	}
	return strings.Join(parts, "\n\n")
}
