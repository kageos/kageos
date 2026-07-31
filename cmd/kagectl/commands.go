package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func cmdInit(paths Paths, args []string) error {
	dev, args := takeDevFlag(args)
	if dev {
		return cmdInitDev(paths, args)
	}
	opts, err := parseInitFlags("init", args)
	if err != nil {
		return err
	}
	created, err := writeInitialConfig(paths, opts)
	if err != nil {
		return err
	}
	if err := writeWorkspaceConfig(paths, workspaceModeProd, workspaceDevConfig{}); err != nil {
		return err
	}
	if created {
		fmt.Println("next: edit site.base_url if needed and run `kagectl up`")
	}
	return nil
}

func cmdInitDev(paths Paths, args []string) error {
	opts, err := parseInitDevFlags(args)
	if err != nil {
		return err
	}
	return runInitDev(paths, opts)
}

func runInitDev(paths Paths, opts initDevOptions) error {
	if err := renderDevConfig(paths, opts.RegenSecrets); err != nil {
		return err
	}
	if err := writeWorkspaceConfig(paths, workspaceModeDev, workspaceDevConfig{Engine: opts.Engine}); err != nil {
		return err
	}
	if err := runDevInfraScript(paths, opts); err != nil {
		return err
	}
	if opts.SkipBase {
		fmt.Println("==> skip app base image (--skip-base)")
		printDevInitSummary(paths, opts)
		return nil
	}
	fmt.Println("==> ensure app base image")
	if err := runBuildAppBaseScript(paths, buildAppBaseOptions{
		Image:   opts.BaseImage,
		Force:   opts.BaseForce,
		NoCache: opts.BaseNoCache,
	}); err != nil {
		return err
	}
	printDevInitSummary(paths, opts)
	return nil
}

func cmdBootstrap(paths Paths, args []string) error {
	dev, args := takeDevFlag(args)
	if dev {
		return cmdBootstrapDev(paths, args)
	}

	opts, err := parseBootstrapFlags(args)
	if err != nil {
		return err
	}
	if fileExists(paths.ConfigPath) && !opts.Init.Force {
		fmt.Printf("using existing config: %s\n", paths.ConfigPath)
	} else {
		if opts.Init.BaseURL == "" {
			return fmt.Errorf("bootstrap requires --base-url when creating config")
		}
		if _, err := writeInitialConfig(paths, opts.Init); err != nil {
			return err
		}
	}
	if err := writeWorkspaceConfig(paths, workspaceModeProd, workspaceDevConfig{}); err != nil {
		return err
	}
	return cmdUp(paths, opts.UpArgs)
}

func cmdBootstrapDev(paths Paths, args []string) error {
	opts, err := parseBootstrapDevFlags(args)
	if err != nil {
		return err
	}
	if err := runInitDev(paths, opts.Init); err != nil {
		return err
	}
	return cmdUp(paths, opts.UpArgs)
}

func cmdBuildAppBase(paths Paths, args []string) error {
	opts, err := parseBuildAppBaseFlags(args)
	if err != nil {
		return err
	}
	return runBuildAppBaseScript(paths, opts)
}

func writeInitialConfig(paths Paths, opts initOptions) (bool, error) {
	if fileExists(paths.ConfigPath) && !opts.Force {
		fmt.Printf("config already exists: %s\n", paths.ConfigPath)
		fmt.Println("use --force to overwrite it")
		return false, nil
	}

	cfg, err := defaultConfig()
	if err != nil {
		return false, err
	}
	cfg.Site.BaseURL = opts.BaseURL
	if opts.TLSMode != "" && opts.TLSMode != tlsModeAuto {
		cfg.Site.TLSMode = opts.TLSMode
	}
	if opts.Timezone != "" {
		cfg.Timezone = opts.Timezone
	}
	if opts.HTTPPort > 0 {
		cfg.Site.HTTPPort = opts.HTTPPort
	}
	if opts.HTTPSPort > 0 {
		cfg.Site.HTTPSPort = opts.HTTPSPort
	}
	cfg.MySQL.Mode = opts.MySQLMode
	if opts.RegistrationMode != "" {
		cfg.Auth.RegistrationMode = opts.RegistrationMode
	}
	if opts.SMTPMode != "" {
		cfg.SMTP.Mode = opts.SMTPMode
	}
	applyEnvOverrides(&cfg)
	tlsMode := opts.TLSMode
	if envTLSMode := strings.TrimSpace(os.Getenv("KAGEOS_TLS_MODE")); envTLSMode != "" {
		tlsMode = envTLSMode
	}
	if err := applyInitialSitePolicy(&cfg.Site, tlsMode); err != nil {
		return false, err
	}
	applyDefaults(&cfg)
	if err := validateTimezoneValue("timezone", cfg.Timezone); err != nil {
		return false, err
	}
	if opts.MySQLMode == "external" {
		cfg.MySQL.Host = ""
		cfg.MySQL.User = ""
		cfg.MySQL.Password = ""
	}

	if err := os.MkdirAll(filepath.Dir(paths.ConfigPath), 0755); err != nil {
		return false, err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(paths.ConfigPath, data, 0600); err != nil {
		return false, err
	}

	fmt.Printf("created config: %s\n", paths.ConfigPath)
	printProdInitSummary(paths, cfg)
	return true, nil
}

func cmdRender(paths Paths) error {
	if currentWorkspaceMode(paths) == workspaceModeDev {
		if err := renderDevConfig(paths, false); err != nil {
			return err
		}
		fmt.Printf("rendered dev config: %s\n", filepath.Join(paths.RepoRoot, defaultDevConfig))
		return nil
	}
	cfg, err := loadConfig(paths)
	if err != nil {
		return err
	}
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if err := renderAll(rt); err != nil {
		return err
	}
	fmt.Printf("rendered deployment files: %s\n", paths.GeneratedDir)
	fmt.Printf("compose file: %s\n", filepath.Join(paths.GeneratedDir, "docker-compose.yaml"))
	return nil
}

func cmdReloadTLS(paths Paths) error {
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return fmt.Errorf("reload-tls is only available in prod mode")
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if rt.Site.TLSMode != "https" && rt.Site.TLSMode != "redirect" {
		return fmt.Errorf("reload-tls requires site.tls_mode=https or redirect, got %q", rt.Site.TLSMode)
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	if err := os.MkdirAll(rt.TLSCertsHostDir, 0755); err != nil {
		return fmt.Errorf("create TLS directory %s: %w", rt.TLSCertsHostDir, err)
	}
	if err := renderTLSFiles(rt); err != nil {
		return err
	}
	if err := runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "nginx", "-t"); err != nil {
		return err
	}
	if err := runComposeCapture(rt.Paths.GeneratedDir, "exec", "-T", "main", "nginx", "-s", "reload"); err != nil {
		return err
	}
	fmt.Printf("TLS files ready: %s\n", rt.TLSCertsHostDir)
	fmt.Println("Nginx TLS reloaded without recreating containers")
	return nil
}

func cmdLayers(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("layers", args)
	if err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if opts.JSON {
		return writeJSON(buildDeploymentReport(rt))
	}
	printDeploymentLayers(rt)
	return nil
}

func cmdDoctor(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("doctor", args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		if opts.JSON {
			return runLayerChecksJSON("doctor", devDoctorChecks(paths))
		}
		return runLayerChecks("doctor", devDoctorChecks(paths))
	}
	cfg, err := loadConfig(paths)
	if err != nil {
		return err
	}
	rt, err := buildRuntimeConfig(paths, cfg)
	if err != nil {
		return err
	}

	if opts.JSON {
		return runLayerChecksJSON("doctor", doctorLayerChecks(rt))
	}
	return runLayerChecks("doctor", doctorLayerChecks(rt))
}

func cmdUp(paths Paths, args []string) error {
	opts, err := parseUpFlags(args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return cmdDevUp(paths, opts)
	}
	fmt.Println("[L0 部署控制层] 检查宿主机和读取配置")
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	fmt.Println("[L0-L1] 运行分层部署预检")
	if err := runLayerChecks("doctor", doctorLayerChecks(rt)); err != nil {
		return err
	}
	fmt.Println("[L0 部署控制层] 校验配置、准备运行目录、渲染 Compose/config")
	if err := ensureRuntimeLayout(rt); err != nil {
		return err
	}
	if err := renderAll(rt); err != nil {
		return err
	}

	if infraServices := composeServicesForLayer(rt, layerInfra); len(infraServices) > 0 {
		fmt.Printf("[L1 基础设施层] 启动基础设施服务: %s\n", strings.Join(infraServices, ", "))
		args := append([]string{"up", "-d", "--no-build"}, infraServices...)
		if err := runCompose(rt.Paths.GeneratedDir, args...); err != nil {
			return err
		}
	}
	if checks := startupDependencyChecks(rt); len(checks) > 0 {
		fmt.Printf("[L1 基础设施层] 等待基础设施可用（timeout %s）\n", opts.VerifyTimeout)
		if err := waitLayerChecks("startup dependencies", checks, opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
			return err
		}
	}

	fmt.Println("[L4 运行时管理层] 构建用户应用基础镜像工具")
	if err := runCompose(rt.Paths.GeneratedDir, "build", "app-base-builder"); err != nil {
		return err
	}
	fmt.Println("[L4 运行时管理层] 准备用户应用基础镜像")
	if err := runCompose(rt.Paths.GeneratedDir, "run", "--rm", "--no-deps", "app-base-builder"); err != nil {
		return err
	}

	switch {
	case opts.UseImage:
		fmt.Println("[L3 平台服务层] 拉取主镜像")
		if err := runCompose(rt.Paths.GeneratedDir, "pull", "main"); err != nil {
			return err
		}
	case opts.NoBuild:
		fmt.Println("[L3 平台服务层] 跳过主镜像构建/拉取 (--no-build)")
	default:
		fmt.Println("[L3 平台服务层] 构建主镜像")
		if err := runCompose(rt.Paths.GeneratedDir, "build", "main"); err != nil {
			return err
		}
	}

	fmt.Println("[L2-L4] 启动/更新主应用服务")
	if err := runCompose(rt.Paths.GeneratedDir, "up", "-d", "--no-build", "--force-recreate", "main"); err != nil {
		return err
	}
	if opts.SkipVerify {
		fmt.Println("deployment started; layered verify skipped (--skip-verify)")
		if err := finishDeploymentSummary(rt, "started (verification skipped)"); err != nil {
			return err
		}
		return nil
	}
	fmt.Printf("[L0-L5] 等待分层健康检查通过（timeout %s）\n", opts.VerifyTimeout)
	if err := waitLayerChecks("verify", verifyLayerChecks(rt), opts.VerifyTimeout, defaultUpVerifyInterval); err != nil {
		return err
	}
	fmt.Println("deployment ready")
	if err := finishDeploymentSummary(rt, "ready"); err != nil {
		return err
	}
	return nil
}

func cmdVerify(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("verify", args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		if opts.JSON {
			return runLayerChecksJSON("verify", devVerifyChecks(paths))
		}
		return runLayerChecks("verify", devVerifyChecks(paths))
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if opts.JSON {
		return runLayerChecksJSON("verify", verifyLayerChecks(rt))
	}
	return runLayerChecks("verify", verifyLayerChecks(rt))
}

func cmdStatus(paths Paths, args []string) error {
	opts, _, err := parseOutputFlags("status", args)
	if err != nil {
		return err
	}
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return cmdDevStatus(paths, opts)
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}
	if opts.JSON {
		composePS, err := runComposeOutput(paths.GeneratedDir, "ps")
		if err != nil {
			return err
		}
		return writeJSON(statusReport{
			Deployment:        buildDeploymentReport(rt),
			ComposeConfigPath: rt.ComposeConfigPath,
			ComposePS:         composePS,
		})
	}
	fmt.Println("Deployment layers")
	printDeploymentLayers(rt)
	fmt.Println("\nCompose service ownership")
	printComposeServiceOwnership(rt)
	fmt.Println("\nCompose services")
	return runCompose(paths.GeneratedDir, "ps")
}

func cmdLogs(paths Paths, args []string) error {
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return cmdDevLogs(paths, args)
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	layerArg := ""
	targets := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--layer":
			i++
			if i >= len(args) {
				return fmt.Errorf("--layer requires L0-L5, edge, infra, platform, runtime, or apps")
			}
			layerArg = args[i]
		default:
			targets = append(targets, args[i])
		}
	}
	if layerArg != "" && len(targets) > 0 {
		return fmt.Errorf("logs cannot combine --layer with a service name")
	}
	if len(targets) > 1 {
		return fmt.Errorf("logs accepts at most one service name or layer")
	}

	target := "main"
	if len(targets) == 1 {
		target = targets[0]
	}
	if layerArg == "" {
		if _, ok := parseDeploymentLayer(target); !ok {
			return runCompose(paths.GeneratedDir, "logs", "-f", target)
		}
	}

	rt, err := loadRuntimeConfig(paths)
	if err != nil {
		return err
	}
	if err := validateConfig(rt); err != nil {
		return err
	}

	if layerArg != "" {
		return runLayerLogs(rt, layerArg)
	}
	return runLayerLogs(rt, target)
}

func cmdDown(paths Paths) error {
	if currentWorkspaceMode(paths) == workspaceModeDev {
		return cmdDevDown(paths)
	}
	if err := requireGeneratedCompose(paths); err != nil {
		return err
	}
	return runCompose(paths.GeneratedDir, "down")
}

func cmdUninstall(paths Paths, args []string) error {
	opts, err := parseUninstallFlags(args)
	if err != nil {
		return err
	}

	rt, rtErr := loadRuntimeConfig(paths)
	if rtErr != nil && uninstallNeedsRuntimeConfig(opts, paths) {
		return rtErr
	}

	printUninstallPlan(paths, rt, rtErr, opts)
	if opts.DryRun {
		return nil
	}

	composeReady, err := ensureGeneratedComposeForUninstall(paths, rt, rtErr)
	if err != nil {
		return err
	}
	if composeReady {
		downArgs := []string{"down"}
		if opts.PurgeImages {
			downArgs = append(downArgs, "--rmi", "all")
		}
		if err := runCompose(paths.GeneratedDir, downArgs...); err != nil {
			return err
		}
	}

	if opts.PurgeData {
		for _, target := range uninstallDataTargets(rt, opts) {
			if err := removePath(target.Path, target.Label, false); err != nil {
				return err
			}
		}
	}
	if !opts.KeepGenerated {
		if err := removePath(paths.GeneratedDir, "generated deployment files", false); err != nil {
			return err
		}
	}
	if opts.PurgePrivateConfig {
		if err := removePath(paths.ConfigPath, "private deploy config", false); err != nil {
			return err
		}
	}
	fmt.Println("uninstall completed")
	return nil
}

func runLayerLogs(rt RuntimeConfig, value string) error {
	layer, ok := parseDeploymentLayer(value)
	if !ok {
		return fmt.Errorf("unknown deployment layer %q", value)
	}
	services := composeServicesForLayer(rt, layer)
	if len(services) == 0 {
		return fmt.Errorf("%s has no Compose service logs; %s", layerTitle(layer), layerLogHint(layer))
	}
	fmt.Printf("%s logs: %s\n", layerTitle(layer), strings.Join(services, ", "))
	args := append([]string{"logs", "-f"}, services...)
	return runCompose(rt.Paths.GeneratedDir, args...)
}

func layerLogHint(layer deploymentLayerID) string {
	switch layer {
	case layerControl:
		return "kagectl runs on the host and logs to the current terminal"
	case layerApps:
		return "user App containers are managed by app-runtime; use platform app log APIs"
	default:
		return "check `kagectl status` for layer to service ownership"
	}
}
