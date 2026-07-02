package service

import (
	"context"
	"path"
	"strings"
	"unicode"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

func searchPromptDocMatches(ctx context.Context, fullCodePath string, keyword string, limit int) []searchDocMatch {
	docPath, name, content := resolvePromptDocSearchContent(ctx, fullCodePath)
	if docPath == "" {
		return nil
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	matches := make([]searchDocMatch, 0, limit)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !searchTextMatchesQuery(line, keyword) {
			continue
		}
		matches = append(matches, searchDocMatch{
			Name:         firstNonEmptyString(name, docPath),
			FullCodePath: docPath,
			Line:         i + 1,
			Snippet:      compactSearchSnippet(line, 240),
		})
		if len(matches) >= limit {
			break
		}
	}
	if len(matches) == 0 && strings.TrimSpace(keyword) == "" {
		matches = append(matches, searchDocMatch{
			Name:         firstNonEmptyString(name, docPath),
			FullCodePath: docPath,
			Line:         1,
			Snippet:      "这是内置文档/案例路径；需要完整内容时调用 read_doc(directory=\"" + docPath + "\")。",
		})
	}
	return matches
}

func resolvePromptDocSearchContent(ctx context.Context, fullCodePath string) (docPath string, name string, content string) {
	docPath = prompt.NormalizePromptDocPath(fullCodePath)
	if docPath == "" || !prompt.IsPromptDocPath(docPath) {
		return "", "", ""
	}
	for {
		name, content = prompt.GetPromptDocContent(ctx, docPath)
		if strings.TrimSpace(content) != "" {
			return docPath, name, content
		}
		if docPath == prompt.SystemPromptRootPath {
			return "", "", ""
		}
		parent := path.Dir(docPath)
		if parent == "." || parent == "/" || parent == docPath || !strings.HasPrefix(parent, prompt.SystemPromptRootPath) {
			return "", "", ""
		}
		docPath = parent
	}
}

func searchTextMatchesQuery(text string, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	lowerText := strings.ToLower(text)
	for _, part := range splitSearchKeywords(query) {
		lowerPart := strings.ToLower(strings.TrimSpace(part))
		if lowerPart == "" {
			continue
		}
		if strings.Contains(lowerText, lowerPart) {
			return true
		}
		tokens := searchContentTokens(lowerPart)
		if len(tokens) == 0 {
			continue
		}
		allMatched := true
		for _, token := range tokens {
			if !strings.Contains(lowerText, token) {
				allMatched = false
				break
			}
		}
		if allMatched {
			return true
		}
	}
	return false
}

func searchContentTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
}

func compactSearchSnippet(line string, limit int) string {
	line = strings.TrimSpace(line)
	if limit <= 0 || len([]rune(line)) <= limit {
		return line
	}
	runes := []rune(line)
	return string(runes[:limit]) + "..."
}
