package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
)

type fakeFileInfo struct {
	mode os.FileMode
}

func (f fakeFileInfo) Name() string       { return "tool" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f fakeFileInfo) Sys() any           { return nil }

func TestCommandPathForOSFallsBackToMacOSInstallLocations(t *testing.T) {
	lookPath := func(string) (string, error) { return "", errors.New("not in PATH") }
	stat := func(path string) (os.FileInfo, error) {
		if path == "/opt/podman/bin/podman" {
			return fakeFileInfo{mode: 0o755}, nil
		}
		return nil, errors.New("not found")
	}

	got, err := commandPathForOS("podman", "darwin", lookPath, stat)
	if err != nil {
		t.Fatalf("commandPathForOS() error = %v", err)
	}
	if got != "/opt/podman/bin/podman" {
		t.Fatalf("commandPathForOS() = %q, want /opt/podman/bin/podman", got)
	}
}

func TestCommandPathForOSDoesNotUseMacOSFallbackOnLinux(t *testing.T) {
	wantErr := errors.New("not in PATH")
	statCalled := false
	_, err := commandPathForOS(
		"podman",
		"linux",
		func(string) (string, error) { return "", wantErr },
		func(string) (os.FileInfo, error) {
			statCalled = true
			return fakeFileInfo{mode: 0o755}, nil
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("commandPathForOS() error = %v, want %v", err, wantErr)
	}
	if statCalled {
		t.Fatal("Linux lookup must not inspect macOS fallback paths")
	}
}

func TestPodmanRunBaseArgsDoNotInjectDockerHostGateway(t *testing.T) {
	t.Setenv("TZ", "Asia/Tokyo")

	args := podmanRunBaseArgs("runtime-1", "/host/work", "/app/work", "")
	joined := strings.Join(args, " ")

	for _, forbidden := range []string{"--add-host", "host-gateway", "--network"} {
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

func TestPodmanRunBaseArgsHonorsNetworkMode(t *testing.T) {
	args := podmanRunBaseArgs("runtime-1", "/host/work", "/app/work", "host")
	if !containsArg(args, "--network") || !containsArg(args, "host") {
		t.Fatalf("podman run args missing host network mode: %#v", args)
	}
}

func TestRuntimeTimezoneDefaultsToShanghai(t *testing.T) {
	t.Setenv("TZ", "")

	if got := runtimeTimezone(); got != "Asia/Shanghai" {
		t.Fatalf("runtimeTimezone() = %q, want Asia/Shanghai", got)
	}
}

func TestPodmanArgsHonorsConfiguredSocket(t *testing.T) {
	service := NewPodmanService(&appconfig.ContainerServiceConfig{
		Socket: "unix:///tmp/podman.sock",
	})

	args := service.podmanArgs("ps", "-a")
	want := []string{"--url", "unix:///tmp/podman.sock", "ps", "-a"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("podmanArgs() = %#v, want %#v", args, want)
	}
}

func TestNewPodmanServiceHandlesNilConfig(t *testing.T) {
	service := NewPodmanService(nil)
	if service.GetConfig() == nil {
		t.Fatal("NewPodmanService(nil) must install an empty config")
	}
	if got := service.GetConfig().GetRuntime(); got != "podman" {
		t.Fatalf("runtime = %q, want podman", got)
	}

	args := service.podmanArgs("ps")
	if strings.Join(args, "\x00") != "ps" {
		t.Fatalf("podmanArgs() = %#v, want no --url for empty config", args)
	}
}

func TestPodmanServiceStopClearsRunningState(t *testing.T) {
	service := NewPodmanService(nil)
	service.connected = true

	if err := service.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if service.IsRunning() {
		t.Fatal("Stop() must clear running state")
	}
}

func TestEnvVarNamesRedactsValues(t *testing.T) {
	got := envVarNames([]string{
		"NATS_URL=nats://aos:secret@host.containers.internal:4222",
		"GATEWAY_URL=http://host.containers.internal:9090",
		"FLAG_ONLY",
		" =ignored",
	})
	want := []string{"NATS_URL", "GATEWAY_URL", "FLAG_ONLY"}

	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("envVarNames() = %#v, want %#v", got, want)
	}
	for _, name := range got {
		if strings.Contains(name, "secret") || strings.Contains(name, "://") {
			t.Fatalf("envVarNames() leaked value material: %#v", got)
		}
	}
}

func TestPodmanSecretMountSpecNeverContainsSecretData(t *testing.T) {
	secret := RuntimeSecret{
		Name:   "alice-demo-v9-nats",
		Target: "kageos-nats",
		Data:   []byte("nats://aos:super-secret@host.containers.internal:4222"),
	}

	got := podmanSecretMountSpec(secret)
	want := "source=alice-demo-v9-nats,target=kageos-nats,type=mount,mode=0400"
	if got != want {
		t.Fatalf("podmanSecretMountSpec() = %q, want %q", got, want)
	}
	for _, forbidden := range []string{"super-secret", "nats://", "aos:"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("podman secret mount spec leaked %q: %q", forbidden, got)
		}
	}
}

func TestSplitContainerNames(t *testing.T) {
	got := splitContainerNames("app-one, app-two,,")
	want := []string{"app-one", "app-two"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("splitContainerNames() = %#v, want %#v", got, want)
	}
}

func TestIsPodmanNotFoundOutput(t *testing.T) {
	if !isPodmanNotFoundOutput("Error: no such container: demo") {
		t.Fatal("expected no such container output to be treated as not found")
	}
	if !isPodmanNotFoundOutput(`Error: no secret with name or id "demo"`) {
		t.Fatal("expected missing secret output to be treated as not found")
	}
	if isPodmanNotFoundOutput("Error: permission denied") {
		t.Fatal("permission errors must not be treated as not found")
	}
}

func TestParsePodmanContainerListJSON(t *testing.T) {
	input := []byte(`[
		{"ID":"abc123","Names":["kageos-alice-demo-v1"],"State":"running"},
		{"Id":"def456","Names":"legacy-one, legacy-two","Status":"exited","Exited":true},
		{"Id":"ghi789","Names":["created-one"],"State":"created","Exited":false}
	]`)

	containers, err := parsePodmanContainerListJSON(input)
	if err != nil {
		t.Fatalf("parsePodmanContainerListJSON() error = %v", err)
	}
	if len(containers) != 3 {
		t.Fatalf("container count = %d, want 3", len(containers))
	}
	if containers[0].ID != "abc123" || containers[0].Exited || containers[0].Names[0] != "kageos-alice-demo-v1" {
		t.Fatalf("first container parsed incorrectly: %#v", containers[0])
	}
	if containers[1].ID != "def456" || !containers[1].Exited || strings.Join(containers[1].Names, ",") != "legacy-one,legacy-two" {
		t.Fatalf("second container parsed incorrectly: %#v", containers[1])
	}
	if containers[2].ID != "ghi789" || !containers[2].Exited {
		t.Fatalf("created container must not be treated as running: %#v", containers[2])
	}
}

func TestParsePodmanImageListJSON(t *testing.T) {
	input := []byte(`[
		{"ID":"sha256:abc","Repository":"kageos/app-runtime","Tag":"latest"},
		{"Id":"sha256:def","repository":"busybox","tag":"1.36"},
		{"Id":"sha256:ghi","Names":["localhost/kagebase:latest"]}
	]`)

	images, err := parsePodmanImageListJSON(input)
	if err != nil {
		t.Fatalf("parsePodmanImageListJSON() error = %v", err)
	}
	if len(images) != 3 {
		t.Fatalf("image count = %d, want 3", len(images))
	}
	if images[0].ID != "sha256:abc" || images[0].Repository != "kageos/app-runtime" || images[0].Tag != "latest" {
		t.Fatalf("first image parsed incorrectly: %#v", images[0])
	}
	if images[1].ID != "sha256:def" || images[1].Repository != "busybox" || images[1].Tag != "1.36" {
		t.Fatalf("second image parsed incorrectly: %#v", images[1])
	}
	if images[2].ID != "sha256:ghi" || images[2].Repository != "localhost/kagebase" || images[2].Tag != "latest" {
		t.Fatalf("third image parsed incorrectly: %#v", images[2])
	}
}

func TestParsePodmanInspectRunningJSON(t *testing.T) {
	running, err := parsePodmanInspectRunningJSON([]byte(`[{"State":{"Running":true}}]`))
	if err != nil {
		t.Fatalf("parsePodmanInspectRunningJSON() error = %v", err)
	}
	if !running {
		t.Fatal("expected running container")
	}

	running, err = parsePodmanInspectRunningJSON([]byte(`{"State":{"Running":"false"}}`))
	if err != nil {
		t.Fatalf("parsePodmanInspectRunningJSON() object error = %v", err)
	}
	if running {
		t.Fatal("expected stopped container")
	}
}

func TestPodmanCLIJSONOutputParsesWhenAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	requirePodmanAvailable(t, ctx)

	psOutput := runPodmanForTest(t, ctx, "ps", "-a", "--format", "json")
	containers, err := parsePodmanContainerListJSON(psOutput)
	if err != nil {
		t.Fatalf("parse real podman ps JSON: %v", err)
	}
	for _, container := range containers {
		if container.ID == "" {
			t.Fatalf("parsed container without ID: %#v", container)
		}
	}

	imagesOutput := runPodmanForTest(t, ctx, "images", "--format", "json")
	images, err := parsePodmanImageListJSON(imagesOutput)
	if err != nil {
		t.Fatalf("parse real podman images JSON: %v", err)
	}
	for _, image := range images {
		if image.ID == "" {
			t.Fatalf("parsed image without ID: %#v", image)
		}
	}

	for _, container := range containers {
		if len(container.Names) == 0 {
			continue
		}
		inspectOutput := runPodmanForTest(t, ctx, "container", "inspect", "--format", "json", container.Names[0])
		if _, err := parsePodmanInspectRunningJSON(inspectOutput); err != nil {
			t.Fatalf("parse real podman inspect JSON: %v", err)
		}
		return
	}
}

func TestPodmanServiceReadOnlyMethodsWhenAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	requirePodmanAvailable(t, ctx)

	service := NewPodmanService(&appconfig.ContainerServiceConfig{})
	service.connected = true

	containers, err := service.ListContainers(ctx)
	if err != nil {
		t.Fatalf("ListContainers() error = %v", err)
	}
	for _, container := range containers {
		if container.ID == "" {
			t.Fatalf("ListContainers() returned container without ID: %#v", container)
		}
		if container.State != "" && (container.State == "running") != !container.Exited {
			t.Fatalf("ListContainers() running state mismatch: %#v", container)
		}
	}

	images, err := service.ListImages(ctx)
	if err != nil {
		t.Fatalf("ListImages() error = %v", err)
	}
	for _, image := range images {
		if image.ID == "" {
			t.Fatalf("ListImages() returned image without ID: %#v", image)
		}
	}

	missingName := "kageos-test-missing-" + time.Now().UTC().Format("20060102150405")
	running, err := service.IsContainerRunning(ctx, missingName)
	if err != nil {
		t.Fatalf("IsContainerRunning() missing container error = %v", err)
	}
	if running {
		t.Fatalf("IsContainerRunning() missing container = true for %q", missingName)
	}

	for _, container := range containers {
		if len(container.Names) == 0 {
			continue
		}
		running, err := service.IsContainerRunning(ctx, container.Names[0])
		if err != nil {
			t.Fatalf("IsContainerRunning(%q) error = %v", container.Names[0], err)
		}
		if running != !container.Exited {
			t.Fatalf("IsContainerRunning(%q) = %v, ListContainers running = %v, container = %#v", container.Names[0], running, !container.Exited, container)
		}
		return
	}
}

func requirePodmanAvailable(t *testing.T, ctx context.Context) {
	t.Helper()

	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman CLI is not installed")
	}

	cmd := exec.CommandContext(ctx, "podman", "info", "--format", "json")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("podman CLI is installed but not available: %v, output: %s", err, string(output))
	}
}

func runPodmanForTest(t *testing.T, ctx context.Context, args ...string) []byte {
	t.Helper()

	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("podman %s failed: %v, output: %s", strings.Join(args, " "), err, string(output))
	}
	return output
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
