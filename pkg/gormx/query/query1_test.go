package query

import "testing"

func TestParseFieldValuesAllowsDatetimeColon(t *testing.T) {
	got, err := parseFieldValues("created_at:2026-04-21 16:30:05,status:处理中")
	if err != nil {
		t.Fatalf("parseFieldValues returned error: %v", err)
	}

	if got["created_at"] != "2026-04-21 16:30:05" {
		t.Fatalf("created_at = %q", got["created_at"])
	}
	if got["status"] != "处理中" {
		t.Fatalf("status = %q", got["status"])
	}
}

func TestParseFieldValuesKeepsSQLFunctionCommaTogether(t *testing.T) {
	got, err := parseFieldValues("created_at:DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 7 DAY),status:处理中")
	if err != nil {
		t.Fatalf("parseFieldValues returned error: %v", err)
	}

	if got["created_at"] != "DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 7 DAY)" {
		t.Fatalf("created_at = %q", got["created_at"])
	}
	if got["status"] != "处理中" {
		t.Fatalf("status = %q", got["status"])
	}
}
