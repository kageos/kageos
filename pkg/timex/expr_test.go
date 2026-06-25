package timex

import (
	"regexp"
	"testing"
)

func TestResolveDateTimeExpr(t *testing.T) {
	got, ok := ResolveDateTimeExpr("CURRENT_DATE")
	if !ok {
		t.Fatal("expected CURRENT_DATE to resolve")
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} 00:00:00$`).MatchString(got) {
		t.Fatalf("unexpected CURRENT_DATE format: %q", got)
	}

	got, ok = ResolveDateTimeExpr("CURRENT_TIMESTAMP")
	if !ok {
		t.Fatal("expected CURRENT_TIMESTAMP to resolve")
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(got) {
		t.Fatalf("unexpected CURRENT_TIMESTAMP format: %q", got)
	}
}

func TestResolveDateTimeExprSupportsSQLStyleWhitelist(t *testing.T) {
	got, ok := ResolveDateTimeExpr("DATE_ADD(CURRENT_DATE, INTERVAL 1 DAY)")
	if !ok {
		t.Fatal("expected DATE_ADD to resolve")
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} 00:00:00$`).MatchString(got) {
		t.Fatalf("unexpected DATE_ADD format: %q", got)
	}

	if got, ok := ResolveDateTimeExpr("DATE_FORMAT(created_at, '%Y')"); ok {
		t.Fatalf("arbitrary SQL should not resolve, got %q", got)
	}
}

func TestReplaceTimeExprsInParamValueResolvesDirectSQLFunction(t *testing.T) {
	got := ReplaceTimeExprsInParamValue("DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 7 DAY)")
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`).MatchString(got) {
		t.Fatalf("unexpected output: %q", got)
	}
}
