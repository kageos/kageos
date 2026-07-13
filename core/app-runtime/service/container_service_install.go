package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *PodmanService) isContainerRuntimeInstalled() bool {
	_, err := exec.LookPath("podman")
	return err == nil
}

// installContainerRuntime 自动安装容器运行时
func (s *PodmanService) installContainerRuntime(ctx context.Context) error {
	logger.Infof(ctx, "开始自动安装容器运行时...")

	switch runtime.GOOS {
	case "darwin":
		return s.installOnMacOS(ctx)
	case "linux":
		return s.installOnLinux(ctx)
	case "windows":
		return s.installOnWindows(ctx)
	default:
		return fmt.Errorf("不支持的平台: %s", runtime.GOOS)
	}
}

// installOnMacOS 在 macOS 上安装
func (s *PodmanService) installOnMacOS(ctx context.Context) error {
	logger.Infof(ctx, "正在 macOS 上安装 Podman...")

	// 检查 Homebrew 是否安装
	if _, err := exec.LookPath("brew"); err != nil {
		logger.Errorf(ctx, "未找到 Homebrew，请先安装 Homebrew:")
		logger.Infof(ctx, "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
		return fmt.Errorf("homebrew 未安装")
	}

	// 使用 Homebrew 安装 Podman
	logger.Infof(ctx, "使用 Homebrew 安装 Podman...")
	cmd := exec.CommandContext(ctx, "brew", "install", "podman")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 podman 失败: %w", err)
	}

	logger.Infof(ctx, "Podman 安装成功！")

	// 初始化 Podman Machine
	logger.Infof(ctx, "初始化 Podman Machine...")
	cmd = exec.CommandContext(ctx, "podman", "machine", "init")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Warnf(ctx, "初始化 Podman Machine 失败: %v", err)
		logger.Infof(ctx, "请手动运行: podman machine init")
	} else {
		logger.Infof(ctx, "Podman Machine 初始化成功！")
	}

	// 启动 Podman Machine
	logger.Infof(ctx, "启动 Podman Machine...")
	cmd = exec.CommandContext(ctx, "podman", "machine", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Warnf(ctx, "启动 Podman Machine 失败: %v", err)
		logger.Infof(ctx, "请手动运行: podman machine start")
	} else {
		logger.Infof(ctx, "Podman Machine 启动成功！")
	}

	logger.Infof(ctx, "✅ 容器运行时安装完成！")
	return nil
}

// installOnLinux 在 Linux 上安装
func (s *PodmanService) installOnLinux(ctx context.Context) error {
	logger.Infof(ctx, "正在 Linux 上安装 Podman...")

	// 检测 Linux 发行版
	distro := s.detectLinuxDistro()
	logger.Infof(ctx, "检测到发行版: %s", distro)

	switch distro {
	case "ubuntu", "debian":
		return s.installOnDebian(ctx)
	case "centos", "rhel", "fedora":
		return s.installOnRHEL(ctx)
	case "arch":
		return s.installOnArch(ctx)
	default:
		logger.Warnf(ctx, "未识别的发行版，尝试使用通用方法...")
		return fmt.Errorf("无法自动安装，请手动安装")
	}
}

// installOnWindows 在 Windows 上安装
func (s *PodmanService) installOnWindows(ctx context.Context) error {
	logger.Infof(ctx, "正在 Windows 上安装 Podman...")
	logger.Infof(ctx, "Windows 需要手动下载安装包:")
	logger.Infof(ctx, "1. 访问: https://github.com/containers/podman/releases")
	logger.Infof(ctx, "2. 下载最新的 podman-*-setup.exe")
	logger.Infof(ctx, "3. 运行安装程序")
	logger.Infof(ctx, "4. 安装完成后，打开命令提示符运行:")
	logger.Infof(ctx, "   podman machine init")
	logger.Infof(ctx, "   podman machine start")
	return fmt.Errorf("请手动下载安装 Podman")
}

// 其他辅助方法...
func (s *PodmanService) detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}

	content := strings.ToLower(string(data))
	if strings.Contains(content, "ubuntu") {
		return "ubuntu"
	} else if strings.Contains(content, "debian") {
		return "debian"
	} else if strings.Contains(content, "centos") {
		return "centos"
	} else if strings.Contains(content, "rhel") || strings.Contains(content, "red hat") {
		return "rhel"
	} else if strings.Contains(content, "fedora") {
		return "fedora"
	} else if strings.Contains(content, "arch") {
		return "arch"
	}

	return "unknown"
}

func (s *PodmanService) installOnDebian(ctx context.Context) error {
	logger.Infof(ctx, "使用 apt 安装 Podman...")

	// 更新包列表
	logger.Infof(ctx, "更新包列表...")
	cmd := exec.CommandContext(ctx, "sudo", "apt-get", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("更新包列表失败: %w", err)
	}

	// 安装 Podman
	logger.Infof(ctx, "安装 Podman...")
	cmd = exec.CommandContext(ctx, "sudo", "apt-get", "install", "-y", "podman")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 podman 失败: %w", err)
	}

	logger.Infof(ctx, "✅ Podman 安装完成！")
	return nil
}

func (s *PodmanService) installOnRHEL(ctx context.Context) error {
	logger.Infof(ctx, "使用 yum/dnf 安装 Podman...")

	// 尝试 dnf（较新的系统）
	if _, err := exec.LookPath("dnf"); err == nil {
		cmd := exec.CommandContext(ctx, "sudo", "dnf", "install", "-y", "podman")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装 podman 失败: %w", err)
		}
	} else {
		// 回退到 yum
		cmd := exec.CommandContext(ctx, "sudo", "yum", "install", "-y", "podman")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("安装 podman 失败: %w", err)
		}
	}

	logger.Infof(ctx, "✅ Podman 安装完成！")
	return nil
}

func (s *PodmanService) installOnArch(ctx context.Context) error {
	logger.Infof(ctx, "使用 pacman 安装 Podman...")

	cmd := exec.CommandContext(ctx, "sudo", "pacman", "-S", "--noconfirm", "podman")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 podman 失败: %w", err)
	}

	logger.Infof(ctx, "✅ Podman 安装完成！")
	return nil
}

// prepareContainerRuntimeEnvironment 准备容器运行时环境
func (s *PodmanService) prepareContainerRuntimeEnvironment() error {
	switch runtime.GOOS {
	case "linux":
		return s.prepareLinuxEnvironment()
	case "darwin":
		return s.prepareMacOSEnvironment()
	case "windows":
		return s.prepareWindowsEnvironment()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// prepareLinuxEnvironment 准备 Linux 环境
func (s *PodmanService) prepareLinuxEnvironment() error {
	logger.Infof(s.ctx, "Preparing container runtime on Linux...")

	// 检查 Podman socket 是否存在
	socketPath := "/run/podman/podman.sock"
	if _, err := os.Stat(socketPath); err != nil {
		// Socket 不存在，尝试启动 Podman service
		logger.Infof(s.ctx, "Starting Podman service...")

		// 尝试使用 systemd
		cmd := exec.Command("systemctl", "--user", "start", "podman.socket")
		if err := cmd.Run(); err != nil {
			// systemd 启动失败，尝试直接启动服务
			logger.Warnf(s.ctx, "systemd start failed, trying podman system service...")

			cmd = exec.Command("podman", "system", "service", "--time=0", "unix:///run/user/"+os.Getenv("UID")+"/podman/podman.sock")
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to start podman service: %w", err)
			}

			// 等待服务启动
			cfg := appconfig.GetAppRuntimeConfig()
			timeout := time.Duration(cfg.GetContainerStartupTimeout()) * time.Second
			time.Sleep(timeout)
		}
	}

	return nil
}

// prepareMacOSEnvironment 准备 macOS 环境
func (s *PodmanService) prepareMacOSEnvironment() error {
	// 检查 Podman Machine 状态
	cmd := exec.Command("podman", "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check podman machine status: %w\n\n"+
			"Try running: podman machine init", err)
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command("podman", "machine", "start")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start podman machine: %w\n\n"+
				"Try running manually: podman machine start", err)
		}

		// 等待 Machine 启动
		logger.Infof(s.ctx, "Waiting for Podman Machine to be ready...")
		cfg := appconfig.GetAppRuntimeConfig()
		timeout := time.Duration(cfg.GetContainerCleanupTimeout()) * time.Second
		time.Sleep(timeout)
	}

	return nil
}

// prepareWindowsEnvironment 准备 Windows 环境
func (s *PodmanService) prepareWindowsEnvironment() error {

	// 检查 WSL2
	cmd := exec.Command("wsl", "--status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("WSL2 is not available: %w\n\n"+
			"Please enable WSL2:\n"+
			"  wsl --update\n"+
			"  wsl --install --no-distribution", err)
	}

	// 检查 Podman Machine 状态
	cmd = exec.Command("podman", "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check podman machine status: %w\n\n"+
			"Try running: podman machine init", err)
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command("podman", "machine", "start")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start podman machine: %w\n\n"+
				"Try running manually: podman machine start", err)
		}

		// 等待 Machine 启动
		logger.Infof(s.ctx, "Waiting for Podman Machine to be ready...")
		cfg := appconfig.GetAppRuntimeConfig()
		timeout := time.Duration(cfg.GetContainerCleanupTimeout()) * time.Second
		time.Sleep(timeout)
	}

	return nil
}

// connectToContainerRuntime 连接到容器运行时
func (s *PodmanService) connectToContainerRuntime() error {
	// 使用 Podman CLI 作为稳定边界，避免把 Podman/Docker Go SDK 的未修复 CVE 传递到默认二进制。
	cmd := exec.CommandContext(s.ctx, "podman", s.podmanArgs("info", "--format", "json")...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		target := "default podman context"
		if socket := s.config.GetSocket(); socket != "" {
			target = socket
		}
		return fmt.Errorf("failed to connect to container runtime (%s): %w\n\n"+
			"Output:\n%s\n\n"+
			"Troubleshooting:\n"+
			"  1. Check if podman is running: podman info\n"+
			"  2. Linux: systemctl --user start podman.socket\n"+
			"  3. macOS/Windows: podman machine start", target, err, string(output))
	}

	s.connected = true
	return nil
}
