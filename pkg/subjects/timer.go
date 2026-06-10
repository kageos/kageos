package subjects

import (
	"strings"
	"unicode"
)

const (
	TimerExecutionFinishedSubject = "timer.v1.event.execution.finished"
)

func TimerExecutionRequestedSubject(executorKey string) string {
	return "timer.v1.cmd.execution.requested." + NormalizeTimerSubjectSuffix(executorKey)
}

func TimerWorkerQueueGroup(executorKey string) string {
	return "timer.worker." + NormalizeTimerSubjectSuffix(executorKey)
}

func NormalizeTimerSubjectSuffix(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), ".-_")
	if out == "" {
		return "unknown"
	}
	return out
}
