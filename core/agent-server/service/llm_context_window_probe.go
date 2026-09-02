package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/llms"
)

const maxLLMMetadataBodyBytes = 2 << 20

var contextWindowMetadataKeys = map[string]struct{}{
	"contextlength":        {},
	"contextwindow":        {},
	"maxcontextlength":     {},
	"maxcontexttokens":     {},
	"inputtokenlimit":      {},
	"maxinputtokens":       {},
	"maximumcontextlength": {},
}

var maxOutputTokenMetadataKeys = map[string]struct{}{
	"maxoutputtokens":     {},
	"outputtokenlimit":    {},
	"maximumoutputtokens": {},
	"maxtokens":           {},
}

// probeLLMContextWindow performs a best-effort metadata lookup. Failure is
// deliberately non-fatal because many OpenAI-compatible providers do not
// expose capacity through /models even though inference works normally.
func probeLLMContextWindow(ctx context.Context, clientConfig llms.ClientConfig) (int, string) {
	base := llmMetadataBaseURL(clientConfig.BaseURL, clientConfig.EndpointPath, clientConfig.Protocol)
	if base == "" || strings.TrimSpace(clientConfig.Model) == "" {
		return DefaultLLMContextWindow, "default"
	}
	headers := llmMetadataHeaders(clientConfig)
	httpClient := &http.Client{Timeout: clientConfig.Timeout}
	if httpClient.Timeout <= 0 || httpClient.Timeout > 10*time.Second {
		httpClient.Timeout = 10 * time.Second
	}
	modelName := strings.TrimSpace(clientConfig.Model)
	URLs := []string{
		strings.TrimRight(base, "/") + "/models/" + url.PathEscape(modelName),
		strings.TrimRight(base, "/") + "/models",
	}
	for _, endpoint := range URLs {
		value, err := getLLMContextWindowMetadata(ctx, httpClient, endpoint, headers, modelName)
		if err == nil && value > 0 {
			return value, "provider_metadata"
		}
	}
	if value := knownModelSpecForName(modelName).ContextWindow; value > 0 {
		return value, "model_registry"
	}
	return DefaultLLMContextWindow, "default"
}

// probeLLMMaxOutputTokens reads provider model metadata when available and
// falls back to the bundled model registry. Unknown models keep a conservative
// automatic default instead of pretending that an exact provider limit was detected.
func probeLLMMaxOutputTokens(ctx context.Context, clientConfig llms.ClientConfig) (int, string) {
	base := llmMetadataBaseURL(clientConfig.BaseURL, clientConfig.EndpointPath, clientConfig.Protocol)
	modelName := strings.TrimSpace(clientConfig.Model)
	if base != "" && modelName != "" {
		headers := llmMetadataHeaders(clientConfig)
		httpClient := &http.Client{Timeout: clientConfig.Timeout}
		if httpClient.Timeout <= 0 || httpClient.Timeout > 10*time.Second {
			httpClient.Timeout = 10 * time.Second
		}
		URLs := []string{
			strings.TrimRight(base, "/") + "/models/" + url.PathEscape(modelName),
			strings.TrimRight(base, "/") + "/models",
		}
		for _, endpoint := range URLs {
			value, err := getLLMMaxOutputTokensMetadata(ctx, httpClient, endpoint, headers, modelName)
			if err == nil && value > 0 {
				return value, "provider_metadata"
			}
		}
	}
	if value := knownModelSpecForName(modelName).MaxOutputTokens; value > 0 {
		return value, "model_registry"
	}
	return DefaultLLMMaxOutputTokens, "default"
}

func llmMetadataBaseURL(baseURL, endpointPath, protocol string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(llms.DefaultBaseURL(protocol), "/")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	for _, suffix := range []string{"/chat/completions", "/responses", "/messages"} {
		if strings.HasSuffix(strings.ToLower(parsed.Path), suffix) {
			parsed.Path = strings.TrimSuffix(parsed.Path, parsed.Path[len(parsed.Path)-len(suffix):])
			return strings.TrimRight(parsed.String(), "/")
		}
	}
	endpointDir := path.Dir("/" + strings.TrimLeft(strings.TrimSpace(endpointPath), "/"))
	endpointParts := strings.Split(strings.Trim(endpointDir, "/"), "/")
	versionPrefix := ""
	if len(endpointParts) > 0 && len(endpointParts[0]) > 1 && endpointParts[0][0] == 'v' {
		if _, err := strconv.Atoi(endpointParts[0][1:]); err == nil {
			versionPrefix = "/" + endpointParts[0]
		}
	}
	if versionPrefix != "" && !strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), versionPrefix) {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + versionPrefix
	}
	return strings.TrimRight(parsed.String(), "/")
}

func llmMetadataHeaders(cfg llms.ClientConfig) map[string]string {
	headers := make(map[string]string, len(cfg.Headers)+3)
	for key, value := range cfg.Headers {
		if strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			headers[key] = value
		}
	}
	scheme := strings.ToLower(strings.TrimSpace(cfg.AuthScheme))
	if scheme == "" {
		scheme = llms.DefaultAuthScheme(cfg.Protocol)
	}
	if strings.TrimSpace(cfg.APIKey) != "" {
		if scheme == llms.AuthSchemeXAPIKey {
			headers["x-api-key"] = cfg.APIKey
		} else if scheme != llms.AuthSchemeNone {
			headers["Authorization"] = "Bearer " + cfg.APIKey
		}
	}
	if strings.TrimSpace(cfg.APIVersion) != "" && cfg.Protocol == llms.ProtocolAnthropicMessages {
		headers["anthropic-version"] = cfg.APIVersion
	}
	headers["Accept"] = "application/json"
	return headers
}

func getLLMContextWindowMetadata(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, modelName string) (int, error) {
	payload, err := getLLMModelMetadataPayload(ctx, client, endpoint, headers)
	if err != nil {
		return 0, err
	}
	if matched := findLLMModelMetadata(payload, modelName); matched != nil {
		if value := findContextWindowValue(matched); value > 0 {
			return value, nil
		}
	}
	if value := findContextWindowValue(payload); value > 0 {
		return value, nil
	}
	return 0, fmt.Errorf("context capacity not present in model metadata")
}

func getLLMMaxOutputTokensMetadata(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, modelName string) (int, error) {
	payload, err := getLLMModelMetadataPayload(ctx, client, endpoint, headers)
	if err != nil {
		return 0, err
	}
	if matched := findLLMModelMetadata(payload, modelName); matched != nil {
		if value := findMaxOutputTokenValue(matched); value > 0 {
			return value, nil
		}
	}
	if value := findMaxOutputTokenValue(payload); value > 0 {
		return value, nil
	}
	return 0, fmt.Errorf("maximum output capacity not present in model metadata")
}

func getLLMModelMetadataPayload(ctx context.Context, client *http.Client, endpoint string, headers map[string]string) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLLMMetadataBodyBytes))
	if err != nil {
		return nil, err
	}
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func findLLMModelMetadata(value interface{}, modelName string) interface{} {
	wanted := strings.ToLower(strings.TrimSpace(modelName))
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, key := range []string{"id", "model", "name", "model_id"} {
			if got, ok := typed[key].(string); ok && strings.ToLower(strings.TrimSpace(got)) == wanted {
				return typed
			}
		}
		for _, nested := range typed {
			if matched := findLLMModelMetadata(nested, modelName); matched != nil {
				return matched
			}
		}
	case []interface{}:
		for _, nested := range typed {
			if matched := findLLMModelMetadata(nested, modelName); matched != nil {
				return matched
			}
		}
	}
	return nil
}

func findContextWindowValue(value interface{}) int {
	return findMetadataTokenValue(value, contextWindowMetadataKeys)
}

func findMaxOutputTokenValue(value interface{}) int {
	return findMetadataTokenValue(value, maxOutputTokenMetadataKeys)
}

func findMetadataTokenValue(value interface{}, keys map[string]struct{}) int {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, raw := range typed {
			normalizedKey := strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(strings.ToLower(key))
			if _, ok := keys[normalizedKey]; ok {
				if parsed := positiveMetadataInt(raw); parsed >= 1024 && parsed <= 10000000 {
					return parsed
				}
			}
		}
		for _, nested := range typed {
			if parsed := findMetadataTokenValue(nested, keys); parsed > 0 {
				return parsed
			}
		}
	case []interface{}:
		for _, nested := range typed {
			if parsed := findMetadataTokenValue(nested, keys); parsed > 0 {
				return parsed
			}
		}
	}
	return 0
}

func positiveMetadataInt(value interface{}) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := strconv.Atoi(typed.String())
		return parsed
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	case int:
		return typed
	}
	return 0
}
