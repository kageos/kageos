package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

// ContainerInfo is the runtime-neutral subset of `podman ps` output that upper
// layers need for app lifecycle reconciliation.
type ContainerInfo struct {
	ID     string
	Names  []string
	State  string
	Exited bool
}

// ImageInfo is the runtime-neutral subset of `podman images` output.
type ImageInfo struct {
	ID         string
	Repository string
	Tag        string
}

// ContainerOperator 容器操作接口
type ContainerOperator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	RunContainer(ctx context.Context, image, name string) error
	RunContainerWithMount(ctx context.Context, image, name, hostPath, containerPath string) error
	RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, secrets []RuntimeSecret, envVars ...string) error
	CreateSecret(ctx context.Context, secret RuntimeSecret) error
	RemoveSecret(ctx context.Context, name string) error
	IsContainerRunning(ctx context.Context, name string) (bool, error)
	StartContainer(ctx context.Context, name string) error
	StopContainer(ctx context.Context, name string) error
	RemoveContainer(ctx context.Context, name string) error
	ListImages(ctx context.Context) ([]ImageInfo, error)
	ExecCommand(ctx context.Context, containerName string, command []string) (string, error)
	CopyToContainer(ctx context.Context, containerName, srcPath, destPath string) error
}

// LSM 类型常量：与宿主机内核安全模块对应，供后续安全策略（如防删 code/workplace）选用其一。
const (
	LSMNone     = "none"
	LSMAppArmor = "apparmor"
	LSMSELinux  = "selinux"
)

// PodmanService Podman 容器服务实现
type PodmanService struct {
	ctx       context.Context
	cancel    context.CancelFunc
	config    *appconfig.ContainerServiceConfig
	connected bool

	// detectedLSM：启动时检测到的宿主机 LSM，仅设置一次，后续起容器时只启用该种（与配置配合）。
	// 用于：runtime 与容器同机时读 /sys；Mac/Win 时起临时容器探测。见 detectLSMOnce。
	detectedLSM string
}

// NewDefaultPodmanService 创建新的 Podman 服务（默认，内部获取依赖）
func NewDefaultPodmanService() *PodmanService {
	cfg := appconfig.GetAppRuntimeConfig()
	return NewPodmanService(&cfg.Container)
}

// NewPodmanService 创建新的 Podman 服务（依赖注入）
func NewPodmanService(cfg *appconfig.ContainerServiceConfig) *PodmanService {
	if cfg == nil {
		cfg = &appconfig.ContainerServiceConfig{}
	}
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

	// 4. LSM 检测（仅启动时一次，结果缓存供后续安全模块使用）
	// 背景：容器内代码为 AI 生成且可能引用会删除文件的第三方包，需在内核层限制删除 code/workplace。
	// 宿主机可能是 AppArmor（如 Ubuntu）或 SELinux（如 Fedora CoreOS），只启用检测到的那一种。
	s.detectedLSM = s.detectLSMOnce(ctx)
	logger.Infof(ctx, "[LSM] 检测结果: %s (配置 lsm_mode=%s)", s.detectedLSM, s.config.GetLSMMode())

	return nil
}

// Stop 停止容器服务
func (s *PodmanService) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	s.connected = false

	return nil
}

// IsRunning 检查容器服务是否在运行
func (s *PodmanService) IsRunning() bool {
	return s.connected
}

// GetConfig 获取配置
func (s *PodmanService) GetConfig() *appconfig.ContainerServiceConfig {
	return s.config
}

// GetDetectedLSM 返回启动时检测到的 LSM 类型（apparmor / selinux / none），后续起容器时只启用该种。
func (s *PodmanService) GetDetectedLSM() string {
	if s.detectedLSM != "" {
		return s.detectedLSM
	}
	return LSMNone
}

// detectLSMOnce 在 runtime 启动时执行一次：若配置为 auto 则检测宿主机 LSM，否则使用配置值。
// 同机 Linux 读 /sys/kernel/security/lsm；Mac/Win 无法直接读宿主机，通过起临时容器执行 cat /proc/self/attr/current 推断。
func (s *PodmanService) detectLSMOnce(ctx context.Context) string {
	mode := s.config.GetLSMMode()
	switch mode {
	case "apparmor", "selinux", "none":
		return mode
	}
	// auto：实际检测
	if runtime.GOOS == "linux" {
		return s.detectLSMFromHost(ctx)
	}
	return s.detectLSMFromProbeContainer(ctx)
}

// detectLSMFromHost 在 Linux 上直接读宿主机 LSM 列表（runtime 与容器同机时使用）。
func (s *PodmanService) detectLSMFromHost(ctx context.Context) string {
	const lsmPath = "/sys/kernel/security/lsm"
	data, err := os.ReadFile(lsmPath)
	if err != nil {
		logger.Warnf(ctx, "[LSM] 读取 %s 失败: %v，视为 none", lsmPath, err)
		return LSMNone
	}
	lsm := strings.TrimSpace(string(data))
	// 同时存在时优先用 apparmor（策略更易维护）；仅 selinux 时用 selinux。
	if strings.Contains(lsm, "apparmor") {
		return LSMAppArmor
	}
	if strings.Contains(lsm, "selinux") {
		return LSMSELinux
	}
	logger.Infof(ctx, "[LSM] 宿主机 lsm=%s，未包含 apparmor/selinux", lsm)
	return LSMNone
}

// detectLSMFromProbeContainer 在 Mac/Windows 上起临时容器，用容器内 /proc/self/attr/current 推断宿主机 LSM。
// 临时容器与后续用户容器同处 Podman VM，所见 LSM 一致。
func (s *PodmanService) detectLSMFromProbeContainer(ctx context.Context) string {
	image := s.config.GetBaseImage()
	// 使用本项目运行镜像执行一条命令并退出，避免引入额外镜像依赖
	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs("run", "--rm", image, "cat", "/proc/self/attr/current")...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[LSM] 临时容器探测失败 (image=%s): %v，输出: %s，视为 none", image, err, string(out))
		return LSMNone
	}
	current := strings.TrimSpace(string(out))
	// SELinux 上下文形如 system_u:system_r:container_t:s0:c952,c991；AppArmor 为单一 token 如 unconfined、docker-default
	if strings.Count(current, ":") >= 2 && (strings.Contains(current, "container_t") || strings.HasPrefix(current, "system_u:")) {
		return LSMSELinux
	}
	if current == "unconfined" || current == "" {
		// 无法区分“无 LSM”与“AppArmor 但未挂 profile”，保守返回 none；后续若需可再试 /sys/kernel/security/lsm。
		return LSMNone
	}
	// 单 token 且非 unconfined，视为 AppArmor profile 名
	return LSMAppArmor
}

// isAppArmorProfileLoaded 检查宿主机上指定 AppArmor profile 是否已加载（读 /sys/kernel/security/apparmor/profiles）。
// 若未加载或读失败则返回 false，起容器时不加 apparmor 选项，避免阻塞启动。
func isAppArmorProfileLoaded(profile string) bool {
	if profile == "" {
		return false
	}
	const profilesPath = "/sys/kernel/security/apparmor/profiles"
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), profile)
}

// ExecCommand 在容器内执行命令
func (s *PodmanService) ExecCommand(ctx context.Context, containerName string, command []string) (string, error) {
	if !s.IsRunning() {
		return "", fmt.Errorf("container service not connected")
	}

	// 构建 podman exec 命令
	args := []string{"exec", containerName}
	args = append(args, command...)

	// 执行命令
	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs(args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command in container: %w, output: %s", err, string(output))
	}

	return string(output), nil
}

// CopyToContainer 复制文件到容器
func (s *PodmanService) CopyToContainer(ctx context.Context, containerName, srcPath, destPath string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service not connected")
	}

	// 构建 podman cp 命令
	args := []string{"cp", srcPath, fmt.Sprintf("%s:%s", containerName, destPath)}

	// 执行命令
	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs(args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file to container: %w, output: %s", err, string(output))
	}

	return nil
}

// commandPath resolves tools that macOS GUI applications commonly cannot find
// because they do not inherit the user's login-shell PATH. Returning an
// absolute path also makes all subsequent child processes use the same binary.
func commandPath(name string) (string, error) {
	return commandPathForOS(name, runtime.GOOS, exec.LookPath, os.Stat)
}

func commandPathForOS(
	name string,
	goos string,
	lookPath func(string) (string, error),
	stat func(string) (os.FileInfo, error),
) (string, error) {
	path, lookErr := lookPath(name)
	if lookErr == nil {
		return path, nil
	}
	if goos != "darwin" {
		return "", lookErr
	}

	var candidates []string
	switch name {
	case "podman":
		candidates = []string{
			"/opt/podman/bin/podman",
			"/opt/homebrew/bin/podman",
			"/usr/local/bin/podman",
		}
	case "brew":
		candidates = []string{
			"/opt/homebrew/bin/brew",
			"/usr/local/bin/brew",
		}
	default:
		return "", lookErr
	}

	for _, candidate := range candidates {
		info, err := stat(candidate)
		if err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", lookErr
}

func podmanPath() string {
	path, err := commandPath("podman")
	if err != nil {
		return "podman"
	}
	return path
}

// isContainerRuntimeInstalled 检查容器运行时是否安装
func (s *PodmanService) isContainerRuntimeInstalled() bool {
	_, err := commandPath("podman")
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
	brew, err := commandPath("brew")
	if err != nil {
		logger.Errorf(ctx, "未找到 Homebrew，请先安装 Homebrew:")
		logger.Infof(ctx, "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
		return fmt.Errorf("homebrew 未安装")
	}

	// 使用 Homebrew 安装 Podman
	logger.Infof(ctx, "使用 Homebrew 安装 Podman...")
	cmd := exec.CommandContext(ctx, brew, "install", "podman")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 podman 失败: %w", err)
	}

	logger.Infof(ctx, "Podman 安装成功！")

	// 初始化 Podman Machine
	logger.Infof(ctx, "初始化 Podman Machine...")
	cmd = exec.CommandContext(ctx, podmanPath(), "machine", "init")
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
	cmd = exec.CommandContext(ctx, podmanPath(), "machine", "start")
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

			cmd = exec.Command(podmanPath(), "system", "service", "--time=0", "unix:///run/user/"+os.Getenv("UID")+"/podman/podman.sock")
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
	cmd := exec.Command(podmanPath(), "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check podman machine status: %w\n\n"+
			"Try running: podman machine init", err)
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command(podmanPath(), "machine", "start")
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
	cmd = exec.Command(podmanPath(), "machine", "list", "--format", "{{.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check podman machine status: %w\n\n"+
			"Try running: podman machine init", err)
	}

	running := strings.TrimSpace(string(output))
	if running != "true" {
		// Machine 未运行，启动它
		logger.Infof(s.ctx, "Starting Podman Machine...")
		cmd = exec.Command(podmanPath(), "machine", "start")
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
	cmd := exec.CommandContext(s.ctx, podmanPath(), s.podmanArgs("info", "--format", "json")...)
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

func (s *PodmanService) podmanArgs(args ...string) []string {
	socket := strings.TrimSpace(s.config.GetSocket())
	if socket == "" {
		return args
	}

	out := make([]string, 0, len(args)+2)
	out = append(out, "--url", socket)
	out = append(out, args...)
	return out
}

func (s *PodmanService) runPodman(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs(args...)...)
	return cmd.CombinedOutput()
}

// CreateSecret creates or replaces a Podman-managed secret. Remove+create is
// used instead of `secret create --replace` so the deployment remains
// compatible with the Podman version shipped by Debian Bookworm. Secret data
// is delivered over stdin so it never appears in argv, the process
// environment, or container inspection output.
func (s *PodmanService) CreateSecret(ctx context.Context, secret RuntimeSecret) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "secret", "rm", secret.Name)
	if err != nil && !isPodmanNotFoundOutput(string(output)) {
		return fmt.Errorf("failed to replace runtime secret %s: %w, output: %s", secret.Name, err, string(output))
	}

	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs("secret", "create", secret.Name, "-")...)
	cmd.Stdin = bytes.NewReader(secret.Data)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create runtime secret %s: %w", secret.Name, err)
	}

	logger.Infof(ctx, "Runtime secret %s created successfully", secret.Name)
	return nil
}

// RemoveSecret removes a Podman-managed secret and treats an already missing
// secret as success so app-version cleanup remains idempotent.
func (s *PodmanService) RemoveSecret(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "secret", "rm", name)
	if err != nil {
		if isPodmanNotFoundOutput(string(output)) {
			logger.Infof(ctx, "Runtime secret %s not found, nothing to remove", name)
			return nil
		}
		return fmt.Errorf("failed to remove runtime secret %s: %w, output: %s", name, err, string(output))
	}

	logger.Infof(ctx, "Runtime secret %s removed successfully", name)
	return nil
}

func isPodmanNotFoundOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no such container") ||
		strings.Contains(lower, "no such secret") ||
		strings.Contains(lower, "container does not exist") ||
		strings.Contains(lower, "no container with name or id") ||
		strings.Contains(lower, "no secret with name or id") ||
		strings.Contains(lower, "not found")
}

func decodePodmanJSONObjectList(output []byte) ([]map[string]json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var objects []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &objects); err != nil {
			return nil, err
		}
		return objects, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var object map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &object); err != nil {
			return nil, err
		}
		return []map[string]json.RawMessage{object}, nil
	}

	return nil, fmt.Errorf("unexpected JSON output: %q", trimmed)
}

func podmanJSONField(object map[string]json.RawMessage, names ...string) (json.RawMessage, bool) {
	for _, name := range names {
		if raw, ok := object[name]; ok {
			return raw, true
		}
	}
	for key, raw := range object {
		for _, name := range names {
			if strings.EqualFold(key, name) {
				return raw, true
			}
		}
	}
	return nil, false
}

func podmanJSONString(object map[string]json.RawMessage, names ...string) string {
	raw, ok := podmanJSONField(object, names...)
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return strings.TrimSpace(value)
	}
	return ""
}

func podmanJSONBool(object map[string]json.RawMessage, names ...string) (bool, bool) {
	raw, ok := podmanJSONField(object, names...)
	if !ok {
		return false, false
	}

	var boolValue bool
	if err := json.Unmarshal(raw, &boolValue); err == nil {
		return boolValue, true
	}

	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		switch strings.ToLower(strings.TrimSpace(stringValue)) {
		case "true", "running":
			return true, true
		case "false", "exited", "stopped", "created":
			return false, true
		}
	}

	return false, false
}

func podmanJSONNames(object map[string]json.RawMessage) []string {
	raw, ok := podmanJSONField(object, "Names", "names")
	if !ok {
		return nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		names := make([]string, 0, len(values))
		for _, name := range values {
			name = strings.TrimSpace(name)
			if name != "" {
				names = append(names, name)
			}
		}
		return names
	}

	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return splitContainerNames(value)
	}

	return nil
}

func parsePodmanContainerListJSON(output []byte) ([]ContainerInfo, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return nil, err
	}

	containers := make([]ContainerInfo, 0, len(objects))
	for _, object := range objects {
		state := strings.ToLower(podmanJSONString(object, "State", "state", "Status", "status"))
		exited := true
		if state != "" {
			exited = state != "running"
		} else if value, ok := podmanJSONBool(object, "Exited", "exited"); ok {
			exited = value
		}

		containers = append(containers, ContainerInfo{
			ID:     podmanJSONString(object, "ID", "Id", "id"),
			Names:  podmanJSONNames(object),
			State:  state,
			Exited: exited,
		})
	}

	return containers, nil
}

func parsePodmanImageListJSON(output []byte) ([]ImageInfo, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return nil, err
	}

	images := make([]ImageInfo, 0, len(objects))
	for _, object := range objects {
		repository := podmanJSONString(object, "Repository", "repository")
		tag := podmanJSONString(object, "Tag", "tag")
		if repository == "" && tag == "" {
			repository, tag = splitImageRepositoryTag(firstPodmanJSONName(object))
		}

		images = append(images, ImageInfo{
			ID:         podmanJSONString(object, "ID", "Id", "id"),
			Repository: repository,
			Tag:        tag,
		})
	}

	return images, nil
}

func parsePodmanInspectRunningJSON(output []byte) (bool, error) {
	objects, err := decodePodmanJSONObjectList(output)
	if err != nil {
		return false, err
	}
	if len(objects) == 0 {
		return false, nil
	}

	stateRaw, ok := podmanJSONField(objects[0], "State", "state")
	if !ok {
		return false, nil
	}

	var stateObject map[string]json.RawMessage
	if err := json.Unmarshal(stateRaw, &stateObject); err != nil {
		return false, err
	}
	running, _ := podmanJSONBool(stateObject, "Running", "running")
	return running, nil
}

// 容器管理方法

// ListContainers 列出所有容器（包括已停止的）
func (s *PodmanService) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "ps", "-a", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w, output: %s", err, string(output))
	}

	containers, err := parsePodmanContainerListJSON(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse container list JSON: %w, output: %s", err, string(output))
	}
	return containers, nil
}

// RunContainer 运行容器
func (s *PodmanService) RunContainer(ctx context.Context, image, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	logger.Infof(ctx, "Creating and starting container: %s", name)
	output, err := s.runPodman(ctx, "run", "-d", "--name", name, image)
	if err != nil {
		return fmt.Errorf("failed to run container: %w, output: %s", err, string(output))
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
	logger.Infof(ctx, "Creating container with mount: %s", name)
	args := podmanRunBaseArgs(name, hostPath, containerPath, s.config.GetNetworkMode())
	args = append(args,
		image,
		"tail", "-f", "/dev/null", // 保持容器运行
	)

	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs(args...)...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with mount: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s", name, hostPath, containerPath)
	return nil
}

// RunContainerWithCommand 运行容器并挂载目录，使用指定命令作为主进程
func (s *PodmanService) RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, secrets []RuntimeSecret, envVars ...string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 使用 podman 命令行工具运行容器并挂载目录，使用指定命令作为主进程
	logger.Infof(ctx, "Creating container with mount and command: %s", name)

	// 构建命令参数
	args := podmanRunBaseArgs(name, hostPath, containerPath, s.config.GetNetworkMode())

	// 按 LSM 检测结果施加内核级安全策略（禁止容器内删除 code/workplace）
	switch s.GetDetectedLSM() {
	case LSMAppArmor:
		if profile := s.config.GetAppArmorProfile(); profile != "" {
			if isAppArmorProfileLoaded(profile) {
				args = append(args, "--security-opt", "apparmor="+profile)
				logger.Infof(ctx, "[LSM] AppArmor profile=%s 已应用到容器 %s", profile, name)
			} else {
				// profile 未加载时不加选项，避免 failed to start container: profile specified but not loaded；静默继续运行
				logger.Infof(ctx, "[LSM] AppArmor profile=%s 未加载，跳过安全选项，容器 %s 正常启动", profile, name)
			}
		} else {
			logger.Warnf(ctx, "[LSM] 检测到 AppArmor 但未配置 apparmor_profile，容器 %s 无内核级防删", name)
		}
	case LSMSELinux:
		// SELinux 防护依赖宿主机上安装的策略模块 + 目录标签（deploy/security/selinux/install.sh），
		// 挂载时不加 :z/:Z 以保持 kageos_data_t 标签不被覆盖。
		logger.Infof(ctx, "[LSM] SELinux 环境，容器 %s 依赖宿主机策略模块与目录标签防删", name)
	default:
		logger.Warnf(ctx, "[LSM] 未检测到可用 LSM，容器 %s 无内核级防删保护", name)
	}

	// Mount runtime-managed secrets. Only secret names and mount targets enter
	// the container command; secret contents stay in Podman's secret store.
	for _, secret := range secrets {
		args = append(args, "--secret", podmanSecretMountSpec(secret))
	}

	// 添加环境变量
	for _, envVar := range envVars {
		args = append(args, "-e", envVar)
	}

	// 添加镜像和命令
	args = append(args, image)
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, podmanPath(), s.podmanArgs(args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with command: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s, command %v, env keys %v, and secret targets %v", name, hostPath, containerPath, command, envVarNames(envVars), runtimeSecretTargets(secrets))
	return nil
}

func podmanSecretMountSpec(secret RuntimeSecret) string {
	return fmt.Sprintf("source=%s,target=%s,type=mount,mode=0400", secret.Name, secret.Target)
}

func runtimeSecretTargets(secrets []RuntimeSecret) []string {
	targets := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		targets = append(targets, secret.Target)
	}
	return targets
}

func podmanRunBaseArgs(name, hostPath, containerPath string, networkMode string) []string {
	args := []string{
		"run", "-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:%s", hostPath, containerPath),
		"-e", "TZ=" + runtimeTimezone(),
	}
	if networkMode = strings.TrimSpace(networkMode); networkMode != "" {
		args = append(args, "--network", networkMode)
	}
	return args
}

func runtimeTimezone() string {
	timezone := strings.TrimSpace(os.Getenv("TZ"))
	if timezone == "" {
		return "Asia/Shanghai"
	}
	return timezone
}

func envVarNames(envVars []string) []string {
	names := make([]string, 0, len(envVars))
	for _, envVar := range envVars {
		name, _, _ := strings.Cut(envVar, "=")
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// IsContainerRunning 检查容器是否正在运行
func (s *PodmanService) IsContainerRunning(ctx context.Context, name string) (bool, error) {
	if !s.IsRunning() {
		return false, fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "container", "inspect", "--format", "json", name)
	if err != nil {
		if isPodmanNotFoundOutput(string(output)) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect container: %w, output: %s", err, string(output))
	}

	running, err := parsePodmanInspectRunningJSON(output)
	if err != nil {
		return false, fmt.Errorf("failed to parse container inspect JSON: %w, output: %s", err, string(output))
	}
	return running, nil
}

// StartContainer 启动已存在的容器
func (s *PodmanService) StartContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	running, err := s.IsContainerRunning(ctx, name)
	if err != nil {
		return err
	}
	if running {
		return nil
	}

	output, err := s.runPodman(ctx, "start", name)
	if err != nil {
		if isPodmanNotFoundOutput(string(output)) {
			return fmt.Errorf("container %s not found", name)
		}
		return fmt.Errorf("failed to start container: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully", name)
	return nil
}

// StopContainer 停止容器
// 若容器已不存在或已处于退出状态，视为成功（不报错）
func (s *PodmanService) StopContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	running, err := s.IsContainerRunning(ctx, name)
	if err != nil {
		return err
	}
	if !running {
		logger.Infof(ctx, "Container %s already stopped or missing, skipping stop", name)
		return nil
	}

	output, err := s.runPodman(ctx, "stop", "--time", "10", name)
	if err != nil {
		if isPodmanNotFoundOutput(string(output)) {
			logger.Infof(ctx, "Container %s already removed, skipping stop", name)
			return nil
		}
		return fmt.Errorf("failed to stop container: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s stopped successfully", name)
	return nil
}

// RemoveContainer 删除容器
func (s *PodmanService) RemoveContainer(ctx context.Context, name string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "rm", "-f", name)
	if err != nil {
		if isPodmanNotFoundOutput(string(output)) {
			logger.Infof(ctx, "Container %s not found, nothing to remove", name)
			return nil
		}
		return fmt.Errorf("failed to remove container: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s removed successfully (forced)", name)
	return nil
}

// ListImages 列出所有镜像
func (s *PodmanService) ListImages(ctx context.Context) ([]ImageInfo, error) {
	if !s.IsRunning() {
		return nil, fmt.Errorf("container service is not running")
	}

	output, err := s.runPodman(ctx, "images", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w, output: %s", err, string(output))
	}

	images, err := parsePodmanImageListJSON(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image list JSON: %w, output: %s", err, string(output))
	}
	return images, nil
}

func splitContainerNames(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	rawNames := strings.Split(value, ",")
	names := make([]string, 0, len(rawNames))
	for _, name := range rawNames {
		name = strings.TrimSpace(name)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func firstPodmanJSONName(object map[string]json.RawMessage) string {
	names := podmanJSONNames(object)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func splitImageRepositoryTag(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}

	lastSlash := strings.LastIndex(name, "/")
	lastColon := strings.LastIndex(name, ":")
	if lastColon > lastSlash {
		return name[:lastColon], name[lastColon+1:]
	}
	return name, ""
}
