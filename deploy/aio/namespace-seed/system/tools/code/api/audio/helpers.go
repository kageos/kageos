package audio

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func ensureFFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("未找到 ffmpeg，请确认运行环境已安装 FFmpeg")
	}
	return nil
}

func ensureFFprobe() error {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("未找到 ffprobe，请确认运行环境已安装 FFmpeg")
	}
	return nil
}

func outputFileName(originalName, localPath, suffix, ext string, seen map[string]int) string {
	name := strings.TrimSpace(originalName)
	if name == "" {
		name = filepath.Base(localPath)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = sanitizeFileName(base, "audio")
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if ext == "" {
		ext = "mp3"
	}
	candidate := base + suffix + "." + ext
	if seen == nil {
		return candidate
	}
	if _, ok := seen[candidate]; !ok {
		seen[candidate] = 1
		return candidate
	}
	baseWithSuffix := base + suffix
	for {
		seen[candidate]++
		next := fmt.Sprintf("%s_%d.%s", baseWithSuffix, seen[candidate], ext)
		if _, ok := seen[next]; !ok {
			seen[next] = 1
			return next
		}
	}
}

func sanitizeFileName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	return name
}

func normalizeAudioFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "m4a", "wav", "ogg", "flac":
		return strings.ToLower(strings.TrimSpace(format))
	default:
		return "mp3"
	}
}

func audioCodecArgs(format, quality string) []string {
	bitrate := audioBitrate(quality)
	switch normalizeAudioFormat(format) {
	case "m4a":
		return []string{"-c:a", "aac", "-b:a", bitrate}
	case "wav":
		return []string{"-c:a", "pcm_s16le"}
	case "ogg":
		return []string{"-c:a", "libopus", "-b:a", bitrate}
	case "flac":
		return []string{"-c:a", "flac"}
	default:
		return []string{"-c:a", "libmp3lame", "-b:a", bitrate}
	}
}

func audioBitrate(quality string) string {
	switch strings.TrimSpace(quality) {
	case "高质量":
		return "320k"
	case "低体积":
		return "96k"
	default:
		return "192k"
	}
}

func formatSeconds(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	value := strconv.FormatFloat(seconds, 'f', 3, 64)
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	if value == "" {
		return "0"
	}
	return value
}
