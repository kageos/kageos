package prompt

import (
	"encoding/json"
	"io/fs"
	"strings"
	"sync"
)

// ModeConfig 模式目录下的 config.json 结构
type ModeConfig struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	ToolNames           []string `json:"tool_names"`
	SystemPromptFile    string   `json:"system_prompt_file"`
	FirstAssistantFile  string   `json:"first_assistant_file"`
	OperationPromptFile string   `json:"operation_prompt_file"` // 可选，本模式额外操作提示词；空则不追加额外文档
}

// WorkspaceModePromptProvider 工作台「模式提示词提供者」多态接口；每种模式一个实现，参数封装在内部
type WorkspaceModePromptProvider interface {
	Code() string
	SystemPrompt(env *WorkspaceEnvData) string
	FirstAssistantContent() string
	ToolNames() []string
	// OperationPrompt 本模式专属操作提示词（PRD/SOP/工具用法等）；空则调用方不追加额外提示词
	OperationPrompt() string
}

// modeProvider 从本地 seed 的 prompt/system/prompt/mode/<code>/ 加载，内部持有所需内容。
type modeProvider struct {
	code            string
	systemPrompt    string
	firstAssistant  string
	operationPrompt string
	toolNames       []string
}

func (p *modeProvider) Code() string { return p.code }

func (p *modeProvider) SystemPrompt(_ *WorkspaceEnvData) string {
	return strings.TrimSpace(p.systemPrompt)
}

func (p *modeProvider) FirstAssistantContent() string {
	return strings.TrimSpace(p.firstAssistant)
}

func (p *modeProvider) ToolNames() []string {
	return p.toolNames
}

func (p *modeProvider) OperationPrompt() string {
	return strings.TrimSpace(p.operationPrompt)
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
	for _, code := range []string{"dev", "modify", "execute", "agent"} {
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
	firstAssistant, _ := readModeFile(prefix + cfg.FirstAssistantFile)
	var operationPrompt string
	if cfg.OperationPromptFile != "" {
		operationPrompt, _ = readModeFile(prefix + cfg.OperationPromptFile)
	}
	toolNames := cfg.ToolNames
	if toolNames == nil {
		toolNames = []string{}
	}
	return &modeProvider{
		code:            code,
		systemPrompt:    systemPrompt,
		firstAssistant:  firstAssistant,
		operationPrompt: operationPrompt,
		toolNames:       toolNames,
	}
}

func readModeFile(path string) (string, error) {
	data, err := fs.ReadFile(promptFS, path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
