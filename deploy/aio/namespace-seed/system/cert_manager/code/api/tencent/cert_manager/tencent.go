package cert_manager

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	tencentAPIEndpoint = "https://dnspod.tencentcloudapi.com"
	tencentAPIHost     = "dnspod.tencentcloudapi.com"
	tencentAPIService  = "dnspod"
	tencentAPIVersion  = "2021-03-23"
	tencentAPIRegion   = "ap-guangzhou"
)

var tencentHTTPClient = &http.Client{Timeout: 15 * time.Second}

type tencentCredentials struct {
	SecretID  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`
}

type tencentZone struct {
	ID   string
	Name string
}

type tencentDNSRecord struct {
	ID      string
	Type    string
	Name    string
	Content string
}

func encodeTencentCredentials(secretID string, secretKey string) string {
	data, _ := json.Marshal(tencentCredentials{
		SecretID:  strings.TrimSpace(secretID),
		SecretKey: strings.TrimSpace(secretKey),
	})
	return string(data)
}

func parseTencentCredentials(value string) (*tencentCredentials, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("腾讯云 SecretId/SecretKey 未配置")
	}
	var creds tencentCredentials
	if strings.HasPrefix(value, "{") {
		if err := json.Unmarshal([]byte(value), &creds); err != nil {
			return nil, fmt.Errorf("解析腾讯云密钥失败: %w", err)
		}
	} else {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("腾讯云密钥格式错误")
		}
		creds.SecretID = parts[0]
		creds.SecretKey = parts[1]
	}
	creds.SecretID = strings.TrimSpace(creds.SecretID)
	creds.SecretKey = strings.TrimSpace(creds.SecretKey)
	if creds.SecretID == "" || creds.SecretKey == "" {
		return nil, fmt.Errorf("腾讯云 SecretId 和 SecretKey 不能为空")
	}
	return &creds, nil
}

func verifyTencentToken(token string) error {
	var raw json.RawMessage
	return tencentRequest(token, "DescribeDomainList", map[string]interface{}{
		"Offset": int64(0),
		"Limit":  int64(1),
	}, &raw)
}

func findTencentZone(token string, domain string) (*tencentZone, error) {
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
		zone, err := getTencentZoneByName(token, candidate)
		if err != nil {
			return nil, err
		}
		if zone != nil {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("腾讯云 DNSPod 中未找到域名 %s 对应的 Zone，请检查 SecretId 权限或域名是否托管在该账号下", domain)
}

func getTencentZoneByName(token string, name string) (*tencentZone, error) {
	var raw json.RawMessage
	if err := tencentRequest(token, "DescribeRecordList", map[string]interface{}{
		"Domain": name,
		"Offset": int64(0),
		"Limit":  int64(1),
	}, &raw); err != nil {
		if strings.Contains(err.Error(), "ResourceNotFound") || strings.Contains(err.Error(), "InvalidParameterValue.DomainNotExists") {
			return nil, nil
		}
		return nil, err
	}
	return &tencentZone{ID: name, Name: name}, nil
}

func createTencentTXTRecord(token string, zoneID string, name string, content string) (*tencentDNSRecord, error) {
	zoneName := strings.TrimSpace(zoneID)
	subDomain := dnsProviderRelativeRecordName(name, zoneName)
	var out struct {
		RecordID int64 `json:"RecordId"`
	}
	if err := tencentRequest(token, "CreateRecord", map[string]interface{}{
		"Domain":     zoneName,
		"SubDomain":  subDomain,
		"RecordType": "TXT",
		"RecordLine": "默认",
		"Value":      content,
		"TTL":        uint64(600),
	}, &out); err != nil {
		return nil, err
	}
	if out.RecordID == 0 {
		return nil, fmt.Errorf("腾讯云创建 TXT 记录未返回 RecordId")
	}
	return &tencentDNSRecord{ID: strconv.FormatInt(out.RecordID, 10), Type: "TXT", Name: subDomain, Content: content}, nil
}

func deleteTencentDNSRecord(token string, zoneID string, recordID string) error {
	recordID = strings.TrimSpace(recordID)
	if recordID == "" {
		return nil
	}
	id, err := strconv.ParseInt(recordID, 10, 64)
	if err != nil {
		return fmt.Errorf("腾讯云 RecordId 格式错误: %w", err)
	}
	var raw json.RawMessage
	return tencentRequest(token, "DeleteRecord", map[string]interface{}{
		"Domain":   strings.TrimSpace(zoneID),
		"RecordId": id,
	}, &raw)
}

// tencentRequest signs DNSPod API 3.0 JSON requests with TC3-HMAC-SHA256. Keep
// this boundary free of secret, payload and TXT-value logging.
func tencentRequest(token string, action string, payload map[string]interface{}, out interface{}) error {
	creds, err := parseTencentCredentials(token)
	if err != nil {
		return err
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化腾讯云请求失败: %w", err)
	}
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	contentType := "application/json; charset=utf-8"
	hashedRequestPayload := sha256Hex(body)
	canonicalHeaders := "content-type:" + contentType + "\n" + "host:" + tencentAPIHost + "\n"
	signedHeaders := "content-type;host"
	canonicalRequest := strings.Join([]string{
		http.MethodPost,
		"/",
		"",
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload,
	}, "\n")
	credentialScope := date + "/" + tencentAPIService + "/tc3_request"
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256",
		strconv.FormatInt(timestamp, 10),
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")
	secretDate := hmacSHA256([]byte("TC3"+creds.SecretKey), date)
	secretService := hmacSHA256(secretDate, tencentAPIService)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.SecretID, credentialScope, signedHeaders, signature)

	req, err := http.NewRequest(http.MethodPost, tencentAPIEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", tencentAPIHost)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Version", tencentAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Region", tencentAPIRegion)

	httpResp, err := tencentHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求腾讯云 DNSPod API 失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 2<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("腾讯云 DNSPod API HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	var base struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			RequestID string `json:"RequestId"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(respBody, &base); err != nil {
		return fmt.Errorf("解析腾讯云 DNSPod 响应失败: %w", err)
	}
	if base.Response.Error != nil {
		return fmt.Errorf("腾讯云 DNSPod API 失败: %s %s", base.Response.Error.Code, base.Response.Error.Message)
	}
	if out != nil {
		var wrapped struct {
			Response json.RawMessage `json:"Response"`
		}
		if err := json.Unmarshal(respBody, &wrapped); err != nil {
			return fmt.Errorf("解析腾讯云 DNSPod 响应失败: %w", err)
		}
		if err := json.Unmarshal(wrapped.Response, out); err != nil {
			return fmt.Errorf("解析腾讯云 DNSPod Response 失败: %w", err)
		}
	}
	return nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

// dnsProviderRelativeRecordName converts a full ACME record name into the host
// record expected by DNSPod, for example _acme-challenge.foo under example.com.
func dnsProviderRelativeRecordName(fqdn string, zoneName string) string {
	name := strings.TrimSuffix(strings.TrimSpace(fqdn), ".")
	zoneName = strings.TrimSuffix(strings.TrimSpace(zoneName), ".")
	suffix := "." + zoneName
	if strings.EqualFold(name, zoneName) {
		return "@"
	}
	if strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
		rr := strings.TrimSuffix(name[:len(name)-len(suffix)], ".")
		if rr != "" {
			return rr
		}
	}
	return name
}
