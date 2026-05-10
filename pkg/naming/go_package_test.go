package naming

import (
	"strings"
	"testing"
)

func TestValidateGoPackageNameAcceptsValidNames(t *testing.T) {
	t.Parallel()

	validNames := []string{"a", "user", "user_v2", "ticket123"}
	for _, name := range validNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateGoPackageName(name, "目录英文标识"); err != nil {
				t.Fatalf("expected %q to be valid, got %v", name, err)
			}
		})
	}
}

func TestValidateGoPackageNameRejectsInvalidNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		wantErr string
	}{
		{name: "hyphen", code: "user-center", wantErr: "不能使用横线"},
		{name: "starts with digit", code: "1user", wantErr: "需以小写英文字母开头"},
		{name: "keyword", code: "type", wantErr: "已被系统占用"},
		{name: "empty", code: "", wantErr: "不能为空"},
		{name: "outer space", code: " user", wantErr: "首尾空格"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateGoPackageName(tt.code, "目录英文标识")
			if err == nil {
				t.Fatalf("expected %q to be invalid", tt.code)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error to contain %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateGoPackageNameLengthUsesMinimum(t *testing.T) {
	t.Parallel()

	err := ValidateGoPackageNameLength("a", "工作空间英文标识", 2, MaxGoPackageNameLength)
	if err == nil {
		t.Fatal("expected short workspace code to be rejected")
	}
	if !strings.Contains(err.Error(), "2-50") {
		t.Fatalf("unexpected error: %v", err)
	}
}
