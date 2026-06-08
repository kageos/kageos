package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

func normalizeConnectorCodes(connectors []string) []string {
	seen := make(map[string]bool, len(connectors))
	out := make([]string, 0, len(connectors))
	for _, connector := range connectors {
		code := strings.ToLower(strings.TrimSpace(connector))
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	return out
}

func joinConnectorCodes(connectors []string) string {
	return strings.Join(normalizeConnectorCodes(connectors), ",")
}

func splitConnectorCodes(connectors string) []string {
	return normalizeConnectorCodes(strings.Split(connectors, ","))
}

func normalizeConnectorEndpoints(endpoints []dto.ConnectorEndpoint) []dto.ConnectorEndpoint {
	seen := make(map[string]int, len(endpoints))
	out := make([]dto.ConnectorEndpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		provider := strings.ToLower(strings.TrimSpace(endpoint.Provider))
		method := strings.ToUpper(strings.TrimSpace(endpoint.Method))
		url := strings.TrimSpace(endpoint.URL)
		if provider == "" || url == "" {
			continue
		}
		if method == "" {
			method = http.MethodGet
		}
		key := provider + "\x00" + method + "\x00" + url
		if index, ok := seen[key]; ok {
			out[index].RequiredScopes = mergeConnectorScopes(out[index].RequiredScopes, endpoint.RequiredScopes)
			continue
		}
		seen[key] = len(out)
		out = append(out, dto.ConnectorEndpoint{
			Provider:       provider,
			Method:         method,
			URL:            url,
			Name:           strings.TrimSpace(endpoint.Name),
			Desc:           strings.TrimSpace(endpoint.Desc),
			RequiredScopes: normalizeConnectorEndpointScopes(endpoint.RequiredScopes),
		})
	}
	return out
}

func normalizeConnectorEndpointScopes(scopes []string) []string {
	seen := make(map[string]bool, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		for _, part := range strings.Fields(strings.ReplaceAll(scope, ",", " ")) {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			out = append(out, part)
		}
	}
	return out
}

func connectorCodesFromEndpoints(endpoints []dto.ConnectorEndpoint) []string {
	codes := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		codes = append(codes, endpoint.Provider)
	}
	return normalizeConnectorCodes(codes)
}

func joinConnectorEndpoints(endpoints []dto.ConnectorEndpoint) string {
	normalized := normalizeConnectorEndpoints(endpoints)
	if len(normalized) == 0 {
		return ""
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(data)
}

func splitConnectorEndpoints(raw string) []dto.ConnectorEndpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var endpoints []dto.ConnectorEndpoint
	if err := json.Unmarshal([]byte(raw), &endpoints); err != nil {
		return nil
	}
	return normalizeConnectorEndpoints(endpoints)
}

func functionConnectorStatuses(ctx context.Context, fullCodePath string, connectors []string, endpoints []dto.ConnectorEndpoint) []dto.FunctionConnectorStatus {
	requiredScopes := connectorScopesByProvider(connectors, endpoints)
	codes := make([]string, 0, len(requiredScopes))
	for provider := range requiredScopes {
		codes = append(codes, provider)
	}
	codes = normalizeConnectorCodes(codes)
	sort.Strings(codes)
	if len(codes) == 0 {
		return nil
	}
	providerDisplays := connectorProviderDisplays(ctx, codes)
	connectorResourcePath := apicall.ConnectorGlobalResourcePath
	statuses := make([]dto.FunctionConnectorStatus, 0, len(codes))
	for _, provider := range codes {
		scopes := requiredScopes[provider]
		status := dto.FunctionConnectorStatus{
			Provider:       provider,
			Required:       true,
			RequiredScopes: scopes,
		}
		applyConnectorProviderDisplay(&status, providerDisplays[provider])
		resp, err := apicall.ResolveConnectorBindingWithScopes(ctx, provider, connectorResourcePath, scopes)
		if err != nil {
			status.Message = err.Error()
			statuses = append(statuses, status)
			continue
		}
		if resp == nil {
			status.Message = "连接器解析结果为空"
			statuses = append(statuses, status)
			continue
		}
		status.Connected = true
		status.ConnectionID = resp.Connection.ConnectionID
		status.DisplayName = resp.Connection.DisplayName
		status.Profile = resp.Connection.Profile
		status.ResolvedFrom = resp.ResolvedFrom
		status.GrantedScopes = resp.GrantedScopes
		status.MissingScopes = resp.MissingScopes
		if len(status.MissingScopes) > 0 {
			status.Message = "缺少权限: " + strings.Join(status.MissingScopes, "、")
		}
		statuses = append(statuses, status)
	}
	return statuses
}

func connectorProviderDisplays(ctx context.Context, providers []string) map[string]*dto.ConnectorOAuthProviderInfo {
	displays := make(map[string]*dto.ConnectorOAuthProviderInfo, len(providers))
	for _, provider := range normalizeConnectorCodes(providers) {
		resp, err := apicall.GetConnectorOAuthProvider(ctx, provider)
		if err != nil || resp == nil {
			continue
		}
		info := resp.Provider
		displays[provider] = &info
	}
	return displays
}

func applyConnectorProviderDisplay(status *dto.FunctionConnectorStatus, provider *dto.ConnectorOAuthProviderInfo) {
	if status == nil || provider == nil {
		return
	}
	status.ProviderName = strings.TrimSpace(provider.Name)
	status.ProviderLogoURL = strings.TrimSpace(provider.LogoURL)
	status.ProviderBrandColor = strings.TrimSpace(provider.BrandColor)
	status.ProviderAccountURL = strings.TrimSpace(provider.ProviderAccountURL)
}

func missingConnectorProviders(statuses []dto.FunctionConnectorStatus) []string {
	missing := make([]string, 0)
	for _, status := range statuses {
		if status.Required && (!status.Connected || len(status.MissingScopes) > 0) {
			missing = append(missing, status.Provider)
		}
	}
	return missing
}

func connectorDependencyError(statuses []dto.FunctionConnectorStatus) error {
	messages := make([]string, 0)
	for _, status := range statuses {
		if !status.Required {
			continue
		}
		if !status.Connected {
			messages = append(messages, fmt.Sprintf("%s 未连接", status.Provider))
			continue
		}
		if len(status.MissingScopes) > 0 {
			messages = append(messages, fmt.Sprintf("%s 缺少权限 %s", status.Provider, strings.Join(status.MissingScopes, "、")))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return fmt.Errorf("函数依赖连接器未就绪：%s，请先完成连接或补充授权后再执行", strings.Join(messages, "；"))
}

func connectorScopesByProvider(connectors []string, endpoints []dto.ConnectorEndpoint) map[string][]string {
	out := make(map[string][]string)
	for _, provider := range normalizeConnectorCodes(connectors) {
		out[provider] = nil
	}
	for _, endpoint := range normalizeConnectorEndpoints(endpoints) {
		provider := strings.ToLower(strings.TrimSpace(endpoint.Provider))
		if provider == "" {
			continue
		}
		out[provider] = mergeConnectorScopes(out[provider], endpoint.RequiredScopes)
	}
	return out
}

func mergeConnectorScopes(base []string, extra []string) []string {
	merged := append(append([]string{}, base...), extra...)
	return normalizeConnectorEndpointScopes(merged)
}

func requestFunctionFullCodePath(req *dto.RequestAppReq) string {
	if req == nil {
		return ""
	}
	router := strings.TrimSpace(req.Router)
	if strings.Trim(router, "/") == "_callback" {
		if callbackRouter := callbackTargetRouter(req.Body); callbackRouter != "" {
			router = callbackRouter
		}
	}
	router = strings.Trim(router, "/")
	if req.User == "" || req.App == "" {
		return ""
	}
	appPrefix := fmt.Sprintf("/%s/%s", req.User, req.App)
	if router == "" {
		return appPrefix
	}
	normalizedRouter := "/" + router
	if normalizedRouter == appPrefix || strings.HasPrefix(normalizedRouter, appPrefix+"/") {
		return normalizedRouter
	}
	return fmt.Sprintf("/%s/%s/%s", req.User, req.App, router)
}

func callbackTargetRouter(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var envelope struct {
		Router string `json:"router"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return ""
	}
	return envelope.Router
}
