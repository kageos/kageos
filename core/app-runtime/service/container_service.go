package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

// ContainerSecret describes sensitive runtime data mounted into a container.
// Data is passed to Podman over stdin and is never placed in process arguments
// or environment variables.
type ContainerSecret struct {
	Name   string
	Target string
	Data   []byte
}

// ContainerOperator 容器操作接口
type ContainerOperator interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	ListContainers(ctx context.Context) ([]ContainerInfo, error)
	RunContainer(ctx context.Context, image, name string) error
	RunContainerWithMount(ctx context.Context, image, name, hostPath, containerPath string) error
	RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, envVars ...string) error
	RunContainerWithCommandAndSecrets(ctx context.Context, image, name, hostPath, containerPath string, command []string, secrets []ContainerSecret, envVars ...string) error
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
	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs("run", "--rm", image, "cat", "/proc/self/attr/current")...)
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
