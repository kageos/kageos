package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

const ConnectorGlobalResourcePath = apicall.ConnectorGlobalResourcePath

type ConnectorRequest = dto.ConnectorProxyReq

type ConnectorResponse = dto.ConnectorProxyResp

func (c *Context) GetConnector(provider string) (*dto.ResolveConnectorBindingResp, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	resourcePath := strings.TrimSpace(c.GetFullCodePath())
	if resourcePath == "" {
		resourcePath = ConnectorGlobalResourcePath
	}
	return apicall.ResolveConnectorBinding(c.apiCallContext(), provider, resourcePath)
}

func (c *Context) GetConnectorForResource(provider, resourcePath string) (*dto.ResolveConnectorBindingResp, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	return apicall.ResolveConnectorBinding(c.apiCallContext(), provider, resourcePath)
}

func (c *Context) CallConnector(provider string, req ConnectorRequest) (*ConnectorResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	return c.CallConnectorForResource(provider, c.GetFullCodePath(), req)
}

func (c *Context) CallConnectorForResource(provider, resourcePath string, req ConnectorRequest) (*ConnectorResponse, error) {
	if c == nil {
		return nil, fmt.Errorf("context is nil")
	}
	req.Provider = strings.TrimSpace(provider)
	if strings.TrimSpace(req.Method) == "" {
		req.Method = http.MethodGet
	}
	if strings.TrimSpace(req.ResourcePath) == "" {
		req.ResourcePath = strings.TrimSpace(resourcePath)
	}
	if strings.TrimSpace(req.ResourcePath) == "" {
		req.ResourcePath = ConnectorGlobalResourcePath
	}
	return apicall.ProxyConnector(c.apiCallContext(), &req)
}

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

func normalizeConnectorEndpoints(endpoints []ConnectorEndpoint) []ConnectorEndpoint {
	seen := make(map[string]int, len(endpoints))
	out := make([]ConnectorEndpoint, 0, len(endpoints))
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
			out[index].RequiredScopes = mergeConnectorEndpointScopes(out[index].RequiredScopes, endpoint.RequiredScopes)
			continue
		}
		seen[key] = len(out)
		out = append(out, ConnectorEndpoint{
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

func mergeConnectorEndpointScopes(base []string, extra []string) []string {
	merged := append(append([]string{}, base...), extra...)
	return normalizeConnectorEndpointScopes(merged)
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

func connectorCodesFromEndpoints(endpoints []ConnectorEndpoint) []string {
	codes := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		codes = append(codes, endpoint.Provider)
	}
	return normalizeConnectorCodes(codes)
}
