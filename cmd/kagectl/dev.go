package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func cmdDevUp(paths Paths, opts upOptions) error {
	if opts.UseImage || opts.NoBuild {
		return fmt.Errorf("dev up runs source code directly and does not support --image or --no-build")
	}
	if err := requireDevLayout(paths); err != nil {
		return err
	}
	fmt.Println("[dev] 启动本地基础设施")
	if err := runDevInfraCommand(paths, currentDevEngine(paths), "up", "-d"); err != nil {
		return err
	}
	if !opts.SkipVerify {
		fmt.Printf("[dev] 检查本地依赖（timeout %s）\n", opts.VerifyTimeout)
		if err := waitLayerChecks("dev dependencies", devDependencyChecks(paths), opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
			return err
		}
	}
	fmt.Println("[dev] 启动 kageos 后端主进程")
	fmt.Println("[dev] Stop with Ctrl-C. Run `kagectl down` to stop local infra containers.")
	return runDevMain(paths)
}

func cmdDevStatus(paths Paths, opts outputOptions) error {
	if opts.JSON {
		report := map[string]any{
			"mode":       workspaceModeDev,
			"mode_env":   workspaceEnvPath(paths),
			"config_dir": filepath.Join(paths.RepoRoot, defaultDevConfig),
			"env_file":   devEnvPath(paths),
			"engine":     currentDevEngine(paths),
		}
		return writeJSON(report)
	}
	if err := requireDevLayout(paths); err != nil {
		return err
	}
	fmt.Printf("kageos mode: dev (%s)\n", workspaceEnvPath(paths))
	return runDevInfraCommand(paths, currentDevEngine(paths), "ps")
}

func cmdDevLogs(paths Paths, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("dev logs accepts at most one target: main or infra")
	}
	target := "main"
	if len(args) == 1 {
		target = args[0]
	}
	switch target {
	case "infra":
		if err := requireDevLayout(paths); err != nil {
			return err
		}
		return runDevInfraCommand(paths, currentDevEngine(paths), "logs", "-f")
	case "main", "platform":
		return tailDevMainLog(paths)
	default:
		return fmt.Errorf("dev logs supports main/platform or infra, got %q", target)
	}
}

func cmdDevDown(paths Paths) error {
	if err := requireDevLayout(paths); err != nil {
		return err
	}
	fmt.Println("[dev] 停止本地基础设施")
	return runDevInfraCommand(paths, currentDevEngine(paths), "down")
}

func requireDevLayout(paths Paths) error {
	missing := make([]string, 0)
	if !workspaceModeFileExists(paths) {
		missing = append(missing, workspaceEnvPath(paths))
	}
	for _, path := range []string{filepath.Join(paths.RepoRoot, defaultDevConfig), devEnvPath(paths)} {
		if !fileExists(path) && !dirExists(path) {
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("dev workspace is not initialized; missing %s; run `kagectl init --dev`", strings.Join(missing, ", "))
	}
	return nil
}

func devEnvPath(paths Paths) string {
	return filepath.Join(paths.RepoRoot, ".kageos", "dev", "env", "kageos.env")
}

func runDevMain(paths Paths) error {
	cmd := exec.Command("go", "run", "./core/cmd/main")
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "KAGEOS_ROOT="+paths.RepoRoot)
	return cmd.Run()
}

func tailDevMainLog(paths Paths) error {
	logPath := filepath.Join(paths.RepoRoot, "logs", "all-services.log")
	if !fileExists(logPath) {
		return fmt.Errorf("dev main log not found: %s; start the backend with `kagectl up` first", logPath)
	}
	cmd := exec.Command("tail", "-f", logPath)
	cmd.Dir = paths.RepoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func devDoctorChecks(paths Paths) []layerCheck {
	return []layerCheck{
		{Layer: layerControl, Name: "workspace mode", Target: workspaceEnvPath(paths), Fn: func() error {
			if currentWorkspaceMode(paths) != workspaceModeDev {
				return fmt.Errorf("workspace mode is not dev")
			}
			return nil
		}},
		{Layer: layerControl, Name: "dev config dir", Target: filepath.Join(paths.RepoRoot, defaultDevConfig), Fn: func() error {
			if !dirExists(filepath.Join(paths.RepoRoot, defaultDevConfig)) {
				return fmt.Errorf("dev config dir missing; run `kagectl init --dev`")
			}
			return nil
		}},
		{Layer: layerControl, Name: "dev env file", Target: devEnvPath(paths), Fn: func() error {
			if !fileExists(devEnvPath(paths)) {
				return fmt.Errorf("dev env file missing; run `kagectl init --dev`")
			}
			return nil
		}},
		{Layer: layerControl, Name: "compose command", Target: currentDevEngine(paths) + " compose", Fn: func() error {
			return checkDevComposeCommand(currentDevEngine(paths))
		}},
	}
}

func devDependencyChecks(paths Paths) []layerCheck {
	return []layerCheck{
		{Layer: layerInfra, Name: "mysql tcp", Target: "127.0.0.1:3318", Fn: func() error {
			return checkTCP("mysql", "127.0.0.1", 3318)
		}},
		{Layer: layerInfra, Name: "nats tcp", Target: "127.0.0.1:4222", Fn: func() error {
			return checkTCP("nats", "127.0.0.1", 4222)
		}},
		{Layer: layerInfra, Name: "minio tcp", Target: "127.0.0.1:9000", Fn: func() error {
			return checkTCP("minio", "127.0.0.1", 9000)
		}},
		{Layer: layerInfra, Name: "minio clock", Target: minIOClockCheckURL("127.0.0.1", 9000, false), Fn: func() error {
			return checkMinIOClock("127.0.0.1", 9000, false)
		}},
	}
}

func devVerifyChecks(paths Paths) []layerCheck {
	checks := append(devDoctorChecks(paths), devDependencyChecks(paths)...)
	checks = append(checks,
		layerCheck{Layer: layerPlatform, Name: "api-gateway", Target: "http://127.0.0.1:9090/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9090/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-server", Target: "http://127.0.0.1:9091/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9091/health") }},
		layerCheck{Layer: layerPlatform, Name: "app-storage", Target: "http://127.0.0.1:9092/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9092/health") }},
		layerCheck{Layer: layerRuntime, Name: "app-runtime", Target: "http://127.0.0.1:9093/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9093/health") }},
		layerCheck{Layer: layerPlatform, Name: "agent-server", Target: "http://127.0.0.1:9095/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9095/health") }},
		layerCheck{Layer: layerPlatform, Name: "connector-server", Target: "http://127.0.0.1:9096/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9096/health") }},
		layerCheck{Layer: layerPlatform, Name: "hr-server", Target: "http://127.0.0.1:9097/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9097/health") }},
		layerCheck{Layer: layerPlatform, Name: "timer-scheduler", Target: "http://127.0.0.1:9098/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9098/health") }},
		layerCheck{Layer: layerPlatform, Name: "message-server", Target: "http://127.0.0.1:9099/health", Fn: func() error { return checkHTTP("http://127.0.0.1:9099/health") }},
	)
	return checks
}
