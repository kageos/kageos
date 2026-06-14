package service

import (
	"strings"
	"testing"
)

func TestPodmanRunBaseArgsDoNotInjectDockerHostGateway(t *testing.T) {
	t.Setenv("TZ", "Asia/Tokyo")

	args := podmanRunBaseArgs("runtime-1", "/host/work", "/app/work")
	joined := strings.Join(args, " ")

	for _, forbidden := range []string{"--add-host", "host-gateway"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("podman run args must not contain %q: %#v", forbidden, args)
		}
	}
	if !containsArg(args, "-v") || !containsArg(args, "/host/work:/app/work") {
		t.Fatalf("podman run args missing mount: %#v", args)
	}
	if !containsArg(args, "-e") || !containsArg(args, "TZ=Asia/Tokyo") {
		t.Fatalf("podman run args missing timezone env: %#v", args)
	}
}

func TestRuntimeTimezoneDefaultsToShanghai(t *testing.T) {
	t.Setenv("TZ", "")

	if got := runtimeTimezone(); got != "Asia/Shanghai" {
		t.Fatalf("runtimeTimezone() = %q, want Asia/Shanghai", got)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
