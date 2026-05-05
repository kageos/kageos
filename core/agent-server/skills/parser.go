package skills

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func ParseSkill(content string, path string) (*Skill, error) {
	meta, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var skillMeta SkillMeta
	if err := yaml.Unmarshal([]byte(meta), &skillMeta); err != nil {
		return nil, fmt.Errorf("%s: parse frontmatter: %w", path, err)
	}
	skillMeta.Path = path
	if strings.TrimSpace(skillMeta.ID) == "" {
		return nil, fmt.Errorf("%s: frontmatter id is required", path)
	}
	if strings.TrimSpace(skillMeta.Name) == "" {
		return nil, fmt.Errorf("%s: frontmatter name is required", path)
	}
	if strings.TrimSpace(skillMeta.Description) == "" {
		return nil, fmt.Errorf("%s: frontmatter description is required", path)
	}

	return &Skill{
		Meta: normalizeMeta(skillMeta),
		Body: strings.TrimSpace(body),
	}, nil
}

func splitFrontmatter(content string) (meta string, body string, err error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", fmt.Errorf("missing YAML frontmatter")
	}

	rest := strings.TrimPrefix(content, "---\n")
	parts := strings.SplitN(rest, "\n---\n", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unterminated YAML frontmatter")
	}
	return strings.TrimSpace(parts[0]), parts[1], nil
}

func normalizeMeta(meta SkillMeta) SkillMeta {
	meta.ID = strings.TrimSpace(meta.ID)
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	meta.Path = strings.TrimSpace(meta.Path)
	meta.Triggers = normalizeStringList(meta.Triggers)
	meta.Modes = normalizeStringList(meta.Modes)
	meta.RequiredDocs = normalizeStringList(meta.RequiredDocs)
	meta.RecommendedDemos = normalizeStringList(meta.RecommendedDemos)
	meta.Capabilities = normalizeStringList(meta.Capabilities)
	meta.AllowedTools = normalizeStringList(meta.AllowedTools)
	meta.Completion = normalizeStringList(meta.Completion)
	return meta
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
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
