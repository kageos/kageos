package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func detectComposeCommand() ([]string, error) {
	if forced := strings.TrimSpace(os.Getenv(composeEngineEnv)); forced != "" {
		return detectComposeCommandForEngine(forced)
	}
	if compose, err := detectComposeCommandForEngine("podman"); err == nil {
		return compose, nil
	}
	if compose, err := detectComposeCommandForEngine("docker"); err == nil {
		return compose, nil
	}
	return nil, fmt.Errorf("podman compose or docker compose is required; set %s=podman or %s=docker to force one engine", composeEngineEnv, composeEngineEnv)
}

func detectComposeCommandForEngine(engine string) ([]string, error) {
	engine = strings.TrimSpace(engine)
	switch engine {
	case "podman", "docker":
	default:
		return nil, fmt.Errorf("%s must be podman or docker, got %q", composeEngineEnv, engine)
	}
	if _, err := execLookPath(engine); err != nil {
		return nil, fmt.Errorf("%s not found: %w", engine, err)
	}
	if err := execRun(engine, "compose", "version"); err != nil {
		return nil, fmt.Errorf("%s compose is not available: %w", engine, err)
	}
	return []string{engine, "compose"}, nil
}

func checkComposeCommand() error {
	_, err := detectComposeCommand()
	return err
}

func checkDevComposeCommand(engine string) error {
	engine = normalizeDevEngine(engine)
	if engine == "auto" {
		return checkComposeCommand()
	}
	_, err := detectComposeCommandForEngine(engine)
	return err
}

func checkComposeRuntime() error {
	compose, err := detectComposeCommand()
	if err != nil {
		return err
	}
	_, err = checkComposeRuntimeForCommand(compose)
	return err
}

func checkComposeRuntimeForCommand(compose []string) (string, error) {
	if len(compose) == 0 {
		return "", fmt.Errorf("compose command is empty")
	}
	engine := compose[0]
	args := append(append([]string{}, compose[1:]...), "ls")
	output, err := execOutput(engine, args...)
	if err == nil {
		return strings.Join(compose, " "), nil
	}
	if isComposeListUnsupported(output) {
		if infoErr := execRun(engine, "info"); infoErr == nil {
			return strings.Join(compose, " "), nil
		} else {
			return "", formatComposeRuntimeError(engine, output, infoErr)
		}
	}
	return "", formatComposeRuntimeError(engine, output, err)
}

func isComposeListUnsupported(output string) bool {
	normalized := strings.ToLower(output)
	return strings.Contains(normalized, "unknown command") ||
		strings.Contains(normalized, "no such command") ||
		strings.Contains(normalized, "unrecognized command") ||
		strings.Contains(normalized, "invalid choice")
}

func formatComposeRuntimeError(engine, output string, err error) error {
	message := strings.TrimSpace(output)
	if message != "" {
		return fmt.Errorf("compose runtime is not ready for %s: %w; output: %s; %s", engine, err, oneLine(message), composeRuntimeHint(engine))
	}
	return fmt.Errorf("compose runtime is not ready for %s: %w; %s", engine, err, composeRuntimeHint(engine))
}

func composeRuntimeHint(engine string) string {
	switch engine {
	case "podman":
		return fmt.Sprintf("for rootless Podman run `systemctl --user enable --now podman.socket` and `sudo loginctl enable-linger %s`; expected socket: %s; use `%s=docker` to force Docker", currentUserName(), podmanSocketPath(), composeEngineEnv)
	case "docker":
		if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); host != "" {
			return fmt.Sprintf("Docker is using DOCKER_HOST=%s; start that daemon/socket or use `%s=podman` to force Podman", host, composeEngineEnv)
		}
		return fmt.Sprintf("start the Docker daemon or use `%s=podman` to force Podman", composeEngineEnv)
	default:
		return fmt.Sprintf("check container engine daemon/socket or set `%s=podman|docker`", composeEngineEnv)
	}
}

func currentUserName() string {
	if user := strings.TrimSpace(os.Getenv("USER")); user != "" {
		return user
	}
	return strconv.Itoa(os.Getuid())
}

func podmanSocketPath() string {
	if host := strings.TrimSpace(os.Getenv("DOCKER_HOST")); strings.HasPrefix(host, "unix://") {
		return strings.TrimPrefix(host, "unix://")
	}
	if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
		return filepath.Join(runtimeDir, "podman", "podman.sock")
	}
	return filepath.Join("/run", "user", strconv.Itoa(os.Getuid()), "podman", "podman.sock")
}

func oneLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func runCompose(workDir string, args ...string) error {
	compose, err := detectComposeCommand()
	if err != nil {
		return err
	}
	cmdArgs := append(compose[1:], args...)
	cmd := exec.Command(compose[0], cmdArgs...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runComposeCapture(workDir string, args ...string) error {
	_, err := runComposeOutput(workDir, args...)
	return err
}

func runComposeOutput(workDir string, args ...string) (string, error) {
	compose, err := detectComposeCommand()
	if err != nil {
		return "", err
	}
	cmdArgs := append(compose[1:], args...)
	cmd := exec.Command(compose[0], cmdArgs...)
	cmd.Dir = workDir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, message)
	}
	return strings.TrimSpace(output.String()), nil
}

func runBuildAppBaseScript(paths Paths, opts buildAppBaseOptions) error {
	scriptPath := filepath.Join(paths.RepoRoot, "deploy", "base", "scripts", "build-app-base-image.sh")
	if !fileExists(scriptPath) {
		return fmt.Errorf("app-base build script not found: %s", scriptPath)
	}

	args := []string{scriptPath}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	cmd := exec.Command("bash", args...)
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	if opts.Image != "" {
		cmd.Env = append(cmd.Env, "KAGEOS_APP_BASE_IMAGE="+opts.Image)
	}
	return cmd.Run()
}

func runDevInfraScript(paths Paths, opts initDevOptions) error {
	return runDevInfraCommand(paths, opts.Engine, "up", "-d")
}

func runDevInfraCommand(paths Paths, engine string, commandArgs ...string) error {
	scriptPath := filepath.Join(paths.RepoRoot, "deploy", "dev", "scripts", "infra.sh")
	if !fileExists(scriptPath) {
		return fmt.Errorf("dev infra script not found: %s", scriptPath)
	}

	args := []string{scriptPath}
	engine = normalizeDevEngine(engine)
	if engine != "" {
		args = append(args, engine)
	}
	args = append(args, commandArgs...)

	cmd := exec.Command("bash", args...)
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	return cmd.Run()
}
