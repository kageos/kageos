package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
)

func (s *PodmanService) ExecCommand(ctx context.Context, containerName string, command []string) (string, error) {
	if !s.IsRunning() {
		return "", fmt.Errorf("container service not connected")
	}

	// 构建 podman exec 命令
	args := []string{"exec", containerName}
	args = append(args, command...)

	// 执行命令
	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs(args...)...)
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
	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs(args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy file to container: %w, output: %s", err, string(output))
	}

	return nil
}

// isContainerRuntimeInstalled 检查容器运行时是否安装
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
	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs(args...)...)
	return cmd.CombinedOutput()
}

func isPodmanNotFoundOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "no such container") ||
		strings.Contains(lower, "container does not exist") ||
		strings.Contains(lower, "no container with name or id") ||
		strings.Contains(lower, "not found")
}

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

	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs(args...)...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with mount: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s", name, hostPath, containerPath)
	return nil
}

// RunContainerWithCommand 运行容器并挂载目录，使用指定命令作为主进程
func (s *PodmanService) RunContainerWithCommand(ctx context.Context, image, name, hostPath, containerPath string, command []string, envVars ...string) error {
	return s.RunContainerWithCommandAndSecrets(ctx, image, name, hostPath, containerPath, command, nil, envVars...)
}

// RunContainerWithCommandAndSecrets runs an app container with Podman-managed
// file secrets. Secret values are provided to `podman secret create` over stdin,
// mounted read-only at /run/secrets/<target>, and removed from the Podman secret
// store immediately after container creation. The container keeps its private
// mounted copy for restarts.
func (s *PodmanService) RunContainerWithCommandAndSecrets(ctx context.Context, image, name, hostPath, containerPath string, command []string, secrets []ContainerSecret, envVars ...string) error {
	if !s.IsRunning() {
		return fmt.Errorf("container service is not running")
	}

	// 使用 podman 命令行工具运行容器并挂载目录，使用指定命令作为主进程
	logger.Infof(ctx, "Creating container with mount and command: %s", name)

	// 构建命令参数
	args := podmanRunBaseArgs(name, hostPath, containerPath, s.config.GetNetworkMode())

	if err := s.createPodmanSecrets(ctx, secrets); err != nil {
		return err
	}
	defer s.removePodmanSecrets(ctx, secrets)
	for _, secret := range secrets {
		args = append(args, "--secret", podmanSecretRunOption(secret))
	}

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

	// 添加环境变量
	for _, envVar := range envVars {
		args = append(args, "-e", envVar)
	}

	// 添加镜像和命令
	args = append(args, image)
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, "podman", s.podmanArgs(args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run container with command: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "Container %s started successfully with mount %s:%s, command %v, env keys %v, and %d runtime secrets", name, hostPath, containerPath, command, envVarNames(envVars), len(secrets))
	return nil
}

func (s *PodmanService) createPodmanSecrets(ctx context.Context, secrets []ContainerSecret) error {
	created := make([]ContainerSecret, 0, len(secrets))
	for _, secret := range secrets {
		if err := validateContainerSecret(secret); err != nil {
			s.removePodmanSecrets(ctx, created)
			return err
		}

		cmd := exec.CommandContext(ctx, "podman", s.podmanArgs("secret", "create", "--replace", secret.Name, "-")...)
		cmd.Stdin = bytes.NewReader(secret.Data)
		output, err := cmd.CombinedOutput()
		if err != nil {
			s.removePodmanSecrets(ctx, created)
			return fmt.Errorf("create Podman runtime secret %s: %w, output: %s", secret.Name, err, strings.TrimSpace(string(output)))
		}
		created = append(created, secret)
	}
	return nil
}

func (s *PodmanService) removePodmanSecrets(ctx context.Context, secrets []ContainerSecret) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	for _, secret := range secrets {
		if _, err := s.runPodman(cleanupCtx, "secret", "rm", "--ignore", secret.Name); err != nil {
			logger.Warnf(cleanupCtx, "Failed to remove transient Podman runtime secret %s: %v", secret.Name, err)
		}
	}
}

func validateContainerSecret(secret ContainerSecret) error {
	name := strings.TrimSpace(secret.Name)
	target := strings.TrimSpace(secret.Target)
	if name == "" {
		return fmt.Errorf("container secret name is required")
	}
	if target == "" {
		return fmt.Errorf("container secret target is required")
	}
	if name != secret.Name || strings.ContainsAny(name, ",=/\x00") {
		return fmt.Errorf("invalid container secret name")
	}
	if target != secret.Target || strings.ContainsAny(target, ",=/\x00") {
		return fmt.Errorf("invalid container secret target")
	}
	if len(secret.Data) == 0 {
		return fmt.Errorf("container secret data is empty")
	}
	if len(secret.Data) > 512*1024 {
		return fmt.Errorf("container secret data exceeds Podman 512 KiB limit")
	}
	return nil
}

func podmanSecretRunOption(secret ContainerSecret) string {
	return fmt.Sprintf("%s,type=mount,target=%s,mode=0400", secret.Name, secret.Target)
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
