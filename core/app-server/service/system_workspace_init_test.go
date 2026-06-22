package service

import "testing"

func TestSystemAppDefinitionsOnlyCreateRequiredSystemApps(t *testing.T) {
	defs := systemAppDefinitions()
	got := make(map[string]string, len(defs))
	for _, def := range defs {
		got[def.Code] = def.Name
	}

	for code, name := range map[string]string{
		"tools": "官方工具",
	} {
		if got[code] != name {
			t.Fatalf("system app %q = %q, want %q", code, got[code], name)
		}
	}
	if _, ok := got["openapi"]; ok {
		t.Fatal("system app definitions should not create openapi workspace")
	}
	if _, ok := got["prompt"]; ok {
		t.Fatal("system app definitions should not create prompt workspace")
	}
	deprecatedCode := "off" + "icial"
	if _, ok := got[deprecatedCode]; ok {
		t.Fatal("system app definitions should not recreate deprecated workspace")
	}
}
