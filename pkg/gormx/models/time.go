package models

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type Time time.Time

const ctLayout = "2006-01-02 15:04:05"

func (t *Time) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), `"`)
	nt, err := time.Parse(ctLayout, s)
	*t = Time(nt)
	return
}

func (t Time) GetUnix() int64 {
	return time.Time(t).Unix()
}

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t Time) String() string {
	return fmt.Sprintf("%q", time.Time(t).Format(ctLayout))
}

func (date *Time) Scan(value interface{}) (err error) {
	nullTime := &sql.NullTime{}
	err = nullTime.Scan(value)
	*date = Time(nullTime.Time)
	return
}

func (date Time) Value() (driver.Value, error) {
	ti := time.Time(date)
	y, m, d := ti.Date()
	h := ti.Hour()
	minute := ti.Minute()
	s := ti.Second()
	return time.Date(y, m, d, h, minute, s, 0, time.Time(date).Location()), nil
}
