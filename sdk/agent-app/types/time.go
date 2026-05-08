package types

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const TimeLayout = "2006-01-02 15:04:05"

type Time time.Time

func ParseTime(value string) (Time, error) {
	t, err := parseTimeString(value)
	if err != nil {
		return Time{}, err
	}
	return Time(t), nil
}

func (t Time) Time() time.Time {
	return time.Time(t)
}

func (t Time) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t Time) String() string {
	if t.IsZero() {
		return ""
	}
	return time.Time(t).Format(TimeLayout)
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("%q", time.Time(t).Format(TimeLayout))), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if t == nil {
		return nil
	}

	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		*t = Time{}
		return nil
	}

	value = strings.Trim(value, `"`)
	if strings.TrimSpace(value) == "" {
		*t = Time{}
		return nil
	}

	parsed, err := parseTimeString(value)
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}

func (t Time) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Time) UnmarshalText(data []byte) error {
	if t == nil {
		return nil
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		*t = Time{}
		return nil
	}
	parsed, err := parseTimeString(value)
	if err != nil {
		return err
	}
	*t = Time(parsed)
	return nil
}

func (t *Time) Scan(value interface{}) error {
	if t == nil {
		return nil
	}

	if value == nil {
		*t = Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*t = Time(v)
		return nil
	case string:
		parsed, err := parseTimeString(v)
		if err != nil {
			return err
		}
		*t = Time(parsed)
		return nil
	case []byte:
		parsed, err := parseTimeString(string(v))
		if err != nil {
			return err
		}
		*t = Time(parsed)
		return nil
	default:
		nullTime := &sql.NullTime{}
		if err := nullTime.Scan(value); err != nil {
			return err
		}
		if !nullTime.Valid {
			*t = Time{}
			return nil
		}
		*t = Time(nullTime.Time)
		return nil
	}
}

func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

func parseTimeString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		TimeLayout,
		time.DateOnly,
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if layout == time.DateOnly {
			if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
				return parsed, nil
			}
			continue
		}
		if layout == TimeLayout {
			if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
				return parsed, nil
			}
			continue
		}
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.In(time.Local), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid datetime %q, expected %s", value, TimeLayout)
}
