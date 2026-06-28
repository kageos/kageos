package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/buildtrace"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/sdkmodule"
	"github.com/kageos/kageos/pkg/sourcepolicy"
	"golang.org/x/mod/modfile"
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
	if opts == nil {
		opts = &BuildOpts{}
	}

	// 设置默认值（目标固定为 Linux，架构与当前机器一致：本地 Mac 即 Mac 架构，线上 Linux 即服务器架构）
	platform := "linux/" + runtime.GOARCH
	if opts.SourceDir == "" {
		opts.SourceDir = filepath.Join(b.workDir, "namespace", user, app, "code", "cmd", "app")
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(b.workDir, "namespace", user, app, "workplace", "bin", "releases")
	}

	// 生成版本号（如果未提供）
	version := opts.Version
	if version == "" {
		version = b.generateVersion(user, app, opts.OutputDir)
	}

	// 设置构建信息
	opts.User = user
	opts.App = app
	opts.Version = version

	validateSpan := buildtrace.Start(ctx, "builder.validate_go_source", buildtrace.String("source_dir", opts.SourceDir))
	if err := sourcepolicy.ValidateAppGoSourceDir(opts.SourceDir); err != nil {
		validateSpan.Finish(err)
		return nil, err
	}
	validateSpan.Finish(nil)

	// opts.OutputDir 和 opts.SourceDir 都是绝对路径
	// 确保输出目录存在（必须在编译前创建，否则 go build 可能创建错误的目录）
	mkdirSpan := buildtrace.Start(ctx, "builder.ensure_output_dir", buildtrace.String("output_dir", opts.OutputDir))
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		mkdirSpan.Finish(err)
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	mkdirSpan.Finish(nil)

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

	//logger.Infof(ctx, "Building app: %s/%s, version: %s", user, app, version)
	//logger.Infof(ctx, "Source: %s", opts.SourceDir)
	//logger.Infof(ctx, "Output: %s", binaryPath)

	moduleSpan := buildtrace.Start(ctx, "builder.find_go_module_root", buildtrace.String("source_dir", opts.SourceDir))
	moduleRoot, err := findGoModuleRoot(opts.SourceDir)
	if err != nil {
		moduleSpan.Finish(err)
		return nil, err
	}
	moduleSpan.Finish(nil)

	if err := b.runGoGetLatestSDK(ctx, moduleRoot); err != nil {
		return nil, err
	}

	// 再执行 go mod tidy 清理依赖图
	if err := b.runGoModTidy(ctx, moduleRoot); err != nil {
		logger.Warnf(ctx, "go mod tidy failed, continuing with build: %v", err)
		return nil, err
	}

	// 构建 Go 命令
	cmd := b.buildGoCommand(ctx, moduleRoot, opts.SourceDir, binaryPath, platform, opts)

	// 执行编译
	buildSpan := buildtrace.Start(ctx, "builder.go_build",
		buildtrace.String("module_root", moduleRoot),
		buildtrace.String("source_dir", opts.SourceDir),
		buildtrace.String("output_path", binaryPath),
		buildtrace.String("platform", platform),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		buildErr := fmt.Errorf("go build failed: %w, output: %s", err, string(output))
		buildSpan.Finish(buildErr)
		return nil, buildErr
	}
	buildSpan.Finish(nil)

	// 获取文件信息
	statSpan := buildtrace.Start(ctx, "builder.stat_binary", buildtrace.String("binary_path", binaryPath))
	fileInfo, err := os.Stat(binaryPath)
	if err != nil {
		statSpan.Finish(err)
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	statSpan.Finish(nil)

	//logger.Infof(ctx, "Build successful: %s (size: %d bytes)", binaryName, fileInfo.Size())

	return &BuildResult{
		Version:    version,
		BinaryPath: binaryPath,
		BuildTime:  time.Now(),
		Platform:   platform,
		Size:       fileInfo.Size(),
	}, nil
}

// buildGoCommand 构建 Go 编译命令
// platform 固定为 linux/当前架构，由 Build 内自动设置
func (b *Builder) buildGoCommand(ctx context.Context, moduleRoot, sourceDir, outputPath, platform string, opts *BuildOpts) *exec.Cmd {
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		parts = []string{"linux", runtime.GOARCH}
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

	packageArg := sourceDir
	if rel, err := filepath.Rel(moduleRoot, sourceDir); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		packageArg = "./" + filepath.ToSlash(rel)
	} else if rel == "." {
		packageArg = "."
	}
	args = append(args, packageArg)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = moduleRoot
	cmd.Env = env

	return cmd
}

// buildLdFlags 构建链接参数，注入 user、app、version 信息
func (b *Builder) buildLdFlags(opts *BuildOpts) []string {
	var ldFlags []string

	// 添加用户自定义的 LdFlags
	ldFlags = append(ldFlags, opts.LdFlags...)

	// 为 SDK 应用注入构建信息到 env 包
	importPath := sdkmodule.AgentAppImport("env")
	ldFlags = append(ldFlags, fmt.Sprintf("-X %s.User=%s", importPath, opts.User))
	ldFlags = append(ldFlags, fmt.Sprintf("-X %s.App=%s", importPath, opts.App))
	ldFlags = append(ldFlags, fmt.Sprintf("-X %s.Version=%s", importPath, opts.Version))

	return ldFlags
}

func (b *Builder) runGoGetLatestSDK(ctx context.Context, moduleRoot string) error {
	span := buildtrace.Start(ctx, "builder.go_get_sdk",
		buildtrace.String("module_root", moduleRoot),
		buildtrace.String("module", sdkmodule.ModulePath),
		buildtrace.String("version_query", sdkmodule.LatestVersionQuery),
	)
	goModPath := filepath.Join(moduleRoot, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		wrapped := fmt.Errorf("failed to read go.mod before sdk sync: %w", err)
		span.Finish(wrapped)
		return wrapped
	}
	if goModHasSDKReplace(goModPath, data) {
		logger.Infof(ctx, "go.mod has local/custom %s replace, skip go get @%s", sdkmodule.ModulePath, sdkmodule.LatestVersionQuery)
		span.Finish(nil)
		return nil
	}

	cmd := exec.CommandContext(ctx, "go", "get", sdkmodule.ModulePath+"@"+sdkmodule.LatestVersionQuery)
	cmd.Dir = moduleRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		wrapped := fmt.Errorf("go get %s@%s failed: %w, output: %s", sdkmodule.ModulePath, sdkmodule.LatestVersionQuery, err, string(output))
		span.Finish(wrapped)
		return wrapped
	}
	span.Finish(nil)
	return nil
}

func goModHasSDKReplace(filename string, data []byte) bool {
	file, err := modfile.Parse(filename, data, nil)
	if err != nil {
		return false
	}
	for _, replace := range file.Replace {
		if replace.Old.Path == sdkmodule.ModulePath {
			return true
		}
	}
	return false
}

// runGoModTidy 执行 go mod tidy
func (b *Builder) runGoModTidy(ctx context.Context, moduleRoot string) error {
	//logger.Infof(ctx, "Running go mod tidy in: %s", sourceDir)

	span := buildtrace.Start(ctx, "builder.go_mod_tidy", buildtrace.String("module_root", moduleRoot))
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = moduleRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		wrapped := fmt.Errorf("go mod tidy failed: %w, output: %s", err, string(output))
		span.Finish(wrapped)
		return wrapped
	}

	span.Finish(nil)
	//logger.Infof(ctx, "go mod tidy completed successfully")
	return nil
}

func findGoModuleRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve source directory: %w", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("failed to stat source directory: %w", err)
	}
	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat go.mod: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found for source directory %s", startDir)
		}
		dir = parent
	}
}

// generateVersion 生成版本号
func (b *Builder) generateVersion(user, app, releasesDir string) string {
	// 查找现有版本
	if releasesDir == "" {
		releasesDir = filepath.Join(b.workDir, "namespace", user, app, "workplace", "bin", "releases")
	}

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
