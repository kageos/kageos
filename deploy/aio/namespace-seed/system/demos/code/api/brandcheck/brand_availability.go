package brandcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

type BrandCheckHistory struct {
	ID                 int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt          types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:检查时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	BrandName          string         `json:"brand_name" gorm:"column:brand_name;comment:品牌名;index" widget:"name:品牌名;type:input" validate:"required"`
	NormalizedName     string         `json:"normalized_name" gorm:"column:normalized_name;comment:规范化名称;index" widget:"name:规范化名称;type:input" hide:"create,update"`
	Scenario           string         `json:"scenario" gorm:"column:scenario;comment:业务描述" widget:"name:业务描述;type:text_area"`
	Score              int            `json:"score" gorm:"column:score;comment:可用性评分" widget:"name:可用性评分;type:progress;min:0;max:100;unit:%" hide:"create,update"`
	Verdict            string         `json:"verdict" gorm:"column:verdict;comment:结论" widget:"name:结论;type:select;options:强可用,可考虑,风险较高;options_colors:67C23A,E6A23C,F56C6C" hide:"create,update"`
	AvailableCount     int            `json:"available_count" gorm:"column:available_count;comment:可用项数量" widget:"name:可用项;type:integer" hide:"create,update"`
	UnavailableCount   int            `json:"unavailable_count" gorm:"column:unavailable_count;comment:被占用项数量" widget:"name:被占用项;type:integer" hide:"create,update"`
	UnknownCount       int            `json:"unknown_count" gorm:"column:unknown_count;comment:未知项数量" widget:"name:未知项;type:integer" hide:"create,update"`
	AvailableDomains   string         `json:"available_domains" gorm:"column:available_domains;type:text;comment:可用域名" widget:"name:可用域名;type:text_area" hide:"create,update"`
	UnavailableDomains string         `json:"unavailable_domains" gorm:"column:unavailable_domains;type:text;comment:被占用域名" widget:"name:被占用域名;type:text_area" hide:"create,update"`
	Summary            string         `json:"summary" gorm:"column:summary;type:text;comment:摘要" widget:"name:摘要;type:text_area" hide:"create,update"`
	ReportFile         string         `json:"report_file" gorm:"column:report_file;type:text;comment:报告文件" widget:"name:报告文件;type:files" hide:"create,update"`
}

func (BrandCheckHistory) TableName() string {
	return "brand_check_history"
}

type BrandAvailabilityReq struct {
	BrandName               string `json:"brand_name" widget:"name:品牌名;type:input;placeholder:例如 KageOS" validate:"required"`
	Scenario                string `json:"scenario" widget:"name:业务描述;type:text_area;placeholder:可选，例如 self-hosted AI business app templates"`
	DomainSuffixes          string `json:"domain_suffixes" widget:"name:域名后缀;type:input;placeholder:默认 .com,.ai,.io,.dev,.app,.co"`
	CheckDeveloperPlatforms string `json:"check_developer_platforms" widget:"name:开发者平台;type:select;options:检查,不检查;options_colors:67C23A,909399;render_default:检查"`
}

type PlatformCheck struct {
	Platform string `json:"platform" widget:"name:平台;type:input"`
	Target   string `json:"target" widget:"name:检查对象;type:input"`
	Status   string `json:"status" widget:"name:状态;type:select;options:可用,被占用,未知;options_colors:67C23A,F56C6C,909399"`
	Reason   string `json:"reason" widget:"name:说明;type:input"`
	URL      string `json:"url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
}

type BrandAvailabilityResp struct {
	Score            int             `json:"score" widget:"name:可用性评分;type:progress;min:0;max:100;unit:%"`
	Verdict          string          `json:"verdict" widget:"name:结论;type:select;options:强可用,可考虑,风险较高;options_colors:67C23A,E6A23C,F56C6C"`
	NormalizedName   string          `json:"normalized_name" widget:"name:规范化名称;type:input"`
	Summary          string          `json:"summary" widget:"name:摘要;type:text_area"`
	PlatformChecks   []PlatformCheck `json:"platform_checks" widget:"name:平台检查结果;type:table"`
	AlternativeNames string          `json:"alternative_names" widget:"name:替代名称建议;type:text_area"`
	ManualSearch     []PlatformCheck `json:"manual_search" widget:"name:人工复查入口;type:table"`
	ReportFile       string          `json:"report_file" widget:"name:Markdown 报告;type:files"`
}

func BrandAvailabilityCheck(ctx *app.Context, resp response.Response) error {
	var req BrandAvailabilityReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, history, err := DoBrandAvailabilityCheck(ctx, &req)
	if err != nil {
		return err
	}
	if db := ctx.GetGormDB(); db != nil {
		if err := db.Create(history).Error; err != nil {
			logger.Warnf(ctx, "[BrandAvailabilityCheck] 保存检查历史失败: %v", err)
		}
	}
	return resp.Form(res).Build()
}

func DoBrandAvailabilityCheck(ctx *app.Context, req *BrandAvailabilityReq) (*BrandAvailabilityResp, *BrandCheckHistory, error) {
	normalized := normalizeBrandName(req.BrandName)
	if normalized == "" {
		return nil, nil, fmt.Errorf("品牌名至少需要包含一个英文字母或数字")
	}

	suffixes := parseDomainSuffixes(req.DomainSuffixes)
	checks := make([]PlatformCheck, 0, len(suffixes)+4)
	client := &http.Client{Timeout: 8 * time.Second}
	for _, suffix := range suffixes {
		domain := normalized + suffix
		checks = append(checks, checkDomain(client, domain))
	}

	if req.CheckDeveloperPlatforms != "不检查" {
		checks = append(checks,
			checkHTTPStatus(client, "GitHub", normalized, "https://api.github.com/users/"+url.PathEscape(normalized), "https://github.com/"+normalized, 404),
			checkGitLabUser(client, normalized),
			checkGitLabGroup(client, normalized),
			checkNPMPackage(client, normalized),
			checkNPMAccount(client, normalized),
			checkHTTPStatus(client, "PyPI", normalized, "https://pypi.org/pypi/"+url.PathEscape(normalized)+"/json", "https://pypi.org/project/"+normalized+"/", 404),
			checkDockerNamespace(client, normalized),
			checkRedditSubreddit(client, normalized),
			checkProfilePage(client, "X handle", normalized, "https://x.com/"+normalized, []string{"this account doesn", "account suspended"}, "X 页面常返回软 200，建议人工复查"),
			checkLinkedInCompany(client, normalized),
		)
	}

	score, verdict, available, unavailable, unknown := scoreChecks(checks)
	availableDomains, unavailableDomains := splitDomains(checks)
	alternatives := suggestAlternativeNames(normalized)
	summary := buildBrandSummary(req.BrandName, normalized, score, verdict, available, unavailable, unknown, availableDomains, unavailableDomains)
	manualSearch := buildManualSearchLinks(normalized)

	reportPath, reportRef, err := writeBrandReport(ctx, req, normalized, score, verdict, checks, alternatives, manualSearch, summary)
	if err != nil {
		return nil, nil, err
	}

	resp := &BrandAvailabilityResp{
		Score:            score,
		Verdict:          verdict,
		NormalizedName:   normalized,
		Summary:          summary,
		PlatformChecks:   checks,
		AlternativeNames: strings.Join(alternatives, "\n"),
		ManualSearch:     manualSearch,
		ReportFile:       reportRef,
	}
	history := &BrandCheckHistory{
		BrandName:          strings.TrimSpace(req.BrandName),
		NormalizedName:     normalized,
		Scenario:           strings.TrimSpace(req.Scenario),
		Score:              score,
		Verdict:            verdict,
		AvailableCount:     available,
		UnavailableCount:   unavailable,
		UnknownCount:       unknown,
		AvailableDomains:   strings.Join(availableDomains, "\n"),
		UnavailableDomains: strings.Join(unavailableDomains, "\n"),
		Summary:            summary,
		ReportFile:         reportPath,
	}
	return resp, history, nil
}

type BrandCheckHistoryReq struct {
	BrandName         string `json:"brand_name" form:"brand_name" widget:"name:品牌名;type:input"`
	Verdict           string `json:"verdict" form:"verdict" widget:"name:结论;type:select;options:强可用,可考虑,风险较高;options_colors:67C23A,E6A23C,F56C6C"`
	query.PageSortReq `widget:"-"`
}

func BrandCheckHistoryList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req BrandCheckHistoryReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&BrandCheckHistory{})
	if strings.TrimSpace(req.BrandName) != "" {
		queryDB = queryDB.Where("brand_name LIKE ? OR normalized_name LIKE ?", "%"+strings.TrimSpace(req.BrandName)+"%", "%"+normalizeBrandName(req.BrandName)+"%")
	}
	if strings.TrimSpace(req.Verdict) != "" {
		queryDB = queryDB.Where("verdict = ?", strings.TrimSpace(req.Verdict))
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var items []BrandCheckHistory
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func normalizeBrandName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 63 {
		out = out[:63]
		out = strings.Trim(out, "-")
	}
	return out
}

func parseDomainSuffixes(input string) []string {
	if strings.TrimSpace(input) == "" {
		return []string{".com", ".ai", ".io", ".dev", ".app", ".co"}
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		suffix := strings.ToLower(strings.TrimSpace(part))
		if suffix == "" {
			continue
		}
		if !strings.HasPrefix(suffix, ".") {
			suffix = "." + suffix
		}
		if _, ok := seen[suffix]; ok {
			continue
		}
		seen[suffix] = struct{}{}
		result = append(result, suffix)
	}
	if len(result) == 0 {
		return []string{".com", ".ai", ".io", ".dev", ".app", ".co"}
	}
	return result
}

func checkDomain(client *http.Client, domain string) PlatformCheck {
	rdapURL := "https://rdap.org/domain/" + url.PathEscape(domain)
	req, err := http.NewRequest(http.MethodGet, rdapURL, nil)
	if err == nil {
		req.Header.Set("Accept", "application/rdap+json, application/json")
		req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
		res, err := client.Do(req)
		if err == nil {
			defer res.Body.Close()
			switch {
			case res.StatusCode >= 200 && res.StatusCode < 300:
				return PlatformCheck{Platform: "Domain", Target: domain, Status: "被占用", Reason: fmt.Sprintf("RDAP 返回 HTTP %d，注册局已有该域名记录", res.StatusCode), URL: "https://" + domain}
			case res.StatusCode == http.StatusNotFound:
				return PlatformCheck{Platform: "Domain", Target: domain, Status: "可用", Reason: "RDAP 未找到注册记录，仍建议进入注册商复查", URL: "https://www.namecheap.com/domains/registration/results/?domain=" + url.QueryEscape(domain)}
			case res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusForbidden:
				return PlatformCheck{Platform: "Domain", Target: domain, Status: "未知", Reason: fmt.Sprintf("RDAP 访问受限，返回 HTTP %d", res.StatusCode), URL: "https://www.namecheap.com/domains/registration/results/?domain=" + url.QueryEscape(domain)}
			}
		}
	}

	lookupCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	_, err = net.DefaultResolver.LookupHost(lookupCtx, domain)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return PlatformCheck{Platform: "Domain", Target: domain, Status: "可用", Reason: "DNS 未解析到记录，适合进入注册商复查", URL: "https://www.namecheap.com/domains/registration/results/?domain=" + url.QueryEscape(domain)}
		}
		return PlatformCheck{Platform: "Domain", Target: domain, Status: "未知", Reason: "DNS 查询失败: " + err.Error(), URL: "https://www.namecheap.com/domains/registration/results/?domain=" + url.QueryEscape(domain)}
	}
	return PlatformCheck{Platform: "Domain", Target: domain, Status: "被占用", Reason: "DNS 已存在解析记录", URL: "https://" + domain}
}

func checkHTTPStatus(client *http.Client, platform, target, checkURL, displayURL string, availableStatus int) PlatformCheck {
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: platform, Target: target, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: platform, Target: target, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()

	if res.StatusCode == availableStatus {
		return PlatformCheck{Platform: platform, Target: target, Status: "可用", Reason: fmt.Sprintf("返回 HTTP %d", res.StatusCode), URL: displayURL}
	}
	if res.StatusCode >= 200 && res.StatusCode < 400 {
		return PlatformCheck{Platform: platform, Target: target, Status: "被占用", Reason: fmt.Sprintf("返回 HTTP %d", res.StatusCode), URL: displayURL}
	}
	if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusTooManyRequests {
		return PlatformCheck{Platform: platform, Target: target, Status: "未知", Reason: fmt.Sprintf("平台限制访问，返回 HTTP %d", res.StatusCode), URL: displayURL}
	}
	return PlatformCheck{Platform: platform, Target: target, Status: "未知", Reason: fmt.Sprintf("返回 HTTP %d，需要人工复查", res.StatusCode), URL: displayURL}
}

func checkNPMPackage(client *http.Client, name string) PlatformCheck {
	checkURL := "https://registry.npmjs.org/" + url.PathEscape(name)
	displayURL := "https://www.npmjs.com/package/" + name
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "npm package", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "npm package", Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return PlatformCheck{Platform: "npm package", Target: name, Status: "被占用", Reason: "registry.npmjs.org 已存在同名公开包", URL: displayURL}
	}
	if res.StatusCode == http.StatusNotFound {
		return PlatformCheck{Platform: "npm package", Target: name, Status: "可用", Reason: "未发现同名公开包；不代表 npm 用户名或组织名可用", URL: displayURL}
	}
	return PlatformCheck{Platform: "npm package", Target: name, Status: "未知", Reason: fmt.Sprintf("registry 返回 HTTP %d，需要人工复查", res.StatusCode), URL: displayURL}
}

func checkGitLabUser(client *http.Client, name string) PlatformCheck {
	checkURL := "https://gitlab.com/api/v4/users?username=" + url.QueryEscape(name)
	displayURL := "https://gitlab.com/" + name
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "GitLab user", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "GitLab user", Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	text := strings.TrimSpace(string(body))
	if res.StatusCode == http.StatusOK {
		if text == "[]" {
			return PlatformCheck{Platform: "GitLab user", Target: name, Status: "可用", Reason: "GitLab 用户 API 未发现同名用户", URL: displayURL}
		}
		return PlatformCheck{Platform: "GitLab user", Target: name, Status: "被占用", Reason: "GitLab 用户 API 返回同名用户", URL: displayURL}
	}
	return PlatformCheck{Platform: "GitLab user", Target: name, Status: "未知", Reason: fmt.Sprintf("GitLab 用户 API 返回 HTTP %d", res.StatusCode), URL: displayURL}
}

func checkGitLabGroup(client *http.Client, name string) PlatformCheck {
	checkURL := "https://gitlab.com/api/v4/groups/" + url.PathEscape(name)
	displayURL := "https://gitlab.com/" + name
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "GitLab group", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "GitLab group", Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return PlatformCheck{Platform: "GitLab group", Target: name, Status: "被占用", Reason: "GitLab group API 返回同名 group", URL: displayURL}
	}
	if res.StatusCode == http.StatusNotFound {
		return PlatformCheck{Platform: "GitLab group", Target: name, Status: "可用", Reason: "GitLab group API 返回 404", URL: displayURL}
	}
	return PlatformCheck{Platform: "GitLab group", Target: name, Status: "未知", Reason: fmt.Sprintf("GitLab group API 返回 HTTP %d", res.StatusCode), URL: displayURL}
}

func checkNPMAccount(client *http.Client, name string) PlatformCheck {
	checkURL := "https://registry.npmjs.org/-/user/org.couchdb.user:" + url.PathEscape(name)
	displayURL := "https://www.npmjs.com/~" + name
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "npm account/org", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "npm account/org", Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusOK {
		return PlatformCheck{Platform: "npm account/org", Target: name, Status: "被占用", Reason: "npm 用户/组织接口返回存在记录", URL: displayURL}
	}
	if res.StatusCode == http.StatusNotFound {
		return PlatformCheck{Platform: "npm account/org", Target: name, Status: "可用", Reason: "npm 用户/组织接口返回 404", URL: displayURL}
	}
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return PlatformCheck{Platform: "npm user", Target: name, Status: "未知", Reason: fmt.Sprintf("npm 用户接口返回 HTTP %d；请打开用户 profile 人工复查", res.StatusCode), URL: displayURL}
	}
	return PlatformCheck{Platform: "npm user", Target: name, Status: "未知", Reason: fmt.Sprintf("npm 用户接口返回 HTTP %d，需要人工复查", res.StatusCode), URL: displayURL}
}

func checkDockerNamespace(client *http.Client, name string) PlatformCheck {
	displayURL := "https://hub.docker.com/u/" + name
	profileReq, err := http.NewRequest(http.MethodGet, displayURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	profileReq.Header.Set("Accept", "text/html,application/xhtml+xml")
	profileReq.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	profileRes, err := client.Do(profileReq)
	if err == nil {
		defer profileRes.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(profileRes.Body, 8192))
		text := strings.ToLower(string(body))
		if profileRes.StatusCode == http.StatusOK && !strings.Contains(text, "page not found") {
			return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "被占用", Reason: "Docker Hub 公开 profile 页存在", URL: displayURL}
		}
		if profileRes.StatusCode == http.StatusNotFound || strings.Contains(text, "page not found") {
			return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "可用", Reason: "Docker Hub 公开 profile 页返回不存在；仍建议登录 Docker 复查", URL: displayURL}
		}
	}

	checkURL := "https://app.docker.com/resources/create-org/namespace-available.data?orgName=" + url.QueryEscape(name)
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "text/x-script, application/json, text/plain, */*")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	text := strings.ToLower(string(body))
	if res.StatusCode == http.StatusUnauthorized || strings.Contains(text, "unauthorized") {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: "Docker namespace availability 接口要求登录，当前环境无法可靠判断", URL: displayURL}
	}
	if strings.Contains(text, "available") && (strings.Contains(text, "true") || strings.Contains(text, ":1")) {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "可用", Reason: "Docker namespace-available 接口返回可用信号", URL: displayURL}
	}
	if strings.Contains(text, "available") && (strings.Contains(text, "false") || strings.Contains(text, ":0")) {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "被占用", Reason: "Docker namespace-available 接口返回不可用信号", URL: displayURL}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: "Docker 接口返回成功但响应格式未识别，需要人工复查", URL: displayURL}
	}
	return PlatformCheck{Platform: "Docker Hub namespace", Target: name, Status: "未知", Reason: fmt.Sprintf("Docker 接口返回 HTTP %d，需要人工复查", res.StatusCode), URL: displayURL}
}

func checkRedditSubreddit(client *http.Client, name string) PlatformCheck {
	redditName := strings.ReplaceAll(name, "-", "_")
	checkURL := "https://www.reddit.com/r/" + url.PathEscape(redditName) + "/about.json"
	displayURL := "https://www.reddit.com/r/" + redditName
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return PlatformCheck{Platform: "Reddit subreddit", Target: redditName, Status: "未知", Reason: err.Error(), URL: displayURL}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: "Reddit subreddit", Target: redditName, Status: "未知", Reason: "请求失败: " + err.Error(), URL: displayURL}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
	text := strings.ToLower(string(body))
	if res.StatusCode == http.StatusOK && strings.Contains(text, `"kind": "t5"`) {
		return PlatformCheck{Platform: "Reddit subreddit", Target: redditName, Status: "被占用", Reason: "Reddit about.json 返回 subreddit 对象", URL: displayURL}
	}
	if res.StatusCode == http.StatusNotFound || strings.Contains(text, `"dist": 0`) || strings.Contains(text, `"children": []`) {
		return PlatformCheck{Platform: "Reddit subreddit", Target: redditName, Status: "可用", Reason: "未发现同名 subreddit；Reddit 可能保留部分名称", URL: displayURL}
	}
	return PlatformCheck{Platform: "Reddit subreddit", Target: redditName, Status: "未知", Reason: fmt.Sprintf("Reddit 返回 HTTP %d，需要人工复查", res.StatusCode), URL: displayURL}
}

func checkProfilePage(client *http.Client, platform, name, pageURL string, notFoundMarkers []string, caution string) PlatformCheck {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return PlatformCheck{Platform: platform, Target: name, Status: "未知", Reason: err.Error(), URL: pageURL}
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "KageOS-Brand-Availability-Checker/0.1")
	res, err := client.Do(req)
	if err != nil {
		return PlatformCheck{Platform: platform, Target: name, Status: "未知", Reason: "请求失败: " + err.Error(), URL: pageURL}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 16384))
	text := strings.ToLower(string(body))
	for _, marker := range notFoundMarkers {
		if marker != "" && strings.Contains(text, strings.ToLower(marker)) {
			return PlatformCheck{Platform: platform, Target: name, Status: "可用", Reason: "页面包含不存在标记；仍建议人工复查", URL: pageURL}
		}
	}
	if res.StatusCode == http.StatusNotFound {
		return PlatformCheck{Platform: platform, Target: name, Status: "可用", Reason: "页面返回 404；仍建议人工复查", URL: pageURL}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		return PlatformCheck{Platform: platform, Target: name, Status: "未知", Reason: caution, URL: pageURL}
	}
	if res.StatusCode == http.StatusForbidden || res.StatusCode == http.StatusTooManyRequests || res.StatusCode == http.StatusUnavailableForLegalReasons {
		return PlatformCheck{Platform: platform, Target: name, Status: "未知", Reason: fmt.Sprintf("页面返回 HTTP %d，平台限制访问，需要人工复查", res.StatusCode), URL: pageURL}
	}
	return PlatformCheck{Platform: platform, Target: name, Status: "未知", Reason: fmt.Sprintf("页面返回 HTTP %d，需要人工复查", res.StatusCode), URL: pageURL}
}

func checkLinkedInCompany(client *http.Client, name string) PlatformCheck {
	pageURL := "https://www.linkedin.com/company/" + name
	return checkProfilePage(client, "LinkedIn company", name, pageURL, []string{"page not found", "company not found"}, "LinkedIn 经常按地区/登录状态重定向，建议人工复查")
}

func scoreChecks(checks []PlatformCheck) (int, string, int, int, int) {
	available, unavailable, unknown := 0, 0, 0
	for _, item := range checks {
		switch item.Status {
		case "可用":
			available++
		case "被占用":
			unavailable++
		default:
			unknown++
		}
	}
	total := len(checks)
	if total == 0 {
		return 0, "风险较高", available, unavailable, unknown
	}
	score := int(float64(available)*100/float64(total) + float64(unknown)*35/float64(total))
	if score > 100 {
		score = 100
	}
	verdict := "风险较高"
	if score >= 75 && unavailable <= 1 {
		verdict = "强可用"
	} else if score >= 45 {
		verdict = "可考虑"
	}
	return score, verdict, available, unavailable, unknown
}

func splitDomains(checks []PlatformCheck) ([]string, []string) {
	var available []string
	var unavailable []string
	for _, item := range checks {
		if item.Platform != "Domain" {
			continue
		}
		if item.Status == "可用" {
			available = append(available, item.Target)
		}
		if item.Status == "被占用" {
			unavailable = append(unavailable, item.Target)
		}
	}
	sort.Strings(available)
	sort.Strings(unavailable)
	return available, unavailable
}

func suggestAlternativeNames(normalized string) []string {
	base := strings.ReplaceAll(normalized, "-", "")
	if base == "" {
		base = normalized
	}
	candidates := []string{
		base + "hq",
		base + "app",
		base + "ai",
		base + "labs",
		base + "stack",
		base + "works",
		"use" + base,
		"try" + base,
		"get" + base,
		base + "cloud",
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(candidates))
	for _, item := range candidates {
		item = normalizeBrandName(item)
		if item == "" || item == normalized {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func buildBrandSummary(original, normalized string, score int, verdict string, available, unavailable, unknown int, availableDomains, unavailableDomains []string) string {
	lines := []string{
		fmt.Sprintf("品牌名: %s", strings.TrimSpace(original)),
		fmt.Sprintf("规范化名称: %s", normalized),
		fmt.Sprintf("可用性评分: %d/100", score),
		fmt.Sprintf("结论: %s", verdict),
		fmt.Sprintf("检查结果: 可用 %d 项，被占用 %d 项，未知 %d 项", available, unavailable, unknown),
	}
	if len(availableDomains) > 0 {
		lines = append(lines, "可优先考虑的域名: "+strings.Join(availableDomains, ", "))
	}
	if len(unavailableDomains) > 0 {
		lines = append(lines, "已出现占用信号的域名: "+strings.Join(unavailableDomains, ", "))
	}
	lines = append(lines, "注意: 本工具只做可用性预筛，不构成商标、法律或注册保证。")
	return strings.Join(lines, "\n")
}

func buildManualSearchLinks(normalized string) []PlatformCheck {
	queryText := url.QueryEscape(normalized)
	return []PlatformCheck{
		{Platform: "USPTO", Target: normalized, Status: "未知", Reason: "美国商标需人工复查", URL: "https://tmsearch.uspto.gov/search/search-results?query=" + queryText},
		{Platform: "WIPO", Target: normalized, Status: "未知", Reason: "国际商标需人工复查", URL: "https://branddb.wipo.int/en/quicksearch?by=brandName&v=" + queryText},
		{Platform: "Google", Target: normalized, Status: "未知", Reason: "搜索同名产品、公司、负面含义", URL: "https://www.google.com/search?q=" + queryText},
		{Platform: "Product Hunt", Target: normalized, Status: "未知", Reason: "检查创业产品重名", URL: "https://www.producthunt.com/search?q=" + queryText},
		{Platform: "npm user", Target: normalized, Status: "未知", Reason: "npm 用户名需人工复查", URL: "https://www.npmjs.com/~" + normalized},
		{Platform: "npm org", Target: normalized, Status: "未知", Reason: "npm 组织名需人工复查", URL: "https://www.npmjs.com/org/" + normalized},
		{Platform: "X", Target: normalized, Status: "未知", Reason: "社媒 handle 建议人工复查", URL: "https://x.com/" + normalized},
		{Platform: "LinkedIn", Target: normalized, Status: "未知", Reason: "B2B 公司主页建议人工复查", URL: "https://www.linkedin.com/company/" + normalized},
	}
}

func writeBrandReport(ctx *app.Context, req *BrandAvailabilityReq, normalized string, score int, verdict string, checks []PlatformCheck, alternatives []string, manual []PlatformCheck, summary string) (string, string, error) {
	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeReportFileName(normalized) + "-brand-availability.md"
	outputPath := filepath.Join(outputDir, fileName)

	var b strings.Builder
	b.WriteString("# Brand Availability Report\n\n")
	b.WriteString(summary)
	b.WriteString("\n\n")
	if strings.TrimSpace(req.Scenario) != "" {
		b.WriteString("## Scenario\n\n")
		b.WriteString(strings.TrimSpace(req.Scenario))
		b.WriteString("\n\n")
	}
	b.WriteString("## Platform Checks\n\n")
	b.WriteString("| Platform | Target | Status | Reason | URL |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, item := range checks {
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n", escapeMarkdownCell(item.Platform), escapeMarkdownCell(item.Target), escapeMarkdownCell(item.Status), escapeMarkdownCell(item.Reason), escapeMarkdownCell(item.URL)))
	}
	b.WriteString("\n## Alternative Names\n\n")
	for _, item := range alternatives {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Manual Review\n\n")
	b.WriteString("| Platform | URL | Note |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, item := range manual {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", escapeMarkdownCell(item.Platform), escapeMarkdownCell(item.URL), escapeMarkdownCell(item.Reason)))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Generated by KageOS Brand Availability Checker. Score: %d/100, verdict: %s.\n", score, verdict))

	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		return "", "", fmt.Errorf("写入报告失败: %w", err)
	}
	return outputPath, fs.ResponseFiles([]string{outputPath}), nil
}

func sanitizeReportFileName(name string) string {
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	name = re.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "brand"
	}
	return name
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	return value
}

var BrandAvailabilityTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "品牌占用验证",
		Desc:         "输入一个创业项目或产品品牌名，快速检查常见域名和开发者平台是否已被占用，并生成可用性评分、替代名称和 Markdown 报告。适合命名前的第一轮预筛。",
		Tags:         []string{"品牌命名", "域名检查", "创业工具", "Micro SaaS", "可用性验证"},
		Request:      &BrandAvailabilityReq{},
		Response:     &BrandAvailabilityResp{},
		CreateTables: []interface{}{&BrandCheckHistory{}},
	},
}

var BrandCheckHistoryTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "品牌检查历史",
		Desc:         "查看历史品牌可用性检查记录，包括评分、结论、可用域名和报告文件。",
		Tags:         []string{"品牌命名", "检查历史", "域名检查"},
		Request:      &BrandCheckHistoryReq{},
		CreateTables: []interface{}{&BrandCheckHistory{}},
	},
	AutoCrudTable: &BrandCheckHistory{},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		ids := req.GetIds()
		if len(ids) == 0 {
			return nil, fmt.Errorf("请选择要删除的历史记录")
		}
		if err := ctx.GetGormDB().Delete(&BrandCheckHistory{}, ids).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.POST("brand_availability.form", BrandAvailabilityCheck, BrandAvailabilityTemplate)
	packageContext.GET("brand_check_history.table", BrandCheckHistoryList, BrandCheckHistoryTemplate)
}
