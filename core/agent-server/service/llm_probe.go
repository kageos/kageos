package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
)

const defaultLLMProbeTimeout = 30

type llmProbeCandidate struct {
	Provider string
	Protocol string
}

// ProbeLLMConfig tries the selected protocol, or a small ordered set of common protocols,
// without saving the submitted credentials.
func (s *LLMService) ProbeLLMConfig(ctx context.Context, req dto.LLMProbeReq) (*dto.LLMProbeResp, error) {
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" && req.ID > 0 {
		cfg, err := s.getManageableLLMConfig(ctx, req.ID, "检测")
		if err != nil {
			return nil, err
		}
		apiKey, err = s.OpenAPIKey(cfg.APIKey)
		if err != nil {
			return nil, fmt.Errorf("解密 LLM API Key 失败: %w", err)
		}
	}
	if apiKey != "" {
		opened, err := s.OpenAPIKey(apiKey)
		if err != nil {
			return nil, fmt.Errorf("解析 LLM API Key 失败: %w", err)
		}
		apiKey = opened
	}

	headersRaw := strings.TrimSpace(req.Headers)
	headers, err := llmHeadersFromJSON(&headersRaw)
	if err != nil {
		return nil, err
	}
	extraConfig, err := llmProbeExtraConfig(req.ExtraConfig)
	if err != nil {
		return nil, err
	}

	timeoutSeconds := req.Timeout
	if timeoutSeconds <= 0 {
		timeoutSeconds = defaultLLMProbeTimeout
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 64
	}

	candidates, err := llmProbeCandidates(req)
	if err != nil {
		return nil, err
	}
	attempts := make([]dto.LLMProbeAttempt, 0, len(candidates))
	for _, candidate := range candidates {
		provider, protocol, err := normalizeLLMProviderProtocol(candidate.Provider, candidate.Protocol)
		attempt := dto.LLMProbeAttempt{
			Provider: provider,
			Protocol: protocol,
			APIBase:  strings.TrimSpace(req.APIBase),
		}
		if err != nil {
			attempt.Error = err.Error()
			attempts = append(attempts, attempt)
			continue
		}

		baseURL := strings.TrimSpace(req.APIBase)
		endpointPath := strings.TrimSpace(req.EndpointPath)
		provider, protocol = llms.InferProviderProtocol(provider, protocol, baseURL, endpointPath)
		attempt.Provider = provider
		attempt.Protocol = protocol
		if baseURL == "" {
			baseURL = llms.DefaultBaseURL(protocol)
		}
		if endpointPath == "" {
			endpointPath = llms.DefaultEndpointPath(protocol)
		}
		provider, protocol = llms.InferProviderProtocol(provider, protocol, baseURL, endpointPath)
		attempt.Provider = provider
		attempt.Protocol = protocol
		apiVersion := strings.TrimSpace(req.APIVersion)
		if apiVersion == "" {
			apiVersion = llms.DefaultAPIVersion(protocol)
		}
		authScheme := strings.TrimSpace(req.AuthScheme)
		if authScheme == "" {
			authScheme = llms.DefaultAuthScheme(protocol)
		}
		modelName := strings.TrimSpace(req.Model)
		if modelName == "" {
			modelName = llmProbeDefaultModel(protocol)
		}
		attempt.APIBase = baseURL

		clientConfig := llms.ClientConfig{
			Provider:     provider,
			Protocol:     protocol,
			APIKey:       apiKey,
			BaseURL:      baseURL,
			EndpointPath: endpointPath,
			APIVersion:   apiVersion,
			AuthScheme:   authScheme,
			Headers:      headers,
			Model:        modelName,
			Timeout:      time.Duration(timeoutSeconds) * time.Second,
		}
		if maxRetries, ok := extraConfig["max_retries"].(float64); ok && maxRetries >= 0 {
			clientConfig.MaxRetries = int(maxRetries)
		}
		if userAgent, ok := extraConfig["user_agent"].(string); ok {
			clientConfig.UserAgent = userAgent
		}
		client, err := llms.NewClientFromConfig(clientConfig)
		if err != nil {
			attempt.Error = err.Error()
			attempts = append(attempts, attempt)
			continue
		}

		chatTimeout := time.Duration(timeoutSeconds) * time.Second
		callCtx, cancel := context.WithTimeout(ctx, chatTimeout)
		probeReq := &llms.ChatRequest{
			Messages: []llms.Message{
				{Role: "user", Content: "Reply with exactly: KAGEOS_OK"},
			},
			Model:     modelName,
			MaxTokens: maxTokens,
			Timeout:   &chatTimeout,
		}
		if reasoningEffort, ok := extraConfig["reasoning_effort"].(string); ok {
			probeReq.ReasoningEffort = strings.TrimSpace(reasoningEffort)
		}
		if verbosity, ok := extraConfig["verbosity"].(string); ok {
			probeReq.Verbosity = strings.TrimSpace(verbosity)
		}
		resp, err := client.Chat(callCtx, probeReq)
		cancel()
		if err != nil {
			attempt.Error = err.Error()
			attempts = append(attempts, attempt)
			continue
		}
		if resp != nil && strings.TrimSpace(resp.Error) != "" {
			attempt.Error = resp.Error
			attempts = append(attempts, attempt)
			continue
		}
		if resp == nil || strings.TrimSpace(resp.Content) == "" {
			attempt.Error = "上游响应为空"
			attempts = append(attempts, attempt)
			continue
		}

		attempt.OK = true
		attempts = append(attempts, attempt)
		contextWindow, contextWindowSource := probeLLMContextWindow(ctx, clientConfig)
		return &dto.LLMProbeResp{
			OK:                  true,
			Provider:            provider,
			Protocol:            protocol,
			APIBase:             baseURL,
			EndpointPath:        endpointPath,
			APIVersion:          apiVersion,
			AuthScheme:          authScheme,
			Model:               client.GetModelName(),
			Message:             strings.TrimSpace(resp.Content),
			Capabilities:        llmProbeCapabilities(protocol),
			ContextWindow:       contextWindow,
			ContextWindowSource: contextWindowSource,
			Attempts:            attempts,
		}, nil
	}

	message := "全部候选 LLM 协议检测失败"
	if len(attempts) == 1 && attempts[0].Error != "" {
		message = attempts[0].Error
	}
	return &dto.LLMProbeResp{
		OK:       false,
		Error:    message,
		Attempts: attempts,
	}, nil
}

func llmProbeCandidates(req dto.LLMProbeReq) ([]llmProbeCandidate, error) {
	var candidates []llmProbeCandidate
	add := func(provider, protocol string) {
		for _, candidate := range candidates {
			if candidate.Provider == provider && candidate.Protocol == protocol {
				return
			}
		}
		candidates = append(candidates, llmProbeCandidate{Provider: provider, Protocol: protocol})
	}

	provider := strings.TrimSpace(req.Provider)
	protocol := strings.TrimSpace(req.Protocol)
	key := strings.ToLower(strings.TrimSpace(req.APIKey))
	base := strings.ToLower(strings.TrimSpace(req.APIBase))
	endpoint := strings.ToLower(strings.TrimSpace(req.EndpointPath))
	modelName := strings.ToLower(strings.TrimSpace(req.Model))

	if provider != "" || protocol != "" {
		normalizedProvider, normalizedProtocol, err := normalizeLLMProviderProtocol(provider, protocol)
		if err != nil {
			return nil, err
		}
		normalizedProvider, normalizedProtocol = llms.InferProviderProtocol(normalizedProvider, normalizedProtocol, req.APIBase, req.EndpointPath)
		defaultOpenAIChat := normalizedProvider == llms.ProviderOpenAI &&
			normalizedProtocol == llms.ProtocolOpenAIChatCompletions &&
			endpoint == "" &&
			(base == "" || strings.Contains(base, "api.openai.com"))
		if !defaultOpenAIChat {
			add(normalizedProvider, normalizedProtocol)
		}
	}

	if strings.HasPrefix(key, "sk-ant-") || strings.Contains(base, "anthropic") || strings.Contains(modelName, "claude") {
		add(llms.ProviderAnthropic, llms.ProtocolAnthropicMessages)
	}
	if strings.Contains(endpoint, "responses") || base == "" || strings.Contains(base, "api.openai.com") {
		add(llms.ProviderOpenAI, llms.ProtocolOpenAIResponses)
		add(llms.ProviderOpenAI, llms.ProtocolOpenAIChatCompletions)
	} else {
		add(llms.ProviderOpenAI, llms.ProtocolOpenAIChatCompletions)
		add(llms.ProviderOpenAI, llms.ProtocolOpenAIResponses)
	}
	add(llms.ProviderAnthropic, llms.ProtocolAnthropicMessages)
	return candidates, nil
}

func llmProbeDefaultModel(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case llms.ProtocolAnthropicMessages:
		return "claude-sonnet-4-5"
	default:
		return "gpt-4o-mini"
	}
}

func llmProbeCapabilities(protocol string) map[string]bool {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case llms.ProtocolOpenAIResponses:
		return map[string]bool{"stream": true, "tools": true, "reasoning": true}
	case llms.ProtocolAnthropicMessages:
		return map[string]bool{"stream": true, "tools": true, "thinking": true}
	default:
		return map[string]bool{"stream": true, "tools": true}
	}
}

func llmProbeExtraConfig(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("extra_config 不是有效 JSON: %w", err)
	}
	return out, nil
}
