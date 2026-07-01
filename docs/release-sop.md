# Kageos 发布 SOP

本文档记录 Kageos 正式发布的标准流程。目标是保证 SDK、平台镜像、国内镜像同步和用户 `kageos update` 入口保持一致。

## 发布入口

正式版本只通过 `kageos` 主仓的语义化 tag 发布：

```bash
git tag -a v0.1.9 -m "v0.1.9"
git push origin v0.1.9
```

不要把手动发布 `latest` 当成正式发版。`latest` 是正式版本发布成功后自动更新的移动指针。

## 标准顺序

### 1. 先发布 SDK

如果 `kageos-sdk` 有改动，必须先给 SDK 打 tag：

```bash
cd /path/to/kageos-sdk
go test ./...
git status --short --branch
git tag -a v0.2.2 -m "v0.2.2"
git push origin main v0.2.2
```

如果 SDK 没有改动，跳过这一步。

### 2. 升级主仓 SDK 依赖

当 SDK 发了新 tag 后，进入 `kageos` 主仓升级依赖：

```bash
cd /path/to/kageos
go get github.com/kageos/kageos-sdk@v0.2.2
go mod tidy
git diff -- go.mod go.sum
git commit -am "chore: bump kageos sdk to v0.2.2"
git push origin main
```

主仓的 `pkg/sdkmodule` 会从 `go.mod` 读取平台当前 SDK 版本。运行时创建或更新工作空间应用时，会把应用 `go.mod` 里的 `github.com/kageos/kageos-sdk` 升到平台当前版本，但不会降级用户已经手动使用的更高版本。

### 3. 发布前检查

在 `kageos` 主仓执行：

```bash
bash scripts/test-core-go.sh
npm --prefix web run check:architecture
npm --prefix web run type-check
npm --prefix web run test:unit -- --run
npm --prefix web run build
git diff --check
```

这些检查应与 `.github/workflows/ci.yml` 保持一致。发布前本地工作区应只包含本次发布需要的改动。

### 4. 打 Kageos 版本 tag

确认 `main` 已推送后打正式 tag：

```bash
git status --short --branch
git tag -a v0.1.9 -m "v0.1.9"
git push origin v0.1.9
```

tag 会同时触发：

- `.github/workflows/docker-release.yml`
- `.github/workflows/kagebase-release.yml`

## 发布产物

`docker-release.yml` 发布：

- `docker.io/qiayanai/kageos:<version>`
- `docker.io/qiayanai/kageos:latest`
- 阿里云 ACR 的 `kageos:<version>` 和 `kageos:latest`

`kagebase-release.yml` 发布：

- `docker.io/qiayanai/kagebase:<version>`
- `docker.io/qiayanai/kagebase:latest`
- 阿里云 ACR 的 `kagebase:<version>` 和 `kagebase:latest`

正式 release 镜像里的默认用户应用基础镜像会指向同版本的 `docker.io/qiayanai/kagebase:<version>`。

## 发布后验证

检查 GitHub Actions：

```bash
gh run list --workflow "CI" --limit 5
gh run list --workflow "Docker Release" --limit 5
gh run list --workflow "Kagebase Release" --limit 5
```

检查 Docker Hub 多架构 manifest：

```bash
docker buildx imagetools inspect docker.io/qiayanai/kageos:0.1.9
docker buildx imagetools inspect docker.io/qiayanai/kagebase:0.1.9
docker buildx imagetools inspect docker.io/qiayanai/kageos:latest
```

在国内生产机上验证用户更新：

```bash
sudo kageos update
sudo kageos status
sudo bash -lc 'source /etc/kageos-helper.env; $KAGEOS_ENGINE image inspect "$KAGEOS_IMAGE" --format "{{.Id}} {{.RepoDigests}}"'
sudo bash -lc 'source /etc/kageos-helper.env; $KAGEOS_ENGINE exec "$KAGEOS_CONTAINER_NAME" grep "github.com/kageos/kageos-sdk" /app/go.mod'
```

## 用户更新入口

`sudo kageos update` 的 helper 来自 `kageos-website/public/install-prod.sh`，不是 `kageos` 主仓。

国内安装或更新时，`--cn` 默认使用国内镜像源，不再尝试 release tarball。只有显式设置：

```dotenv
KAGEOS_CN=1
KAGEOS_CN_TARBALL=1
KAGEOS_RELEASE_VERSION=latest
```

helper 才会重新拉取 `https://kageos.com/install-prod.sh`，并把 release tarball 纳入兜底。默认发布流程不再自动生成或同步 R2 release tarball；只有手动运行 `release-archive-sync.yml` 后，release tarball 链路才可用。这个可选链路依赖：

- 官网已经发布最新 `install-prod.sh`
- R2 的 `latest.txt` 指向最新 tag
- R2 上对应架构 tarball 和 sha256 文件存在

如果只是镜像发布成功，但官网安装脚本还没有上线，线上 `kageos update` 的行为仍由旧 helper 脚本决定。

## 补救 Workflow

这些 workflow 是补救入口，不是标准发版入口：

- `registry-sync.yml`: Docker Hub 已有版本镜像，但阿里云 ACR 同步失败时使用。
- `release-archive-sync.yml`: Docker Hub 已有 `kageos:<version>`，且确实需要补发 R2 tarball 或 `latest.txt` 时手动使用。
- `docker-retag.yml`: 已有版本镜像，需要修正 Docker Hub `latest` 指针时使用。
- `dev-latest-release.yml`: 临时测试 `latest`，不代表正式版本。

## 常见风险

- 只推 `latest` 会导致版本不可追踪，线上机器也无法判断自己是否拿到同一个正式版本。
- SDK 改了但没先打 SDK tag，主镜像里的 `go.mod` 仍会固定到旧 SDK。
- 打了 `kageos` tag 后再追加提交，不会进入已经发布的镜像，必须再发下一个版本。
- 国内 ACR `latest` 可能因为同步延迟或失败没有更新，发布后必须看 workflow 日志里的目标 digest。
- `kageos update` 保留数据目录，但会重建外层容器；验证时要同时看镜像 digest、容器启动状态和容器内 `/app/go.mod`。
