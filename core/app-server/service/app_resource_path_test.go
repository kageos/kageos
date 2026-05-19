package service

import "testing"

func TestResolveUserAppFromResourcePath(t *testing.T) {
	user, app, err := resolveUserAppFromResourcePath("/luobei/demo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user != "luobei" || app != "demo" {
		t.Fatalf("unexpected result: user=%s app=%s", user, app)
	}
}

func TestResolveUserAppFromResourcePathRejectsInvalidPath(t *testing.T) {
	_, _, err := resolveUserAppFromResourcePath("/luobei")
	if err == nil {
		t.Fatal("expected invalid resource_path error")
	}
}

func TestResolveUserAppFromRequiredResourcePathRequiresResourcePath(t *testing.T) {
	_, _, err := resolveUserAppFromRequiredResourcePath("")
	if err == nil {
		t.Fatal("expected missing resource_path error")
	}
}
