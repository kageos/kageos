package sdkmodule

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestFallbackVersionMatchesGoMod(t *testing.T) {
	goModPath := filepath.Join("..", "..", "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("read %s: %v", goModPath, err)
	}

	parsed, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		t.Fatalf("parse %s: %v", goModPath, err)
	}

	for _, requirement := range parsed.Require {
		if requirement.Mod.Path != ModulePath {
			continue
		}
		if requirement.Mod.Version != Version {
			t.Fatalf("fallback SDK version %s does not match go.mod requirement %s", Version, requirement.Mod.Version)
		}
		return
	}

	t.Fatalf("%s is not required by go.mod", ModulePath)
}
