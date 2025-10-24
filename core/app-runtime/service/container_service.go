package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/domain/entities"
	"github.com/containers/podman/v5/pkg/specgen"
)

// ContainerOperator 容器操作接口
type ContainerOperator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	ListContainers(ctx context.Context) ([]entities.ListContainer, error)
	RunContainer(ctx context.Context, image, name string) error
	RunContainerWithMount(ctx context.Context, image, name, hostPath, containerPath string) error
	RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, envVars ...string) error
	IsContainerRunning(ctx context.Context, name string) (bool, error)
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	RemoveContainer(ctx context.Context, name string) error
	ListImages(ctx context.Context) ([]*entities.ImageSummary, error)
	ExecCommand(ctx context.Context, containerName string, command []string) (string, error)
	CopyToContainer(ctx context.Context, containerName, srcPath, destPath string) error
}

// PodmanService Podman 容器服务实现
type PodmanService struct {
	ctx    context.Context
	cancel context.CancelFunc
	config *appconfig.ContainerServiceConfig
	conn   context.Context // Podman bindings 的连接 context
}

// NewDefaultPodmanService 创建新的 Podman 服务（默认，内部获取依赖）
func NewDefaultPodmanService() *PodmanService {
	cfg := appconfig.GetAppRuntimeConfig()
	return NewPodmanService(&cfg.Container)
}

// NewPodmanService 创建新的 Podman 服务（依赖注入）
func NewPodmanService(cfg *appconfig.ContainerServiceConfig) *PodmanService {
	return &PodmanService{
		config: cfg,
	}
}

// NewDefaultContainerOperator 创建容器操作器（默认，内部获取依赖）
func NewDefaultContainerOperator() ContainerOperator {
	return NewDefaultPodmanService()
}

// NewContainerOperator 创建容器操作器（依赖注入）
func NewContainerOperator(cfg *appconfig.ContainerServiceConfig) ContainerOperator {
	return NewPodmanService(cfg)
}

// Start 启动容器服务
func (s *PodmanService) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 1. 检查容器运行时是否安装
	if !s.isContainerRuntimeInstalled() {
		logger.Warnf(ctx, "Container runtime 未安装，尝试自动安装...")

		// 尝试自动安装
		if err := s.installContainerRuntime(ctx); err != nil {
			return fmt.Errorf("container runtime 未安装且自动安装失败: %w\n\n"+
				"请手动安装容器运行时:\n"+
				"  macOS:   brew install podman && podman machine init\n"+
				"  Linux:   sudo apt-get install podman\n"+
				"  Windows: https://github.com/containers/podman/releases", err)
		}

		// 安装成功，再次检查
		if !s.isContainerRuntimeInstalled() {
			return fmt.Errorf("container runtime 安装完成但未能正确配置，请重启终端后重试")
		}

		logger.Infof(ctx, "Container runtime 自动安装成功！")
	}

	// 2. 根据平台准备容器运行时环境
	if err := s.prepareContainerRuntimeEnvironment(); err != nil {
		return fmt.Errorf("failed to prepare container runtime environment: %w", err)
	}

	// 3. 连接到容器运行时
	if err := s.connectToContainerRuntime(); err != nil {
		return fmt.Errorf("failed to connect to container runtime: %w", err)
	}

	return nil
}

// Stop 停止容器服务
func (s *PodmanService) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	return nil
}

// IsRunning 检查容器服务是否在运行
func (s *PodmanService) IsRunning() bool {
	return s.conn != nil
}

// GetConfig 获取配置
func (s *PodmanService) GetConfig() *appconfig.ContainerServiceConfig {
	return s.config
}

// ExecCommand 在容器内执行命令
func (s *PodmanService) ExecCommand(ctx context.Context, containerName string, command []string) (string, error) {
	if s.conn == nil {
		return "", fmt.Errorf("container service not connected")
	}

	// 构建 podman exec 命令
	args := []string{"exec", containerName}
	args = append(args, command...)

	// 执行命令
	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command in container: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// CopyToContainer 复制文件到容器
func (s *PodmanService) CopyToContainer(ctx context.Context, containerName, srcPath, destPath string) error {
	if s.conn == nil {
		return fmt.Errorf("container service not connected")
	}

	// 构建 podman cp 命令
	args := []string{"cp", srcPath, fmt.Sprintf("%s:%s", containerName, destPath)}

	// 执行命令
	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file to container: %w, output: %s", err, string(output))
	}

	return nil
}

// isContainerRuntimeInstalled 检查容器运行时是否安装
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
		return fmt.Errorf("failed to check podman machine status: %w\n\n" +
			"Try running: podman machine init")
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command("podman", "machine", "start")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start podman machine: %w\n\n" +
				"Try running manually: podman machine start")
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
		return fmt.Errorf("WSL2 is not available: %w\n\n" +
			"Please enable WSL2:\n" +
			"  wsl --update\n" +
			"  wsl --install --no-distribution")
	}

	// 检查 Podman Machine 状态
	cmd = exec.Command("podman", "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check podman machine status: %w\n\n" +
			"Try running: podman machine init")
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command("podman", "machine", "start")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start podman machine: %w\n\n" +
				"Try running manually: podman machine start")
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

	// 获取连接 URI
	socketURI := s.getSocketURI()

	// 建立连接
	conn, err := bindings.NewConnection(s.ctx, socketURI)
	if err != nil {
		return fmt.Errorf("failed to connect to container runtime at %s: %w\n\n"+
			"Troubleshooting:\n"+
			"  1. Check if podman is running: podman info\n"+
			"  2. Linux: systemctl --user start podman.socket\n"+
			"  3. macOS/Windows: podman machine start", socketURI, err)
	}

	s.conn = conn
	return nil
}

// getSocketURI 获取 Socket URI
func (s *PodmanService) getSocketURI() string {
	// 如果配置中指定了 Socket，使用配置的
	if s.config.Socket != "" {
		return s.config.Socket
	}

	// 根据平台返回默认 URI
	switch runtime.GOOS {
	case "linux":
		// Linux 使用用户级 socket
		uid := os.Getenv("UID")
		if uid == "" {
			uid = "1000" // 默认值
		}
		return fmt.Sprintf("unix:///run/user/%s/podman/podman.sock", uid)
	case "darwin":
		// macOS 使用 Podman Machine 的本地 socket
		return s.getMacOSPodmanSocket()
	case "windows":
		// Windows 使用 Podman Machine 的 SSH 连接
		return "ssh://root@127.0.0.1:55190/run/podman/podman.sock"
	default:
		return "unix:///run/podman/podman.sock"
	}
}

// getMacOSPodmanSocket 获取 macOS 上 Podman Machine 的 socket 路径
func (s *PodmanService) getMacOSPodmanSocket() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warnf(s.ctx, "Failed to get home directory: %v", err)
		return ""
	}

	// Podman Machine 的 socket 符号链接路径
	socketLinkPath := filepath.Join(homeDir, ".local", "share", "containers", "podman", "machine", "podman.sock")

	// 检查符号链接是否存在
	if _, err := os.Stat(socketLinkPath); err != nil {
		logger.Warnf(s.ctx, "Podman Machine socket link not found at %s: %v", socketLinkPath, err)
		return ""
	}

	// 读取符号链接的实际路径
	targetPath, err := os.Readlink(socketLinkPath)
	if err != nil {
		logger.Warnf(s.ctx, "Failed to read socket link: %v", err)
		return ""
	}

	// 验证目标文件是否存在
	if _, err := os.Stat(targetPath); err != nil {
		logger.Warnf(s.ctx, "Socket target file not found: %s, error: %v", targetPath, err)
		return ""
	}

	return "unix://" + targetPath
}

// 容器管理方法

// ListContainers 列出所有容器
func (s *PodmanService) ListContainers(ctx context.Context) ([]entities.ListContainer, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("container service is not running")
	}

	containers, err := containers.List(s.conn, &containers.ListOptions{All: new(bool)})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// RunContainer 运行容器
func (s *PodmanService) RunContainer(ctx context.Context, image, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 创建容器
	logger.Infof(ctx, "Creating container: %s", name)
	spec := specgen.NewSpecGenerator(image, false)
	spec.Name = name

	createResponse, err := containers.CreateWithSpec(s.conn, spec, nil)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// 启动容器
	logger.Infof(ctx, "Starting container: %s", name)
	err = containers.Start(s.conn, createResponse.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	logger.Infof(ctx, "Container %s started successfully", name)
	return nil
}

// RunContainerWithMount 运行容器并挂载目录
func (s *PodmanService) RunContainerWithMount(ctx context.Context, image, name, hostPath, containerPath string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 使用 podman 命令行工具运行容器并挂载目录
	// podman run 会自动处理镜像拉取
	logger.Infof(ctx, "Creating container with mount: %s", name)
	cmd := exec.Command("podman", "run", "-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:%s", hostPath, containerPath),
		"-e", "TZ=Asia/Shanghai", // 设置时区
		image,
		"tail", "-f", "/dev/null") // 保持容器运行

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with mount: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s", name, hostPath, containerPath)
	return nil
}

// RunContainerWithCommand 运行容器并挂载目录，使用指定命令作为主进程
func (s *PodmanService) RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, envVars ...string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 使用 podman 命令行工具运行容器并挂载目录，使用指定命令作为主进程
	logger.Infof(ctx, "Creating container with mount and command: %s", name)

	// 构建命令参数
	args := []string{"run", "-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:%s", hostPath, containerPath),
		"-e", "TZ=Asia/Shanghai"} // 设置时区

	// 添加环境变量
	for _, envVar := range envVars {
		args = append(args, "-e", envVar)
	}

	// 添加镜像和命令
	args = append(args, image)
	args = append(args, command...)

	cmd := exec.Command("podman", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with command: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s, command %v, and env vars %v", name, hostPath, containerPath, command, envVars)
	return nil
}

// IsContainerRunning 检查容器是否正在运行
func (s *PodmanService) IsContainerRunning(ctx context.Context, name string) (bool, error) {
	if !s.IsRunning() {
		return false, fmt.Errorf("container service is not running")
	}

	// 查找运行中的容器
	containerList, err := containers.List(s.conn, &containers.ListOptions{
		Filters: map[string][]string{
			"name": {name},
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to find container: %w", err)
	}

	if len(containerList) == 0 {
		return false, nil // 容器不存在
	}

	// 检查容器状态
	return containerList[0].State == "running", nil
}

// StartContainer 启动已存在的容器
func (s *PodmanService) StartContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 查找容器（包括已停止的）
	all := true
	containerList, err := containers.List(s.conn, &containers.ListOptions{
		All: &all,
		Filters: map[string][]string{
			"name": {name},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find container: %w", err)
	}

	if len(containerList) == 0 {
		return fmt.Errorf("container %s not found", name)
	}

	// 启动容器
	err = containers.Start(s.conn, containerList[0].ID, nil)
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	logger.Infof(ctx, "Container %s started successfully", name)
	return nil
}

// StopContainer 停止容器
func (s *PodmanService) StopContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 查找容器
	containerList, err := containers.List(s.conn, &containers.ListOptions{
		Filters: map[string][]string{
			"name": {name},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find container: %w", err)
	}

	if len(containerList) == 0 {
		return fmt.Errorf("container %s not found", name)
	}

	// 停止容器
	timeout := uint(10)
	err = containers.Stop(s.conn, containerList[0].ID, &containers.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	logger.Infof(ctx, "Container %s stopped successfully", name)
	return nil
}

// RemoveContainer 删除容器
func (s *PodmanService) RemoveContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 查找容器（包括已停止的容器）
	all := true
	containerList, err := containers.List(s.conn, &containers.ListOptions{
		All: &all,
		Filters: map[string][]string{
			"name": {name},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to find container: %w", err)
	}

	if len(containerList) == 0 {
		// 容器不存在，这是正常情况，不需要报错
		logger.Infof(ctx, "Container %s not found, nothing to remove", name)
		return nil
	}

	// 删除容器（强制删除，即使正在运行）
	force := true
	_, err = containers.Remove(s.conn, containerList[0].ID, &containers.RemoveOptions{
		Force: &force,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	logger.Infof(ctx, "Container %s removed successfully (forced)", name)
	return nil
}

// ListImages 列出所有镜像
func (s *PodmanService) ListImages(ctx context.Context) ([]*entities.ImageSummary, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("container service is not running")
	}

	images, err := images.List(s.conn, &images.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return images, nil
}
