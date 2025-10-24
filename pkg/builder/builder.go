package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// Builder 应用构建器
type Builder struct {
	workDir string
}

// NewBuilder 创建构建器
func NewBuilder(workDir string) *Builder {
	return &Builder{workDir: workDir}
}

// BuildOpts 编译选项
type BuildOpts struct {
	User             string            // 用户名称
	App              string            // 应用名称
	Version          string            // 版本号
	SourceDir        string            // 源代码目录
	OutputDir        string            // 输出目录
	Platform         string            // 目标平台 (linux/amd64, linux/arm64)
	BinaryNameFormat string            // 二进制文件名格式
	BuildTags        []string          // 编译标签
	LdFlags          []string          // 链接参数
	Env              map[string]string // 编译环境变量
}

// BuildResult 构建结果
type BuildResult struct {
	Version    string    // 版本号
	BinaryPath string    // 二进制文件路径
	BuildTime  time.Time // 构建时间
	Platform   string    // 目标平台
	Size       int64     // 文件大小
}

// Build 编译应用
func (b *Builder) Build(ctx context.Context, user, app string, opts *BuildOpts) (*BuildResult, error) {
	// 生成版本号（如果未提供）
	version := opts.Version
	if version == "" {
		version = b.generateVersion(user, app)
	}

	// 设置默认值
	if opts.Platform == "" {
		opts.Platform = "linux/amd64"
	}
	if opts.SourceDir == "" {
		opts.SourceDir = filepath.Join(b.workDir, "namespace", user, app, "code", "cmd", "app")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(b.workDir, "namespace", user, app, "workplace", "bin", "releases")
	}

	// 设置构建信息
	opts.User = user
	opts.App = app
	opts.Version = version

	// opts.OutputDir 和 opts.SourceDir 都是绝对路径
	// 确保输出目录存在（必须在编译前创建，否则 go build 可能创建错误的目录）
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 构建二进制文件的完整路径
	var binaryName string
	if opts.BinaryNameFormat != "" {
		// 使用配置的格式，替换占位符
		binaryName = strings.ReplaceAll(opts.BinaryNameFormat, "{user}", user)
		binaryName = strings.ReplaceAll(binaryName, "{app}", app)
		binaryName = strings.ReplaceAll(binaryName, "{version}", version)
	} else {
		// 默认格式
		binaryName = fmt.Sprintf("%s_%s_%s", user, app, version)
	}
	binaryPath := filepath.Join(opts.OutputDir, binaryName)

	logger.Infof(ctx, "Building app: %s/%s, version: %s", user, app, version)
	logger.Infof(ctx, "Source: %s", opts.SourceDir)
	logger.Infof(ctx, "Output: %s", binaryPath)
	logger.Infof(ctx, "Platform: %s", opts.Platform)

	// 先执行 go mod tidy 确保依赖是最新的
	if err := b.runGoModTidy(ctx, opts.SourceDir); err != nil {
		logger.Warnf(ctx, "go mod tidy failed, continuing with build: %v", err)
	}

	// 构建 Go 命令
	cmd := b.buildGoCommand(ctx, opts.SourceDir, binaryPath, opts)

	// 执行编译
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to build: %w, output: %s", err, string(output))
	}

	// 获取文件信息
	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	logger.Infof(ctx, "Build successful: %s (size: %d bytes)", binaryName, fileInfo.Size())

	return &BuildResult{
		Version:    version,
		BinaryPath: binaryPath,
		BuildTime:  time.Now(),
		Platform:   opts.Platform,
		Size:       fileInfo.Size(),
	}, nil
}

// buildGoCommand 构建 Go 编译命令
// 使用绝对路径避免在源代码目录下创建嵌套目录
func (b *Builder) buildGoCommand(ctx context.Context, sourceDir, outputPath string, opts *BuildOpts) *exec.Cmd {
	// 解析平台
	parts := strings.Split(opts.Platform, "/")
	if len(parts) != 2 {
		parts = []string{"linux", "amd64"}
	}
	goos, goarch := parts[0], parts[1]

	// 设置环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", goos))
	env = append(env, fmt.Sprintf("GOARCH=%s", goarch))
	// 禁用 CGO 以使用纯 Go SQLite 驱动 (modernc.org/sqlite)
	env = append(env, "CGO_ENABLED=0")

	// 添加自定义环境变量
	for k, v := range opts.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// 构建命令参数
	args := []string{"build"}

	// 添加编译标签
	if len(opts.BuildTags) > 0 {
		args = append(args, "-tags", strings.Join(opts.BuildTags, ","))
	}

	// 构建链接参数，注入 user、app、version 信息
	ldFlags := b.buildLdFlags(opts)
	if len(ldFlags) > 0 {
		args = append(args, "-ldflags", strings.Join(ldFlags, " "))
	}

	// 使用绝对路径指定输出文件
	args = append(args, "-o", outputPath)

	// 使用绝对路径指定源代码目录
	// 关键：不使用 "." 和 cmd.Dir，直接指定源代码绝对路径
	args = append(args, sourceDir)

	// 创建命令，不设置 cmd.Dir，在项目根目录执行
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = env

	return cmd
}

// buildLdFlags 构建链接参数，注入 user、app、version 信息
func (b *Builder) buildLdFlags(opts *BuildOpts) []string {
	var ldFlags []string

	// 添加用户自定义的 LdFlags
	ldFlags = append(ldFlags, opts.LdFlags...)

	// 为 SDK 应用注入构建信息到 env 包
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env.User=%s", opts.User))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env.App=%s", opts.App))
	ldFlags = append(ldFlags, fmt.Sprintf("-X github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env.Version=%s", opts.Version))

	return ldFlags
}

// runGoModTidy 执行 go mod tidy
func (b *Builder) runGoModTidy(ctx context.Context, sourceDir string) error {
	logger.Infof(ctx, "Running go mod tidy in: %s", sourceDir)

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = sourceDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %w, output: %s", err, string(output))
	}

	logger.Infof(ctx, "go mod tidy completed successfully")
	return nil
}

// generateVersion 生成版本号
func (b *Builder) generateVersion(user, app string) string {
	// 查找现有版本
	releasesDir := filepath.Join(b.workDir, "namespace", user, app, "workplace", "bin", "releases")

	maxVersion := 0
	if entries, err := os.ReadDir(releasesDir); err == nil {
		prefix := fmt.Sprintf("%s_%s_v", user, app)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), prefix) {
				// 提取版本号
				versionStr := strings.TrimPrefix(entry.Name(), prefix)
				if version, err := strconv.Atoi(versionStr); err == nil && version > maxVersion {
					maxVersion = version
				}
			}
		}
	}

	return fmt.Sprintf("v%d", maxVersion+1)
}

// ListVersions 列出所有版本
func (b *Builder) ListVersions(ctx context.Context, user, app string) ([]string, error) {
	releasesDir := filepath.Join(b.workDir, "namespace", user, app, "workplace", "bin", "releases")

	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return nil, err
	}

	var versions []string
	prefix := fmt.Sprintf("%s_%s_v", user, app)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// GetLatestVersion 获取最新版本
func (b *Builder) GetLatestVersion(ctx context.Context, user, app string) (string, error) {
	versions, err := b.ListVersions(ctx, user, app)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	// 简单返回最后一个（按文件名排序）
	return versions[len(versions)-1], nil
}
