package video

import "strings"

func escapeFFmpegDrawtextText(text string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`:`, `\:`,
		`'`, `\'`,
		`%`, `\%`,
		"\r", " ",
		"\n", " ",
	).Replace(text)
}
