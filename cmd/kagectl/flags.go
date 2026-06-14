package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type commonOptions struct {
	ConfigPath string
	ProdDir    string
}

type outputOptions struct {
	JSON bool
}

type upOptions struct {
	UseImage      bool
	NoBuild       bool
	SkipVerify    bool
	VerifyTimeout time.Duration
}

type uninstallOptions struct {
	PurgeData          bool
	PurgePodmanStorage bool
	PurgeImages        bool
	KeepGenerated      bool
	PurgePrivateConfig bool
	Force              bool
	DryRun             bool
}

type initOptions struct {
	Force            bool
	BaseURL          string
	Timezone         string
	HTTPPort         int
	HTTPSPort        int
	MySQLMode        string
	CompanyCode      string
	CompanyName      string
	RegistrationMode string
	SMTPMode         string
}

type bootstrapOptions struct {
	Init   initOptions
	UpArgs []string
}

type bootstrapDevOptions struct {
	Init   initDevOptions
	UpArgs []string
}

type buildAppBaseOptions struct {
	Image   string
	Force   bool
	NoCache bool
}

type initDevOptions struct {
	Engine       string
	SkipBase     bool
	BaseImage    string
	BaseForce    bool
	BaseNoCache  bool
	RegenSecrets bool
	CompanyCode  string
	CompanyName  string
}

func parseCommonFlags(args []string) (commonOptions, []string, error) {
	opts := commonOptions{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--config":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--config requires a path")
			}
			opts.ConfigPath = args[i]
		case "--prod-dir":
			i++
			if i >= len(args) {
				return opts, nil, fmt.Errorf("--prod-dir requires a path")
			}
			opts.ProdDir = args[i]
		default:
			rest = append(rest, arg)
		}
	}
	return opts, rest, nil
}

func parseOutputFlags(command string, args []string) (outputOptions, []string, error) {
	opts := outputOptions{}
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.JSON = true
		default:
			rest = append(rest, arg)
		}
	}
	if len(rest) > 0 {
		return opts, rest, fmt.Errorf("%s does not support argument %q", command, rest[0])
	}
	return opts, rest, nil
}

func parseUpFlags(args []string) (upOptions, error) {
	opts := upOptions{VerifyTimeout: defaultUpVerifyTimeout}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--image":
			opts.UseImage = true
		case "--no-build":
			opts.NoBuild = true
		case "--skip-verify":
			opts.SkipVerify = true
		case "--wait-timeout":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--wait-timeout requires a duration, e.g. 5m or 30s")
			}
			timeout, err := time.ParseDuration(args[i])
			if err != nil {
				return opts, fmt.Errorf("parse --wait-timeout: %w", err)
			}
			if timeout <= 0 {
				return opts, fmt.Errorf("--wait-timeout must be greater than 0")
			}
			opts.VerifyTimeout = timeout
		default:
			return opts, fmt.Errorf("up does not support argument %q", args[i])
		}
	}
	if opts.UseImage && opts.NoBuild {
		return opts, fmt.Errorf("--image and --no-build cannot be used together")
	}
	return opts, nil
}

func parseUninstallFlags(args []string) (uninstallOptions, error) {
	opts := uninstallOptions{}
	for _, arg := range args {
		switch arg {
		case "--purge-data":
			opts.PurgeData = true
		case "--purge-podman-storage":
			opts.PurgePodmanStorage = true
		case "--purge-images":
			opts.PurgeImages = true
		case "--keep-generated":
			opts.KeepGenerated = true
		case "--purge-private-config":
			opts.PurgePrivateConfig = true
		case "--force":
			opts.Force = true
		case "--dry-run":
			opts.DryRun = true
		default:
			return opts, fmt.Errorf("uninstall does not support argument %q", arg)
		}
	}
	if opts.PurgePodmanStorage && !opts.PurgeData {
		return opts, fmt.Errorf("--purge-podman-storage requires --purge-data")
	}
	if uninstallRequiresForce(opts) && !opts.Force && !opts.DryRun {
		return opts, fmt.Errorf("destructive uninstall options require --force; preview with --dry-run first")
	}
	return opts, nil
}

func parseInitFlags(command string, args []string) (initOptions, error) {
	opts := initOptions{MySQLMode: "bundled"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			opts.Force = true
		case "--base-url":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-url requires a value")
			}
			opts.BaseURL = args[i]
		case "--timezone":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--timezone requires a value")
			}
			opts.Timezone = strings.TrimSpace(args[i])
		case "--http-port":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--http-port requires a value")
			}
			port, err := parseTCPPortValue("--http-port", args[i])
			if err != nil {
				return opts, err
			}
			opts.HTTPPort = port
		case "--https-port":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--https-port requires a value")
			}
			port, err := parseTCPPortValue("--https-port", args[i])
			if err != nil {
				return opts, err
			}
			opts.HTTPSPort = port
		case "--mysql-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--mysql-mode requires a value")
			}
			opts.MySQLMode = args[i]
		case "--company-code":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-code requires a value")
			}
			opts.CompanyCode = strings.TrimSpace(args[i])
		case "--company-name":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-name requires a value")
			}
			opts.CompanyName = strings.TrimSpace(args[i])
		case "--registration-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registration-mode requires admin_only, email_code, or debug_code")
			}
			opts.RegistrationMode = strings.TrimSpace(args[i])
		case "--smtp-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--smtp-mode requires smtp or log")
			}
			opts.SMTPMode = strings.TrimSpace(args[i])
		default:
			return opts, fmt.Errorf("%s does not support argument %q", command, args[i])
		}
	}
	if err := validateMode("mysql.mode", opts.MySQLMode); err != nil {
		return opts, err
	}
	if err := validateRegistrationMode(opts.RegistrationMode); err != nil {
		return opts, err
	}
	if err := validateSMTPMode(opts.SMTPMode); err != nil {
		return opts, err
	}
	if err := validateTimezoneValue("--timezone", opts.Timezone); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseBootstrapFlags(args []string) (bootstrapOptions, error) {
	opts := bootstrapOptions{Init: initOptions{MySQLMode: "bundled"}}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			opts.Init.Force = true
		case "--base-url":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-url requires a value")
			}
			opts.Init.BaseURL = args[i]
		case "--timezone":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--timezone requires a value")
			}
			opts.Init.Timezone = strings.TrimSpace(args[i])
		case "--http-port":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--http-port requires a value")
			}
			port, err := parseTCPPortValue("--http-port", args[i])
			if err != nil {
				return opts, err
			}
			opts.Init.HTTPPort = port
		case "--https-port":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--https-port requires a value")
			}
			port, err := parseTCPPortValue("--https-port", args[i])
			if err != nil {
				return opts, err
			}
			opts.Init.HTTPSPort = port
		case "--mysql-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--mysql-mode requires a value")
			}
			opts.Init.MySQLMode = args[i]
		case "--registration-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--registration-mode requires admin_only, email_code, or debug_code")
			}
			opts.Init.RegistrationMode = strings.TrimSpace(args[i])
		case "--smtp-mode":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--smtp-mode requires smtp or log")
			}
			opts.Init.SMTPMode = strings.TrimSpace(args[i])
		case "--image", "--no-build", "--skip-verify":
			opts.UpArgs = append(opts.UpArgs, args[i])
		case "--wait-timeout":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--wait-timeout requires a duration, e.g. 5m or 30s")
			}
			opts.UpArgs = append(opts.UpArgs, "--wait-timeout", args[i])
		default:
			return opts, fmt.Errorf("bootstrap does not support argument %q", args[i])
		}
	}
	if err := validateMode("mysql.mode", opts.Init.MySQLMode); err != nil {
		return opts, err
	}
	if err := validateRegistrationMode(opts.Init.RegistrationMode); err != nil {
		return opts, err
	}
	if err := validateSMTPMode(opts.Init.SMTPMode); err != nil {
		return opts, err
	}
	if err := validateTimezoneValue("--timezone", opts.Init.Timezone); err != nil {
		return opts, err
	}
	if _, err := parseUpFlags(opts.UpArgs); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseTCPPortValue(name string, value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("%s must be a TCP port between 1 and 65535", name)
	}
	return port, nil
}

func parseBootstrapDevFlags(args []string) (bootstrapDevOptions, error) {
	opts := bootstrapDevOptions{}
	initArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--skip-verify":
			opts.UpArgs = append(opts.UpArgs, args[i])
		case "--wait-timeout":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--wait-timeout requires a duration, e.g. 5m or 30s")
			}
			opts.UpArgs = append(opts.UpArgs, "--wait-timeout", args[i])
		case "--image", "--no-build":
			return opts, fmt.Errorf("bootstrap --dev does not support %s; dev runs source code directly", args[i])
		default:
			initArgs = append(initArgs, args[i])
		}
	}

	initOpts, err := parseInitDevFlags(initArgs)
	if err != nil {
		return opts, err
	}
	if _, err := parseUpFlags(opts.UpArgs); err != nil {
		return opts, err
	}
	opts.Init = initOpts
	return opts, nil
}

func parseBuildAppBaseFlags(args []string) (buildAppBaseOptions, error) {
	opts := buildAppBaseOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--image":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--image requires a value")
			}
			opts.Image = strings.TrimSpace(args[i])
			if opts.Image == "" {
				return opts, fmt.Errorf("--image cannot be empty")
			}
		case "--force":
			opts.Force = true
		case "--no-cache":
			opts.NoCache = true
		default:
			return opts, fmt.Errorf("build-app-base does not support argument %q", args[i])
		}
	}
	return opts, nil
}

func parseInitDevFlags(args []string) (initDevOptions, error) {
	opts := initDevOptions{Engine: "podman"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--engine":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--engine requires auto, docker, or podman")
			}
			opts.Engine = strings.TrimSpace(args[i])
		case "--skip-base":
			opts.SkipBase = true
		case "--base-image":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--base-image requires a value")
			}
			opts.BaseImage = strings.TrimSpace(args[i])
			if opts.BaseImage == "" {
				return opts, fmt.Errorf("--base-image cannot be empty")
			}
		case "--base-force":
			opts.BaseForce = true
		case "--base-no-cache":
			opts.BaseNoCache = true
		case "--regen-secrets":
			opts.RegenSecrets = true
		case "--company-code":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-code requires a value")
			}
			opts.CompanyCode = strings.TrimSpace(args[i])
		case "--company-name":
			i++
			if i >= len(args) {
				return opts, fmt.Errorf("--company-name requires a value")
			}
			opts.CompanyName = strings.TrimSpace(args[i])
		default:
			if args[i] == "auto" || args[i] == "docker" || args[i] == "podman" {
				opts.Engine = args[i]
				continue
			}
			return opts, fmt.Errorf("init --dev does not support argument %q", args[i])
		}
	}
	if opts.Engine != "auto" && opts.Engine != "docker" && opts.Engine != "podman" {
		return opts, fmt.Errorf("--engine requires auto, docker, or podman")
	}
	return opts, nil
}

func printUsage() {
	fmt.Println(`kagectl manages Kageos lifecycle.

Usage:
  kagectl init [--force] [--base-url URL] [--timezone TZ] [--http-port PORT] [--https-port PORT] [--mysql-mode bundled|external] [--company-code CODE] [--company-name NAME] [--registration-mode admin_only|email_code|debug_code] [--smtp-mode smtp|log]
  kagectl init --dev [--engine podman|docker|auto] [--skip-base] [--regen-secrets] [--base-image IMAGE] [--base-force] [--base-no-cache] [--company-code CODE] [--company-name NAME]
  kagectl bootstrap --base-url URL [--timezone TZ] [--http-port PORT] [--https-port PORT] [--mysql-mode bundled|external] [--registration-mode admin_only|email_code|debug_code] [--smtp-mode smtp|log] [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  kagectl bootstrap --dev [--engine podman|docker|auto] [--skip-base] [--regen-secrets] [--base-image IMAGE] [--base-force] [--base-no-cache] [--company-code CODE] [--company-name NAME] [--skip-verify] [--wait-timeout 5m]
  kagectl build-app-base [--image IMAGE] [--force] [--no-cache]
  kagectl render [--config .kageos/prod/kage.yaml]
  kagectl layers [--config .kageos/prod/kage.yaml] [--json]
  kagectl doctor [--json]
  kagectl up [--image|--no-build] [--skip-verify] [--wait-timeout 5m]
  kagectl verify [--json]
  kagectl status [--json]
  kagectl logs [main|infra|service|layer] [--layer L0-L5]
  kagectl down
  kagectl uninstall [--config .kageos/prod/kage.yaml] [--purge-data] [--purge-podman-storage] [--purge-images] [--keep-generated] [--purge-private-config] [--force] [--dry-run]

Modes:
  prod is the default and writes .kageos/kageos.env KAGEOS_MODE=prod.
  init --dev writes .kageos/kageos.env KAGEOS_MODE=dev; later up/status/down/logs use that mode.

Environment:
  KAGEOS_COMPOSE_ENGINE=podman|docker forces the production compose engine.
  KAGEOS_TIMEZONE sets the deployment timezone. Defaults to Asia/Shanghai.
  KAGEOS_HTTP_PORT/KAGEOS_HTTPS_PORT override the production edge listen ports.

Compose remains the container execution engine; kagectl owns config rendering, orchestration, and diagnostics.`)
}

func takeDevFlag(args []string) (bool, []string) {
	dev := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--dev" {
			dev = true
			continue
		}
		rest = append(rest, arg)
	}
	return dev, rest
}
