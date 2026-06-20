package wechat_articles

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// 全局共享的 HTTP Client
var sharedClient *http.Client

func init() {
	jar, _ := cookiejar.New(nil)
	sharedClient = &http.Client{Jar: jar}

	req, _ := http.NewRequest("GET", "https://weixin.sogou.com/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	sharedClient.Do(req)

	packageContext.POST("search_articles.form", SearchArticlesAPI, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name: "搜索文章",
			Desc: `## 功能说明
搜索文章 用于处理搜索文章相关的一次性业务任务。

## 适用场景
- 适合一次性提交参数并立即获得处理结果。
- 适合文件转换、内容生成、批量处理、快捷登记或业务动作触发。
- 可作为工作台智能体可直接调用的工具能力复用。

## 使用说明
- 按表单字段填写输入参数，上传文件时使用平台返回的文件引用。
- 提交后查看返回结果、生成文件或结构化响应。
- 如果结果需要长期管理，应沉淀到对应表格或业务目录中。`,
			Request:  &SearchArticlesReq{},
			Response: &SearchArticlesResp{},
		},
	})
	packageContext.POST("read_article.form", ReadArticleAPI, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name: "读取文章",
			Desc: `## 功能说明
读取文章 用于处理读取文章相关的一次性业务任务。

## 适用场景
- 适合一次性提交参数并立即获得处理结果。
- 适合文件转换、内容生成、批量处理、快捷登记或业务动作触发。
- 可作为工作台智能体可直接调用的工具能力复用。

## 使用说明
- 按表单字段填写输入参数，上传文件时使用平台返回的文件引用。
- 提交后查看返回结果、生成文件或结构化响应。
- 如果结果需要长期管理，应沉淀到对应表格或业务目录中。`,
			Request:  &ReadArticleReq{},
			Response: &ReadArticleResp{},
		},
	})
}

// ==================== 搜索文章 ====================

type SearchArticlesReq struct {
	Keyword string `json:"keyword" widget:"name:关键词;type:input" validate:"required"`
	Limit   int    `json:"limit" widget:"name:数量;type:integer;min:0;max:20;render_default:5"`
}

type SearchArticlesResp struct {
	Results []Article `json:"results" widget:"name:结果;type:table"`
	Count   int       `json:"count" widget:"name:数量;type:integer"`
	Message string    `json:"message"`
}

type Article struct {
	Title       string `json:"title"`
	AccountName string `json:"account_name"`
	Summary     string `json:"summary"`
	WechatURL   string `json:"wechat_url"`
}

func SearchArticlesAPI(ctx *app.Context, resp response.Response) error {
	var req SearchArticlesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	results := searchWeixin(req.Keyword, req.Limit)
	return resp.Form(&SearchArticlesResp{
		Results: results,
		Count:   len(results),
		Message: fmt.Sprintf("搜索「%s」完成，返回 %d 条", req.Keyword, len(results)),
	}).Build()
}

// ==================== 读取文章 ====================

type ReadArticleReq struct {
	URL string `json:"url" widget:"name:链接;type:input" validate:"required"`
}

type ReadArticleResp struct {
	Title    string `json:"title"`
	Account  string `json:"account"`
	Markdown string `json:"markdown" widget:"name:Markdown;type:text_area"`
}

func ReadArticleAPI(ctx *app.Context, resp response.Response) error {
	var req ReadArticleReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	title, account, htmlContent := readArticleHTML(req.URL)
	markdown := htmlToMarkdown(htmlContent)
	return resp.Form(&ReadArticleResp{
		Title:    title,
		Account:  account,
		Markdown: markdown,
	}).Build()
}

// ==================== 核心功能函数 ====================

func searchWeixin(keyword string, limit int) []Article {
	results := []Article{}

	searchURL := fmt.Sprintf("https://weixin.sogou.com/weixin?type=2&query=%s&ie=utf8", url.QueryEscape(keyword))
	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Referer", "https://weixin.sogou.com/")

	r, err := sharedClient.Do(req)
	if err != nil {
		return results
	}
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

	doc.Find(".news-list li").Each(func(i int, sel *goquery.Selection) {
		if len(results) >= limit {
			return
		}

		item := Article{}

		// 标题
		if t := sel.Find(".txt-box h3 a"); t.Length() > 0 {
			item.Title = strings.TrimSpace(t.Text())
		}

		// 公众号
		if a := sel.Find(".s-p .all-time-y2"); a.Length() > 0 {
			item.AccountName = strings.TrimSpace(a.Text())
		}

		// 摘要
		if d := sel.Find(".txt-info"); d.Length() > 0 {
			item.Summary = strings.TrimSpace(d.Text())
			item.Summary = regexp.MustCompile(`document\.write\(timeConvert\('\d+'\)`).ReplaceAllString(item.Summary, "")
			item.Summary = strings.TrimSpace(item.Summary)
			if len(item.Summary) > 200 {
				item.Summary = item.Summary[:200] + "..."
			}
		}

		// 微信直链
		if l := sel.Find(".txt-box h3 a"); l.Length() > 0 {
			if href, ok := l.Attr("href"); ok {
				sogouURL := href
				if strings.HasPrefix(href, "/") {
					sogouURL = "https://weixin.sogou.com" + href
				}
				if wechatURL := getRealURL(sogouURL); wechatURL != "" {
					item.WechatURL = wechatURL
				}
			}
		}

		if item.Title != "" {
			results = append(results, item)
		}
	})

	return results
}

func getRealURL(sogouURL string) string {
	req, _ := http.NewRequest("GET", sogouURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://weixin.sogou.com/")

	r, err := sharedClient.Do(req)
	if err != nil {
		return ""
	}
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	html := string(body)

	matches := regexp.MustCompile(`url\s*\+=\s*'([^']*)'`).FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return ""
	}

	var parts []string
	for _, m := range matches {
		if len(m) > 1 {
			parts = append(parts, m[1])
		}
	}

	realURL := strings.Join(parts, "")
	realURL = strings.ReplaceAll(realURL, "@", "")

	if !strings.Contains(realURL, "mp.weixin.qq.com") {
		return ""
	}
	return realURL
}

func readArticleHTML(articleURL string) (title, account, htmlContent string) {
	if strings.Contains(articleURL, "weixin.sogou.com") {
		if realURL := getRealURL(articleURL); realURL != "" {
			articleURL = realURL
		}
	}

	req, _ := http.NewRequest("GET", articleURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	r, err := sharedClient.Do(req)
	if err != nil {
		return "访问失败", "", fmt.Sprintf("<p>访问失败: %s</p>", err.Error())
	}
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)
	doc, _ := goquery.NewDocumentFromReader(bytes.NewReader(body))

	if t := doc.Find("h1.rich_media_title"); t.Length() > 0 {
		title = strings.TrimSpace(t.Text())
	}

	if a := doc.Find("a.rich_media_account_name"); a.Length() > 0 {
		account = strings.TrimSpace(a.Text())
	} else if a := doc.Find("span#js_name"); a.Length() > 0 {
		account = strings.TrimSpace(a.Text())
	}

	if c := doc.Find("#js_content"); c.Length() > 0 {
		c.Find("script, style").Remove()
		htmlContent, _ = c.Html()
	} else if c := doc.Find("div.rich_media_content"); c.Length() > 0 {
		c.Find("script, style").Remove()
		htmlContent, _ = c.Html()
	}

	if htmlContent == "" {
		htmlContent = "<p>文章内容获取失败，可能需要登录或链接已失效</p>"
	}

	return title, account, htmlContent
}

// ==================== HTML 转 Markdown ====================

func htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	var result strings.Builder
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("body > *").Each(func(i int, sel *goquery.Selection) {
		result.WriteString(processElement(sel))
		result.WriteString("\n\n")
	})

	md := result.String()
	md = regexp.MustCompile(`\n{3,}`).ReplaceAllString(md, "\n\n")
	md = strings.Trim(md, "\n")
	return md
}

func processElement(sel *goquery.Selection) string {
	tag := goquery.NodeName(sel)

	switch tag {
	case "p":
		return processParagraph(sel)
	case "h1", "h2", "h3", "h4", "h5", "h6":
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			level := strings.TrimPrefix(tag, "h")
			return strings.Repeat("#", len(level)) + " " + text
		}
	case "img":
		src := getAttr(sel, "data-src")
		if src == "" {
			src = getAttr(sel, "src")
		}
		if src != "" && !strings.HasPrefix(src, "data:") {
			return fmt.Sprintf("![image](%s)", src)
		}
	case "blockquote":
		text := strings.TrimSpace(sel.Text())
		if text != "" {
			lines := strings.Split(text, "\n")
			var quoted []string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					quoted = append(quoted, "> "+strings.TrimSpace(line))
				}
			}
			return strings.Join(quoted, "\n")
		}
	case "section":
		var parts []string
		sel.Children().Each(func(_ int, child *goquery.Selection) {
			parts = append(parts, processElement(child))
		})
		if sel.Text() != "" && len(parts) == 0 {
			parts = append(parts, strings.TrimSpace(sel.Text()))
		}
		return strings.Join(parts, "\n\n")
	case "ul", "ol":
		return processList(sel, tag == "ol")
	case "div":
		var parts []string
		sel.Children().Each(func(_ int, child *goquery.Selection) {
			parts = append(parts, processElement(child))
		})
		return strings.Join(parts, "\n\n")
	}

	return strings.TrimSpace(sel.Text())
}

func processParagraph(sel *goquery.Selection) string {
	var buf bytes.Buffer

	sel.Contents().Each(func(_ int, node *goquery.Selection) {
		if goquery.NodeName(node) == "#text" {
			buf.WriteString(node.Text())
		} else if goquery.NodeName(node) == "img" {
			src := getAttr(node, "data-src")
			if src == "" {
				src = getAttr(node, "src")
			}
			if src != "" && !strings.HasPrefix(src, "data:") {
				buf.WriteString(fmt.Sprintf("\n![image](%s)\n", src))
			}
		} else {
			text := processInlineElement(node)
			buf.WriteString(text)
		}
	})

	result := strings.TrimSpace(buf.String())
	result = regexp.MustCompile(`[ \t]+`).ReplaceAllString(result, " ")
	result = regexp.MustCompile(`\n{2,}`).ReplaceAllString(result, "\n")
	return result
}

func processInlineElement(sel *goquery.Selection) string {
	tag := goquery.NodeName(sel)
	text := strings.TrimSpace(sel.Text())

	switch tag {
	case "strong", "b":
		if text != "" {
			return "**" + text + "**"
		}
	case "em", "i":
		if text != "" {
			return "*" + text + "*"
		}
	case "a":
		href := getAttr(sel, "href")
		if href != "" && text != "" {
			return fmt.Sprintf("[%s](%s)", text, href)
		}
	case "br":
		return "\n"
	case "img":
		src := getAttr(sel, "data-src")
		if src == "" {
			src = getAttr(sel, "src")
		}
		if src != "" && !strings.HasPrefix(src, "data:") {
			return fmt.Sprintf("![image](%s)", src)
		}
	}

	return text
}

func processList(sel *goquery.Selection, ordered bool) string {
	var result strings.Builder

	sel.Find("> li").Each(func(_ int, li *goquery.Selection) {
		text := strings.TrimSpace(li.Text())
		text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

		if ordered {
			result.WriteString(fmt.Sprintf("1. %s\n", text))
		} else {
			result.WriteString(fmt.Sprintf("- %s\n", text))
		}
	})

	return result.String()
}

func getAttr(sel *goquery.Selection, attr string) string {
	val, _ := sel.Attr(attr)
	return val
}
