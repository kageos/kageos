package dto

const (
	WorkspaceToolCallStatusOK        = "ok"
	WorkspaceToolCallStatusError     = "error"
	WorkspaceToolCallStatusRunning   = "running"
	WorkspaceToolCallStatusStreaming = "streaming"
)

const (
	WorkspaceStreamEventSession              = "session"
	WorkspaceStreamEventModelContextPlan     = "model_context_plan"
	WorkspaceStreamEventToolCall             = "tool_call"
	WorkspaceStreamEventToolCallsStreamDelta = "tool_calls_stream_delta"
	WorkspaceStreamEventContent              = "content"
	WorkspaceStreamEventDone                 = "done"
	WorkspaceStreamEventError                = "error"
)

// WorkspaceStreamEvent is the typed envelope sent over workspace SSE.
type WorkspaceStreamEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type WorkspaceStreamSession struct {
	SessionID string `json:"session_id"`
}

type WorkspaceStreamToolCall struct {
	ID         string              `json:"id,omitempty"`
	Index      int                 `json:"index"`
	Round      int                 `json:"round"`
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	Arguments  string              `json:"arguments"`
	Result     string              `json:"result"`
	ResultData interface{}         `json:"result_data,omitempty"`
	Metadata   *ToolResultMetadata `json:"metadata,omitempty"`
	Error      string              `json:"error"`
}

type WorkspaceStreamToolCallDeltaUpdate struct {
	Index int    `json:"index"`
	Round int    `json:"round"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Delta string `json:"delta"`
}

type WorkspaceStreamToolCallDeltaData struct {
	Updates []WorkspaceStreamToolCallDeltaUpdate `json:"updates"`
}

type WorkspaceStreamContent struct {
	Content string `json:"content"`
}

type WorkspaceModelContextPlan struct {
	ProtocolVersion string                         `json:"protocol_version"`
	SessionID       string                         `json:"session_id"`
	Round           int                            `json:"round"`
	ModeCode        string                         `json:"mode_code,omitempty"`
	Role            WorkspaceModelContextRole      `json:"role"`
	Execution       WorkspaceModelContextExecution `json:"execution"`
	Messages        WorkspaceModelContextMessages  `json:"messages"`
	Handoff         *WorkspaceModelContextHandoff  `json:"handoff,omitempty"`
	Docs            WorkspaceModelContextDocs      `json:"docs"`
	Tools           WorkspaceModelContextTools     `json:"tools"`
	CachePlan       WorkspaceModelContextCachePlan `json:"cache_plan"`
	LLM             *WorkspaceModelContextLLM      `json:"llm,omitempty"`
}

type WorkspaceModelContextRole struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"display_name,omitempty"`
	Source             string   `json:"source,omitempty"`
	Responsibility     string   `json:"responsibility,omitempty"`
	HandoffRequired    []string `json:"handoff_required,omitempty"`
	AllowedTools       []string `json:"allowed_tools,omitempty"`
	ForbiddenTools     []string `json:"forbidden_tools,omitempty"`
	AllowedTransitions []string `json:"allowed_transitions,omitempty"`
}

type WorkspaceModelContextExecution struct {
	FullCodePath  string `json:"full_code_path"`
	DirectoryName string `json:"directory_name,omitempty"`
	DirectoryCode string `json:"directory_code,omitempty"`
	DirectoryType string `json:"directory_type,omitempty"`
	ChildrenCount int    `json:"children_count"`
	FilesCount    int    `json:"files_count"`
	ScopePolicy   string `json:"scope_policy"`
}

type WorkspaceModelContextMessages struct {
	ContextPolicy               string                            `json:"context_policy"`
	ModelContextAnchorMessageID int64                             `json:"model_context_anchor_message_id,omitempty"`
	ParentSessionID             string                            `json:"parent_session_id,omitempty"`
	SourceHistoryPolicy         string                            `json:"source_history_policy"`
	SystemMessages              int                               `json:"system_messages"`
	LLMMessages                 int                               `json:"llm_messages"`
	TotalStoredMessages         int                               `json:"total_stored_messages"`
	IncludedStoredMessages      int                               `json:"included_stored_messages"`
	ExcludedStoredMessages      int                               `json:"excluded_stored_messages"`
	ExcludedByAnchor            int                               `json:"excluded_by_anchor"`
	ExcludedDisplayOnly         int                               `json:"excluded_display_only"`
	Included                    []WorkspaceModelContextMessageRef `json:"included,omitempty"`
	Excluded                    []WorkspaceModelContextMessageRef `json:"excluded,omitempty"`
	Truncated                   bool                              `json:"truncated,omitempty"`
}

type WorkspaceModelContextMessageRef struct {
	ID           int64  `json:"id"`
	Role         string `json:"role"`
	ContextUsage string `json:"context_usage,omitempty"`
	ArtifactKind string `json:"artifact_kind,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

type WorkspaceModelContextHandoff struct {
	PacketVersion      string   `json:"packet_version,omitempty"`
	SourceSessionID    string   `json:"source_session_id,omitempty"`
	SourceRole         string   `json:"source_role,omitempty"`
	TargetRole         string   `json:"target_role,omitempty"`
	ArtifactKind       string   `json:"artifact_kind,omitempty"`
	ExecuteDirectory   string   `json:"execute_directory,omitempty"`
	WorkspaceDirectory string   `json:"workspace_directory,omitempty"`
	TargetAppDirectory string   `json:"target_app_directory,omitempty"`
	TaskContext        []string `json:"task_context,omitempty"`
	KeyInformation     []string `json:"key_information,omitempty"`
	References         []string `json:"references,omitempty"`
	ValidationStatus   string   `json:"validation_status,omitempty"`
}

type WorkspaceModelContextDocs struct {
	DocumentPackage []string `json:"document_package,omitempty"`
	RequiredDocs    []string `json:"required_docs,omitempty"`
	OptionalDocs    []string `json:"optional_docs,omitempty"`
	LoadedDocs      []string `json:"loaded_docs,omitempty"`
	MissingDocs     []string `json:"missing_docs,omitempty"`
}

type WorkspaceModelContextTools struct {
	RequestedNames []string `json:"requested_names,omitempty"`
	LLMTools       []string `json:"llm_tools,omitempty"`
	LLMToolCount   int      `json:"llm_tool_count"`
	Policy         string   `json:"policy"`
}

type WorkspaceModelContextCachePlan struct {
	StablePrefixStrategy string                            `json:"stable_prefix_strategy"`
	StablePrefixItems    []string                          `json:"stable_prefix_items,omitempty"`
	ActualUsageField     string                            `json:"actual_usage_field"`
	Result               *WorkspaceModelContextCacheResult `json:"result,omitempty"`
}

type WorkspaceModelContextCacheResult struct {
	Status               string `json:"status"`
	PromptTokens         int    `json:"prompt_tokens"`
	CompletionTokens     int    `json:"completion_tokens"`
	TotalTokens          int    `json:"total_tokens"`
	CachedTokens         int    `json:"cached_tokens"`
	CacheHitRatePercent  int    `json:"cache_hit_rate_percent"`
	CachedTokensReported bool   `json:"cached_tokens_reported"`
	Source               string `json:"source"`
}

type WorkspaceModelContextLLM struct {
	ConfigID     int64  `json:"config_id,omitempty"`
	ConfigName   string `json:"config_name,omitempty"`
	Provider     string `json:"provider,omitempty"`
	Model        string `json:"model,omitempty"`
	RequestModel string `json:"request_model,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	MessageCount int    `json:"message_count"`
	ToolCount    int    `json:"tool_count"`
}

type WorkspaceStreamDone struct {
	SessionID        string                         `json:"session_id"`
	ToolCalls        []WorkspaceChatToolCallSummary `json:"tool_calls"`
	LLMConfigID      int64                          `json:"llm_config_id,omitempty"`
	LLMConfigName    string                         `json:"llm_config_name,omitempty"`
	LLMProvider      string                         `json:"llm_provider,omitempty"`
	LLMModel         string                         `json:"llm_model,omitempty"`
	LLMUsage         *LLMUsageInfo                  `json:"llm_usage,omitempty"`
	ModelContextPlan *WorkspaceModelContextPlan     `json:"model_context_plan,omitempty"`
}

type WorkspaceStreamError struct {
	Message string `json:"message"`
}
