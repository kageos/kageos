package scheduledsdk

import (
	"fmt"
	"strings"
	"time"
)

type ScheduleType string

const (
	ScheduleAt    ScheduleType = "atime"
	ScheduleCron  ScheduleType = "cron"
	ScheduleEvery ScheduleType = "every"
)

type Schedule struct {
	Type            ScheduleType `json:"type"`
	RunAt           time.Time    `json:"run_at,omitempty"`
	CronExpr        string       `json:"cron_expr,omitempty"`
	IntervalSeconds int64        `json:"interval_seconds,omitempty"`
	MaxRuns         int          `json:"max_runs,omitempty"`
	Timezone        string       `json:"timezone,omitempty"`
}

func At(t time.Time) Schedule {
	return Schedule{Type: ScheduleAt, RunAt: t}
}

func Cron(expr string) Schedule {
	return Schedule{Type: ScheduleCron, CronExpr: strings.TrimSpace(expr)}
}

func Every(seconds int64) Schedule {
	return Schedule{Type: ScheduleEvery, IntervalSeconds: seconds}
}

func (s Schedule) Validate() error {
	switch s.Type {
	case ScheduleAt:
		if s.RunAt.IsZero() {
			return fmt.Errorf("scheduledsdk: run_at is required for atime schedule")
		}
	case ScheduleCron:
		if strings.TrimSpace(s.CronExpr) == "" {
			return fmt.Errorf("scheduledsdk: cron_expr is required for cron schedule")
		}
	case ScheduleEvery:
		if s.IntervalSeconds <= 0 {
			return fmt.Errorf("scheduledsdk: interval_seconds must be positive for every schedule")
		}
	default:
		return fmt.Errorf("scheduledsdk: unsupported schedule type %q", s.Type)
	}
	return nil
}
