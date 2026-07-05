package llms

import (
	"fmt"
	"strings"
	"time"
)

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"

	ProtocolOpenAIChatCompletions = "openai_chat_completions"
	ProtocolOpenAIResponses       = "openai_responses"
	ProtocolAnthropicMessages     = "anthropic_messages"
)

const (
	AuthSchemeBearer  = "bearer"
	AuthSchemeXAPIKey = "x-api-key"
	AuthSchemeNone    = "none"
)

type ClientConfig struct {
	Provider     string
	Protocol     string
	APIKey       string
	BaseURL      string
	EndpointPath string
	APIVersion   string
	AuthScheme   string
	Headers      map[string]string
	Model        string
	Timeout      time.Duration
	MaxRetries   int
	UserAgent    string
}

func NewClientFromConfig(cfg ClientConfig) (LLMClient, error) {
	provider, protocol := NormalizeProviderProtocol(cfg.Provider, cfg.Protocol)
	provider, protocol = InferProviderProtocol(provider, protocol, cfg.BaseURL, cfg.EndpointPath)
	options := DefaultClientOptions().
		WithModel(cfg.Model).
		WithBaseURL(cfg.BaseURL).
		WithEndpointPath(cfg.EndpointPath).
		WithAPIVersion(cfg.APIVersion).
		WithAuthScheme(cfg.AuthScheme).
		WithHeaders(cfg.Headers).
		WithMaxRetries(cfg.MaxRetries)
	if cfg.Timeout > 0 {
		options.WithTimeout(cfg.Timeout)
	}
	if strings.TrimSpace(cfg.UserAgent) != "" {
		options.UserAgent = strings.TrimSpace(cfg.UserAgent)
	}

	switch protocol {
	case ProtocolOpenAIChatCompletions:
		return NewOpenAIClientWithOptions(cfg.APIKey, options), nil
	case ProtocolOpenAIResponses:
		return NewOpenAIResponsesClientWithOptions(cfg.APIKey, options), nil
	case ProtocolAnthropicMessages:
		return NewAnthropicMessagesClientWithOptions(cfg.APIKey, options), nil
	default:
		if provider == ProviderAnthropic {
			return nil, fmt.Errorf("unsupported Anthropic protocol: %s", protocol)
		}
		return nil, fmt.Errorf("unsupported LLM protocol: %s", protocol)
	}
}

func InferProviderProtocol(provider, protocol, baseURL, endpointPath string) (string, string) {
	provider, protocol = NormalizeProviderProtocol(provider, protocol)
	baseURL = strings.ToLower(strings.TrimSpace(baseURL))
	endpointPath = strings.ToLower(strings.TrimSpace(endpointPath))

	if provider == ProviderOpenAI && protocol == ProtocolOpenAIChatCompletions {
		if strings.Contains(endpointPath, "responses") || strings.HasSuffix(strings.TrimRight(baseURL, "/"), "/responses") {
			return provider, ProtocolOpenAIResponses
		}
	}
	if strings.Contains(baseURL, "anthropic") || strings.Contains(endpointPath, "messages") {
		return ProviderAnthropic, ProtocolAnthropicMessages
	}
	return provider, protocol
}

func NormalizeProviderProtocol(provider, protocol string) (string, string) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		switch provider {
		case ProviderAnthropic:
			protocol = ProtocolAnthropicMessages
		default:
			protocol = ProtocolOpenAIChatCompletions
		}
	}
	if provider == "" {
		switch protocol {
		case ProtocolAnthropicMessages:
			provider = ProviderAnthropic
		default:
			provider = ProviderOpenAI
		}
	}
	return provider, protocol
}

func defaultBaseURL(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case ProtocolAnthropicMessages:
		return "https://api.anthropic.com"
	default:
		return "https://api.openai.com/v1"
	}
}

func DefaultBaseURL(protocol string) string {
	return defaultBaseURL(protocol)
}

func defaultEndpointPath(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case ProtocolOpenAIResponses:
		return "/responses"
	case ProtocolAnthropicMessages:
		return "/v1/messages"
	default:
		return "/chat/completions"
	}
}

func DefaultEndpointPath(protocol string) string {
	return defaultEndpointPath(protocol)
}

func defaultAuthScheme(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case ProtocolAnthropicMessages:
		return AuthSchemeXAPIKey
	default:
		return AuthSchemeBearer
	}
}

func DefaultAuthScheme(protocol string) string {
	return defaultAuthScheme(protocol)
}

func defaultAPIVersion(protocol string) string {
	if strings.ToLower(strings.TrimSpace(protocol)) == ProtocolAnthropicMessages {
		return "2023-06-01"
	}
	return ""
}

func DefaultAPIVersion(protocol string) string {
	return defaultAPIVersion(protocol)
}

func cleanHeaderMap(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
