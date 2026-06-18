package naming

import (
	"strings"
	"testing"
)

func TestDeriveGoPackageNameUsesPinyinForChineseLabels(t *testing.T) {
	got := DeriveGoPackageName("用户管理", "directory")
	if got != "yong_hu_guan_li" {
		t.Fatalf("DeriveGoPackageName() = %q, want %q", got, "yong_hu_guan_li")
	}
}

func TestDeriveGoPackageNameKeepsASCIIWordsAndNumbers(t *testing.T) {
	got := DeriveGoPackageName("CRM客户2.0", "directory")
	if got != "crm_ke_hu_2_0" {
		t.Fatalf("DeriveGoPackageName() = %q, want %q", got, "crm_ke_hu_2_0")
	}
}

func TestDeriveGoPackageNameFallsBackWhenNeeded(t *testing.T) {
	got := DeriveGoPackageName("2026", "directory")
	if got != "directory_2026" {
		t.Fatalf("DeriveGoPackageName() = %q, want %q", got, "directory_2026")
	}

	got = DeriveGoPackageName("!!!", "directory")
	if got != "directory" {
		t.Fatalf("DeriveGoPackageName() = %q, want %q", got, "directory")
	}
}

func TestGoPackageNameWithNumericSuffixPreservesMaxLength(t *testing.T) {
	base := strings.Repeat("a", MaxGoPackageNameLength)
	got := GoPackageNameWithNumericSuffix(base, 12)
	if len(got) != MaxGoPackageNameLength {
		t.Fatalf("len() = %d, want %d", len(got), MaxGoPackageNameLength)
	}
	if !strings.HasSuffix(got, "_12") {
		t.Fatalf("suffix = %q, want _12", got)
	}
}
