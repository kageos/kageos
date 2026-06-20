package cert_manager

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const aliyunAPIEndpoint = "https://alidns.aliyuncs.com/"

var aliyunHTTPClient = &http.Client{Timeout: 15 * time.Second}

type aliyunCredentials struct {
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
}

type aliyunZone struct {
	ID   string
	Name string
}

type aliyunDNSRecord struct {
	ID      string
	Type    string
	Name    string
	Content string
}

func encodeAliyunCredentials(accessKeyID string, accessKeySecret string) string {
	data, _ := json.Marshal(aliyunCredentials{
		AccessKeyID:     strings.TrimSpace(accessKeyID),
		AccessKeySecret: strings.TrimSpace(accessKeySecret),
	})
	return string(data)
}

func parseAliyunCredentials(value string) (*aliyunCredentials, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("阿里云 AccessKey 未配置")
	}
	var creds aliyunCredentials
	if strings.HasPrefix(value, "{") {
		if err := json.Unmarshal([]byte(value), &creds); err != nil {
			return nil, fmt.Errorf("解析阿里云 AccessKey 失败: %w", err)
		}
	} else {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("阿里云 AccessKey 格式错误")
		}
		creds.AccessKeyID = parts[0]
		creds.AccessKeySecret = parts[1]
	}
	creds.AccessKeyID = strings.TrimSpace(creds.AccessKeyID)
	creds.AccessKeySecret = strings.TrimSpace(creds.AccessKeySecret)
	if creds.AccessKeyID == "" || creds.AccessKeySecret == "" {
		return nil, fmt.Errorf("阿里云 AccessKey ID 和 AccessKey Secret 不能为空")
	}
	return &creds, nil
}

func verifyAliyunToken(token string) error {
	params := url.Values{}
	params.Set("PageNumber", "1")
	params.Set("PageSize", "1")
	var raw json.RawMessage
	return aliyunRequest(token, "DescribeDomains", params, &raw)
}

func findAliyunZone(token string, domain string) (*aliyunZone, error) {
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
		zone, err := getAliyunZoneByName(token, candidate)
		if err != nil {
			return nil, err
		}
		if zone != nil {
			return zone, nil
		}
	}
	return nil, fmt.Errorf("阿里云 DNS 中未找到域名 %s 对应的 Zone，请检查 AccessKey 权限或域名是否托管在该账号下", domain)
}

func getAliyunZoneByName(token string, name string) (*aliyunZone, error) {
	params := url.Values{}
	params.Set("KeyWord", name)
	params.Set("PageNumber", "1")
	params.Set("PageSize", "10")
	var out struct {
		Domains struct {
			Domain []struct {
				DomainName string `json:"DomainName"`
			} `json:"Domain"`
		} `json:"Domains"`
	}
	if err := aliyunRequest(token, "DescribeDomains", params, &out); err != nil {
		return nil, err
	}
	for _, item := range out.Domains.Domain {
		if strings.EqualFold(item.DomainName, name) {
			return &aliyunZone{ID: item.DomainName, Name: item.DomainName}, nil
		}
	}
	return nil, nil
}

func createAliyunTXTRecord(token string, zoneID string, name string, content string) (*aliyunDNSRecord, error) {
	zoneName := strings.TrimSpace(zoneID)
	rr := dnsProviderRelativeRecordName(name, zoneName)
	params := url.Values{}
	params.Set("DomainName", zoneName)
	params.Set("RR", rr)
	params.Set("Type", "TXT")
	params.Set("Value", content)
	params.Set("TTL", "60")
	var out struct {
		RecordID string `json:"RecordId"`
	}
	if err := aliyunRequest(token, "AddDomainRecord", params, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.RecordID) == "" {
		return nil, fmt.Errorf("阿里云创建 TXT 记录未返回 RecordId")
	}
	return &aliyunDNSRecord{ID: out.RecordID, Type: "TXT", Name: rr, Content: content}, nil
}

func deleteAliyunDNSRecord(token string, zoneID string, recordID string) error {
	if strings.TrimSpace(recordID) == "" {
		return nil
	}
	params := url.Values{}
	params.Set("RecordId", recordID)
	var raw json.RawMessage
	return aliyunRequest(token, "DeleteDomainRecord", params, &raw)
}

// aliyunRequest signs AliDNS RPC requests with HMAC-SHA1. The canonical query
// must be built before adding Signature; do not log credentials or TXT values.
func aliyunRequest(token string, action string, params url.Values, out interface{}) error {
	creds, err := parseAliyunCredentials(token)
	if err != nil {
		return err
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2015-01-09")
	params.Set("Format", "JSON")
	params.Set("AccessKeyId", creds.AccessKeyID)
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", uuid.NewString())
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))

	canonical := aliyunCanonicalQuery(params)
	stringToSign := "GET&%2F&" + aliyunPercentEncode(canonical)
	mac := hmac.New(sha1.New, []byte(creds.AccessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	params.Set("Signature", base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	req, err := http.NewRequest(http.MethodGet, aliyunAPIEndpoint+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	httpResp, err := aliyunHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求阿里云 DNS API 失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 2<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("阿里云 DNS API HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	var base struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	_ = json.Unmarshal(respBody, &base)
	if base.Code != "" {
		return fmt.Errorf("阿里云 DNS API 失败: %s %s", base.Code, base.Message)
	}
	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("解析阿里云 DNS 响应失败: %w", err)
		}
	}
	return nil
}

func aliyunCanonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		for _, value := range values[key] {
			parts = append(parts, aliyunPercentEncode(key)+"="+aliyunPercentEncode(value))
		}
	}
	return strings.Join(parts, "&")
}

func aliyunPercentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// dnsProviderRelativeRecordName converts a full ACME record name into the host
// record expected by AliDNS, for example _acme-challenge.foo under example.com.
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
