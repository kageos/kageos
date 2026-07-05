package llms

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxErrorBodyBytes = 4096

func postJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body interface{}) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range cleanHeaderMap(headers) {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return resp, nil
}

func buildEndpoint(baseURL, endpointPath, protocol string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(defaultBaseURL(protocol), "/")
	}
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		if endpointAlreadyIncluded(baseURL, protocol) {
			return baseURL
		}
		endpointPath = defaultEndpointPath(protocol)
	}
	if strings.HasPrefix(endpointPath, "http://") || strings.HasPrefix(endpointPath, "https://") {
		return endpointPath
	}
	if endpointPath == "" {
		return baseURL
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	return baseURL + endpointPath
}

func endpointAlreadyIncluded(baseURL, protocol string) bool {
	baseURL = strings.TrimRight(strings.ToLower(strings.TrimSpace(baseURL)), "/")
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case ProtocolOpenAIResponses:
		return strings.HasSuffix(baseURL, "/responses")
	case ProtocolAnthropicMessages:
		return strings.HasSuffix(baseURL, "/v1/messages") || strings.HasSuffix(baseURL, "/messages")
	default:
		return strings.HasSuffix(baseURL, "/chat/completions")
	}
}

func joinURL(baseURL, endpointPath string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		return baseURL
	}
	if strings.HasPrefix(endpointPath, "http://") || strings.HasPrefix(endpointPath, "https://") {
		return endpointPath
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	return baseURL + endpointPath
}

func applyAuthHeader(headers map[string]string, apiKey, authScheme, protocol string) map[string]string {
	out := map[string]string{}
	for key, value := range cleanHeaderMap(headers) {
		out[key] = value
	}
	apiKey = strings.TrimSpace(apiKey)
	authScheme = strings.ToLower(strings.TrimSpace(authScheme))
	if authScheme == "" {
		authScheme = defaultAuthScheme(protocol)
	}
	if authScheme == AuthSchemeNone || apiKey == "" {
		return out
	}
	switch authScheme {
	case AuthSchemeXAPIKey:
		out["x-api-key"] = apiKey
	default:
		out["Authorization"] = "Bearer " + apiKey
	}
	return out
}

func readSSE(ctx context.Context, body io.Reader, handle func(event, data string) error) error {
	reader := bufio.NewReader(body)
	event := ""
	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			event = ""
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = nil
		curEvent := event
		event = ""
		if strings.TrimSpace(data) == "[DONE]" {
			return io.EOF
		}
		return handle(curEvent, data)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := reader.ReadString('\n')
		if err != nil && len(line) == 0 {
			if err == io.EOF {
				if flushErr := flush(); flushErr != nil && flushErr != io.EOF {
					return flushErr
				}
				return nil
			}
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if flushErr := flush(); flushErr != nil {
				if flushErr == io.EOF {
					return nil
				}
				return flushErr
			}
		} else if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
		if err == io.EOF {
			if flushErr := flush(); flushErr != nil && flushErr != io.EOF {
				return flushErr
			}
			return nil
		}
	}
}

func parseJSONMap(text string) (map[string]interface{}, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseJSONStringMap(text string) (map[string]string, error) {
	obj, err := parseJSONMap(text)
	if err != nil || obj == nil {
		return nil, err
	}
	out := make(map[string]string, len(obj))
	for key, value := range obj {
		switch v := value.(type) {
		case string:
			out[key] = v
		default:
			b, _ := json.Marshal(v)
			out[key] = string(b)
		}
	}
	return out, nil
}

func appendQuery(rawURL, key, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	if q.Get(key) == "" {
		q.Set(key, value)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
