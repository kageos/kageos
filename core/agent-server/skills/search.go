package skills

import (
	"sort"
	"strings"
)

const defaultSearchLimit = 10

func (r *Registry) Search(opts SearchOptions) []SearchResult {
	if r == nil {
		return nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > 50 {
		limit = 50
	}

	keywords := splitKeywords(opts.Keyword)
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))

	results := make([]SearchResult, 0, len(r.skills))
	for _, skill := range r.skills {
		if skill == nil {
			continue
		}
		if mode != "" && !metaSupportsMode(skill.Meta, mode) {
			continue
		}
		score := scoreSkill(skill.Meta, keywords)
		if len(keywords) > 0 && score <= 0 {
			continue
		}
		results = append(results, SearchResult{Meta: skill.Meta, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Meta.ID < results[j].Meta.ID
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results
}

func splitKeywords(keyword string) []string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	normalized := strings.NewReplacer("|", " ", ",", " ", "，", " ", "\n", " ", "\t", " ").Replace(keyword)
	parts := strings.Fields(normalized)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func metaSupportsMode(meta SkillMeta, mode string) bool {
	if len(meta.Modes) == 0 {
		return true
	}
	for _, item := range meta.Modes {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == mode || item == "all" {
			return true
		}
	}
	return false
}

func scoreSkill(meta SkillMeta, keywords []string) int {
	if len(keywords) == 0 {
		return 1
	}
	text := strings.ToLower(strings.Join([]string{
		meta.ID,
		meta.Name,
		meta.Description,
		strings.Join(meta.Triggers, " "),
		strings.Join(meta.Capabilities, " "),
		strings.Join(meta.Modes, " "),
	}, " "))

	score := 0
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		switch {
		case strings.Contains(strings.ToLower(meta.ID), keyword):
			score += 10
		case strings.Contains(strings.ToLower(meta.Name), keyword):
			score += 8
		case containsAnyLower(meta.Triggers, keyword):
			score += 6
		case keywordContainsAnyTrigger(meta.Triggers, keyword):
			score += 5
		case strings.Contains(strings.ToLower(meta.Description), keyword):
			score += 4
		case strings.Contains(text, keyword):
			score += 1
		}
	}
	return score
}

func containsAnyLower(items []string, keyword string) bool {
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), keyword) {
			return true
		}
	}
	return false
}

func keywordContainsAnyTrigger(items []string, keyword string) bool {
	if keyword == "" {
		return false
	}
	for _, item := range items {
		item = strings.ToLower(strings.TrimSpace(item))
		if item != "" && strings.Contains(keyword, item) {
			return true
		}
	}
	return false
}
