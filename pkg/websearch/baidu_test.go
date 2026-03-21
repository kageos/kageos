package websearch

import (
	"strings"
	"testing"
)

func TestParseBaiduHTMLSample(t *testing.T) {
	const sample = `<!DOCTYPE html><html><body><div id="content_left">
<div class="result c-container new-pmd">
  <h3 class="t"><a href="https://www.baidu.com/link?url=mock" target="_blank">示例标题</a></h3>
  <div class="c-abstract c-abstract-hide">这是摘要内容，用于验证解析逻辑是否命中 c-abstract。</div>
</div>
</div></body></html>`
	res, err := parseBaiduHTML(strings.NewReader(sample), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("len = %d", len(res))
	}
	if res[0].Title != "示例标题" {
		t.Fatalf("title %q", res[0].Title)
	}
	if !strings.Contains(res[0].URL, "baidu.com/link") {
		t.Fatalf("url %q", res[0].URL)
	}
	if !strings.Contains(res[0].Snippet, "摘要") {
		t.Fatalf("snippet %q", res[0].Snippet)
	}
	if res[0].Source != "baidu" {
		t.Fatalf("source %q", res[0].Source)
	}
}

func TestResolveBaiduURL(t *testing.T) {
	if u := resolveBaiduURL("//example.com/x"); u != "https://example.com/x" {
		t.Fatalf("got %q", u)
	}
	if u := resolveBaiduURL("/s?wd=a"); u != "https://www.baidu.com/s?wd=a" {
		t.Fatalf("got %q", u)
	}
}
