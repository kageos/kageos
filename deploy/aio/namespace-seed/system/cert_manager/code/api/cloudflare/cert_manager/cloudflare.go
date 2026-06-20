package cert_manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const cloudflareAPIBaseURL = "https://api.cloudflare.com/client/v4"

var cloudflareHTTPClient = &http.Client{Timeout: 15 * time.Second}

type cloudflareBaseResp struct {
	Success  bool              `json:"success"`
	Errors   []cloudflareError `json:"errors"`
	Messages []cloudflareError `json:"messages"`
	Result   json.RawMessage   `json:"result"`
}

type cloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cloudflareDNSRecord struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

func verifyCloudflareToken(token string) error {
	var raw json.RawMessage
	if err := cloudflareRequest(token, http.MethodGet, "/user/tokens/verify", nil, &raw); err != nil {
		return err
	}
	return nil
}

func findCloudflareZone(token string, domain string) (*cloudflareZone, error) {
	domain, err := normalizeDomainName(domain)
	if err != nil {
		return nil, err
	}
	domain = strings.TrimPrefix(domain, "*.")
	parts := strings.Split(domain, ".")
	// Try the longest suffix first so app.foo.example.com resolves to foo.example.com
	// before falling back to example.com when both are managed zones.
	for i := 0; i < len(parts)-1; i++ {
		candidate := strings.Join(parts[i:], ".")
		zone, err := getCloudflareZoneByName(token, candidate)
		if err != nil {
			return nil, err
		}
		if zone != nil {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("Cloudflare 中未找到域名 %s 对应的 Zone，请检查 Token 权限或域名是否托管在该账号下", domain)
}

func getCloudflareZoneByName(token string, name string) (*cloudflareZone, error) {
	values := url.Values{}
	values.Set("name", name)
	values.Set("status", "active")
	var zones []cloudflareZone
	if err := cloudflareRequest(token, http.MethodGet, "/zones?"+values.Encode(), nil, &zones); err != nil {
		return nil, err
	}
	if len(zones) == 0 {
		return nil, nil
	}
	return &zones[0], nil
}

func createCloudflareTXTRecord(token string, zoneID string, name string, content string) (*cloudflareDNSRecord, error) {
	payload := map[string]interface{}{
		"type":    "TXT",
		"name":    strings.TrimSuffix(name, "."),
		"content": content,
		"ttl":     120,
		"comment": "KageOS Cloudflare 证书管家 ACME DNS-01 challenge",
	}
	var record cloudflareDNSRecord
	if err := cloudflareRequest(token, http.MethodPost, "/zones/"+url.PathEscape(zoneID)+"/dns_records", payload, &record); err != nil {
		return nil, err
	}
	if strings.TrimSpace(record.ID) == "" {
		return nil, fmt.Errorf("Cloudflare 创建 TXT 记录未返回记录 ID")
	}
	return &record, nil
}

func deleteCloudflareDNSRecord(token string, zoneID string, recordID string) error {
	if strings.TrimSpace(zoneID) == "" || strings.TrimSpace(recordID) == "" {
		return nil
	}
	var raw json.RawMessage
	return cloudflareRequest(token, http.MethodDelete, "/zones/"+url.PathEscape(zoneID)+"/dns_records/"+url.PathEscape(recordID), nil, &raw)
}

// cloudflareRequest is the single API boundary for this provider. Keep it small
// and never log tokens or request bodies here because bodies may contain TXT values.
func cloudflareRequest(token string, method string, path string, payload interface{}, out interface{}) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("Cloudflare API Token 不能为空")
	}
	endpoint := cloudflareAPIBaseURL + path
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("序列化 Cloudflare 请求失败: %w", err)
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	httpResp, err := cloudflareHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求 Cloudflare API 失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 2<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("Cloudflare API HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	var base cloudflareBaseResp
	if err := json.Unmarshal(respBody, &base); err != nil {
		return fmt.Errorf("解析 Cloudflare 响应失败: %w", err)
	}
	if !base.Success {
		return fmt.Errorf("Cloudflare API 失败: %s", cloudflareErrorsText(base.Errors))
	}
	if out != nil && len(base.Result) > 0 {
		if err := json.Unmarshal(base.Result, out); err != nil {
			return fmt.Errorf("解析 Cloudflare result 失败: %w", err)
		}
	}
	return nil
}

func cloudflareErrorsText(errors []cloudflareError) string {
	if len(errors) == 0 {
		return "未知错误"
	}
	parts := make([]string, 0, len(errors))
	for _, item := range errors {
		if item.Code > 0 {
			parts = append(parts, fmt.Sprintf("[%d] %s", item.Code, item.Message))
		} else {
			parts = append(parts, item.Message)
		}
	}
	return strings.Join(parts, "; ")
}
