package skills

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

var skillRRGGBBPattern = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

func TestSkillOptionsColorsUseRRGGBB(t *testing.T) {
	walkSkillMarkdown(t, func(path, content string) {
		lines := strings.Split(content, "\n")
		for lineNo, line := range lines {
			for _, value := range extractSkillOptionsColorsValues(line) {
				if value == "" {
					t.Fatalf("%s:%d empty options_colors value in %q", path, lineNo+1, line)
				}
				for _, color := range strings.Split(value, ",") {
					color = strings.TrimSpace(color)
					if !skillRRGGBBPattern.MatchString(color) {
						t.Fatalf("%s:%d options_colors must be RRGGBB, got %q in %q", path, lineNo+1, color, line)
					}
				}
			}
		}
	})
}

func TestSkillMarkdownDoesNotContainGenerationArtifacts(t *testing.T) {
	forbidden := []string{
		`""""options_colors`,
		`=~ s`,
		`do { my`,
		`909399:`,
		`widget 909399`,
		`options_colors 支持预设颜色`,
		`options_colors 支持预设`,
	}
	walkSkillMarkdown(t, func(path, content string) {
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s contains forbidden skill artifact %q", path, pattern)
			}
		}
	})
}

func walkSkillMarkdown(t *testing.T, visit func(path, content string)) {
	t.Helper()
	err := fs.WalkDir(skillFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := skillFS.ReadFile(path)
		if err != nil {
			return err
		}
		visit(path, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk skill markdown: %v", err)
	}
}

func extractSkillOptionsColorsValues(line string) []string {
	const marker = "options_colors:"
	var values []string
	for {
		idx := strings.Index(line, marker)
		if idx < 0 {
			return values
		}
		line = line[idx+len(marker):]
		end := len(line)
		for i, r := range line {
			if r == ';' || r == '`' || r == '"' || r == '\'' || r == ' ' || r == '\t' || r == '\r' {
				end = i
				break
			}
		}
		values = append(values, line[:end])
		line = line[end:]
	}
}
