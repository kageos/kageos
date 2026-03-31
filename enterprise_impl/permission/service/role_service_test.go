package service

import "testing"

func TestResolveUserAppFromResourcePath(t *testing.T) {
	user, app, err := resolveUserAppFromResourcePath("/luobei/demo/report", "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user != "luobei" || app != "demo" {
		t.Fatalf("unexpected parse result: user=%s app=%s", user, app)
	}
}

func TestResolveUserAppFromResourcePathRejectsMismatch(t *testing.T) {
	_, _, err := resolveUserAppFromResourcePath("/luobei/demo/report", "other", "demo")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
}
