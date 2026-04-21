package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeJSON(t *testing.T) {
	parsed, err := ParseTime("2026-04-21 16:30:05")
	if err != nil {
		t.Fatalf("ParseTime returned error: %v", err)
	}

	data, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	if string(data) != `"2026-04-21 16:30:05"` {
		t.Fatalf("unexpected json: %s", data)
	}

	var decoded Time
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if decoded.String() != "2026-04-21 16:30:05" {
		t.Fatalf("unexpected decoded time: %s", decoded.String())
	}
}

func TestTimeValueUsesNativeTime(t *testing.T) {
	parsed, err := ParseTime("2026-04-21 16:30:05")
	if err != nil {
		t.Fatalf("ParseTime returned error: %v", err)
	}

	value, err := parsed.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	if _, ok := value.(time.Time); !ok {
		t.Fatalf("expected driver value to be time.Time, got %T", value)
	}
}

func TestTimeScanStringAndNull(t *testing.T) {
	var scanned Time
	if err := scanned.Scan("2026-04-21 16:30:05"); err != nil {
		t.Fatalf("Scan string returned error: %v", err)
	}
	if scanned.String() != "2026-04-21 16:30:05" {
		t.Fatalf("unexpected scanned time: %s", scanned.String())
	}

	if err := scanned.Scan(nil); err != nil {
		t.Fatalf("Scan nil returned error: %v", err)
	}
	if !scanned.IsZero() {
		t.Fatalf("expected zero time after nil scan")
	}
}

func TestTimeRejectsNumericValue(t *testing.T) {
	var scanned Time
	if err := scanned.Scan(int64(1776789005000)); err == nil {
		t.Fatalf("expected numeric scan to fail")
	}

	if _, err := ParseTime("1776789005000"); err == nil {
		t.Fatalf("expected numeric string parse to fail")
	}
}

func TestTimeUnmarshalText(t *testing.T) {
	var parsed Time
	if err := parsed.UnmarshalText([]byte("2026-04-21 16:30:05")); err != nil {
		t.Fatalf("UnmarshalText returned error: %v", err)
	}
	if parsed.String() != "2026-04-21 16:30:05" {
		t.Fatalf("unexpected parsed time: %s", parsed.String())
	}
}
