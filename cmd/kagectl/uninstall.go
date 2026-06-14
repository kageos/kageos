package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func uninstallRequiresForce(opts uninstallOptions) bool {
	return opts.PurgeData || opts.PurgePodmanStorage || opts.PurgeImages || opts.PurgePrivateConfig
}

func uninstallNeedsRuntimeConfig(opts uninstallOptions, paths Paths) bool {
	if opts.PurgeData || opts.PurgePodmanStorage {
		return true
	}
	return !fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml"))
}

func ensureGeneratedComposeForUninstall(paths Paths, rt RuntimeConfig, rtErr error) (bool, error) {
	if fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml")) {
		return true, nil
	}
	if rtErr != nil {
		fmt.Printf("generated compose not found and config unavailable; skipping compose down: %v\n", rtErr)
		return false, nil
	}
	fmt.Println("generated compose not found; rendering it from config before uninstall")
	if err := renderAll(rt); err != nil {
		return false, err
	}
	return true, nil
}

type uninstallTarget struct {
	Label string
	Path  string
}

func uninstallDataTargets(rt RuntimeConfig, opts uninstallOptions) []uninstallTarget {
	root := filepath.Clean(rt.Storage.Root)
	targets := []uninstallTarget{
		{Label: "mysql data", Path: filepath.Join(root, "mysql")},
		{Label: "minio data", Path: filepath.Join(root, "minio")},
		{Label: "user namespace", Path: filepath.Join(root, "namespace")},
		{Label: "app data", Path: filepath.Join(root, "data")},
		{Label: "logs", Path: filepath.Join(root, "logs")},
	}
	if opts.PurgePodmanStorage {
		targets = append(targets, uninstallTarget{Label: "podman storage and app-base image", Path: filepath.Join(root, "podman_storage")})
	}
	return targets
}

func printUninstallPlan(paths Paths, rt RuntimeConfig, rtErr error, opts uninstallOptions) {
	fmt.Println("uninstall plan")
	if fileExists(filepath.Join(paths.GeneratedDir, "docker-compose.yaml")) {
		if opts.PurgeImages {
			fmt.Println("  - compose down --rmi all (stop/remove services and host engine images)")
		} else {
			fmt.Println("  - compose down (stop/remove services; keep host engine images)")
		}
	} else if rtErr == nil {
		fmt.Println("  - render generated compose, then compose down")
	} else {
		fmt.Printf("  - skip compose down: generated compose missing and config unavailable (%v)\n", rtErr)
	}
	if opts.PurgeData {
		for _, target := range uninstallDataTargets(rt, opts) {
			fmt.Printf("  - remove %s: %s\n", target.Label, target.Path)
		}
		if !opts.PurgePodmanStorage {
			fmt.Printf("  - keep podman storage/app-base image: %s\n", filepath.Join(filepath.Clean(rt.Storage.Root), "podman_storage"))
		}
	} else if rtErr == nil {
		fmt.Printf("  - keep runtime data: %s\n", rt.Storage.Root)
	}
	if opts.KeepGenerated {
		fmt.Printf("  - keep generated files: %s\n", paths.GeneratedDir)
	} else {
		fmt.Printf("  - remove generated files: %s\n", paths.GeneratedDir)
	}
	if opts.PurgePrivateConfig {
		fmt.Printf("  - remove private config: %s\n", paths.ConfigPath)
	} else {
		fmt.Printf("  - keep private config: %s\n", paths.ConfigPath)
	}
	if opts.DryRun {
		fmt.Println("dry-run only; no changes made")
	}
}

func removePath(path, label string, dryRun bool) error {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." || clean == string(filepath.Separator) {
		return fmt.Errorf("refuse to remove unsafe %s path: %q", label, path)
	}
	if dryRun {
		fmt.Printf("[dry-run] remove %s: %s\n", label, clean)
		return nil
	}
	if !fileExists(clean) && !dirExists(clean) {
		fmt.Printf("skip missing %s: %s\n", label, clean)
		return nil
	}
	fmt.Printf("remove %s: %s\n", label, clean)
	return os.RemoveAll(clean)
}
