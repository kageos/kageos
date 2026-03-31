package service

import "testing"

func TestResolveUserAppFromResourcePath(t *testing.T) {
	user, app, err := resolveUserAppFromResourcePath("/luobei/demo", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user != "luobei" || app != "demo" {
		t.Fatalf("unexpected result: user=%s app=%s", user, app)
	}
}

func TestResolveUserAppFromResourcePathRejectsMismatch(t *testing.T) {
	_, _, err := resolveUserAppFromResourcePath("/luobei/demo", "other", "demo")
	if err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestResolveUserAppFromResourcePathFallsBackToUserApp(t *testing.T) {
	user, app, err := resolveUserAppFromResourcePath("", "luobei", "demo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user != "luobei" || app != "demo" {
		t.Fatalf("unexpected result: user=%s app=%s", user, app)
	}
}
