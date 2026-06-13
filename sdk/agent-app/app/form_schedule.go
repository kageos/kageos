package app

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormSchedule describes a default scheduled execution owned by a Form.
// It is declarative metadata only; timer-scheduler owns runtime state.
type FormSchedule struct {
	Code         string      `json:"code"`
	Title        string      `json:"title,omitempty"`
	Description  string      `json:"description,omitempty"`
	EverySeconds int64       `json:"every_seconds,omitempty"`
	CronExpr     string      `json:"cron_expr,omitempty"`
	Timezone     string      `json:"timezone,omitempty"`
	MaxRuns      int         `json:"max_runs,omitempty"`
	Body         interface{} `json:"body,omitempty"`
}

type CompiledFormSchedule struct {
	Code         string          `json:"code"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	EverySeconds int64           `json:"every_seconds,omitempty"`
	CronExpr     string          `json:"cron_expr,omitempty"`
	Timezone     string          `json:"timezone,omitempty"`
	MaxRuns      int             `json:"max_runs,omitempty"`
	Body         json.RawMessage `json:"body,omitempty"`
}

func compileFormSchedules(route string, schedules []FormSchedule) ([]CompiledFormSchedule, error) {
	if len(schedules) == 0 {
		return nil, nil
	}
	out := make([]CompiledFormSchedule, 0, len(schedules))
	seen := map[string]struct{}{}
	for i, schedule := range schedules {
		compiled, err := compileFormSchedule(route, i, schedule)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[compiled.Code]; exists {
			return nil, fmt.Errorf("%s form schedule code %q is duplicated", route, compiled.Code)
		}
		seen[compiled.Code] = struct{}{}
		out = append(out, compiled)
	}
	return out, nil
}

func compileFormSchedule(route string, index int, schedule FormSchedule) (CompiledFormSchedule, error) {
	code := strings.TrimSpace(schedule.Code)
	if code == "" {
		return CompiledFormSchedule{}, fmt.Errorf("%s form schedule #%d code is required", route, index+1)
	}
	cronExpr := strings.TrimSpace(schedule.CronExpr)
	hasCron := cronExpr != ""
	hasEvery := schedule.EverySeconds > 0
	if hasCron == hasEvery {
		return CompiledFormSchedule{}, fmt.Errorf("%s form schedule %q must set exactly one of cron_expr or every_seconds", route, code)
	}
	body, err := compileFormScheduleBody(schedule.Body)
	if err != nil {
		return CompiledFormSchedule{}, fmt.Errorf("%s form schedule %q body invalid: %w", route, code, err)
	}
	return CompiledFormSchedule{
		Code:         code,
		Title:        strings.TrimSpace(schedule.Title),
		Description:  strings.TrimSpace(schedule.Description),
		EverySeconds: schedule.EverySeconds,
		CronExpr:     cronExpr,
		Timezone:     strings.TrimSpace(schedule.Timezone),
		MaxRuns:      schedule.MaxRuns,
		Body:         body,
	}, nil
}

func compileFormScheduleBody(body interface{}) (json.RawMessage, error) {
	if body == nil {
		return json.RawMessage(`{}`), nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 || string(data) == "null" {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("must be valid JSON")
	}
	if data[0] != '{' {
		return nil, fmt.Errorf("must be a JSON object")
	}
	return json.RawMessage(data), nil
}
