package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	cmd := args[0]
	opts, rest, err := parseCommonFlags(args[1:])
	if err != nil {
		return err
	}

	paths, err := resolvePaths(opts)
	if err != nil {
		return err
	}

	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "init":
		return cmdInit(paths, rest)
	case "bootstrap":
		return cmdBootstrap(paths, rest)
	case "build-app-base":
		return cmdBuildAppBase(paths, rest)
	case "render":
		return cmdRender(paths)
	case "reload-tls":
		return cmdReloadTLS(paths)
	case "layers", "topology":
		return cmdLayers(paths, rest)
	case "doctor":
		return cmdDoctor(paths, rest)
	case "up":
		return cmdUp(paths, rest)
	case "verify":
		return cmdVerify(paths, rest)
	case "status", "ps":
		return cmdStatus(paths, rest)
	case "logs":
		return cmdLogs(paths, rest)
	case "down":
		return cmdDown(paths)
	case "uninstall", "reset":
		return cmdUninstall(paths, rest)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}
