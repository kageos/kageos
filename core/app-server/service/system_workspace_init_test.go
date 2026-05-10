package service

import "testing"

func TestSystemAppDefinitionsIncludeOpenAPI(t *testing.T) {
	defs := systemAppDefinitions()
	got := make(map[string]string, len(defs))
	for _, def := range defs {
		got[def.Code] = def.Name
	}

	for code, name := range map[string]string{
		"tools":   "官方工具",
		"openapi": "平台接口",
		"prompt":  "提示词",
	} {
		if got[code] != name {
			t.Fatalf("system app %q = %q, want %q", code, got[code], name)
		}
	}
	deprecatedCode := "off" + "icial"
	if _, ok := got[deprecatedCode]; ok {
		t.Fatal("system app definitions should not recreate deprecated workspace")
	}
}
