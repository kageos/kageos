# Podman Runtime 事故复盘与本地客户端风险评估

> 日期：2026-06-04  
> 背景：macOS Podman machine XFS 损坏导致 Kageos 本地容器环境不可用  
> 结论级别：重大事故复盘 + 架构风险判断

## 结论

Kageos 项目本身还有救，而且生产单机 Linux 这条路比 macOS Podman Desktop 这条路健康很多。

这次事故的核心不是 Kageos 业务数据被代码删除，也不是容器被主动清空，而是：

1. macOS 上 Podman 不是原生容器运行时，必须通过一个 Linux VM 承载容器。
2. 本机 Podman machine 的 raw 磁盘里 XFS 文件系统损坏。
3. Kageos dev 基础设施的 MySQL / MinIO / NATS 容器和 volume 都落在这个 Podman VM 里面。
4. VM 起不来后，Podman Desktop / CLI 连不上，就表现为“容器和数据全没了”。
5. 修复 XFS 后，最终确认能读回 25 个容器、16 个镜像、9 个 volume。

因此，这次不是“项目不可用”，而是暴露出一个产品化风险：如果要把 Kageos 做成给普通用户安装的 macOS 本地客户端，不能把用户关键数据默认压在 Podman machine 的单个隐藏 raw 磁盘里。

## 今天事故的实际链路

当天现象：

- 网络异常或断开。
- 桌面非常卡。
- 重启 macOS。
- 重启后 Podman machine 起不来。
- 每次打开终端都会像是在启动 Podman，并一直卡住。
- Podman Desktop / CLI 里看起来像容器没了。

定位结果：

- `.zshrc` 里有自动启动 Podman 的逻辑，每开一个终端都会执行 `podman machine start`。
- Podman machine 的 Linux 根分区报错：`XFS ... Corruption detected. Unmount and run xfs_repair`。
- Podman machine 启动卡死后，终端自动启动逻辑放大了问题。

恢复方式：

- 禁用 shell 自动启动 Podman。
- 备份原始 raw。
- 本机 QEMU 启 Fedora CoreOS 救援系统。
- 对修复候选盘执行 `xfs_repair -L`。
- `xfs_repair -n` 校验通过。
- 用修复盘启动 Podman machine，确认容器、镜像、volume 可读。
- 将修复盘放回默认 Podman raw 路径，保留原损坏盘和修复前备份。

保留下来的关键文件：

```text
/Users/beiluo/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw
/Users/beiluo/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw.backup-before-xfs-repair-20260604-002621
/Users/beiluo/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw.corrupt-before-qemu-repair-20260604-013710
```

事故恢复记录：

```text
/Users/beiluo/podman-recovery-qemu/incident-2026-06-04-podman-xfs-recovery.md
```

## Kageos 当前怎么使用 Podman

### 1. 本地开发模式

入口文档：

- `deploy/dev/README.md`
- `deploy/dev/scripts/infra.sh`
- `deploy/dev/compose/docker-compose.infra.yml`
- `deploy/dev/compose/docker-compose.dev.yml`

本地开发推荐命令：

```bash
go run ./cmd/kagectl init --dev
```

默认 engine 是 Podman：

```go
opts := initDevOptions{Engine: "podman"}
```

Podman dev compose 会启动：

- `mysql8`
- `nats-server`
- `minio`

Podman dev compose 里的数据卷：

```yaml
volumes:
  infra-mysql-data:
    name: kageos-dev-mysql3318-data
  infra-minio-data:
    name: kageos-infra-minio-data
```

这两个是 Podman named volume。macOS 上它们实际在 Podman machine 的 Linux VM 里，不在一个普通可见、可直接 Time Machine/文件级备份的宿主机目录里。

### 2. app-runtime 模式

关键代码：

- `core/app-runtime/service/container_service.go`
- `core/app-runtime/service/infra_watchdog.go`
- `pkg/config/app_runtime.go`

app-runtime 当前只支持 Podman：

```go
defaultContainerRuntime = "podman"
```

用户 App 的运行通过 `podman run` 完成：

```go
podman run -d --name <app> -v <hostPath>:<containerPath> ...
```

`InfraWatchdog` 会把 NATS 断开视为基础设施异常，然后在 macOS / Windows 上自动检查并启动 Podman machine：

```go
podman machine list --format {{.Running}}
podman machine start
```

这点在正常情况下有自恢复价值，但在本次事故里属于危险点：如果 Podman machine 的底层磁盘已经损坏，应用层自动反复尝试启动可能会把问题从“一个明确的启动失败”变成“系统持续卡住、日志噪音增加、用户误以为该 init”。

### 3. 生产单机模式

入口文档：

- `deploy/prod/README.md`
- `deploy/prod/QUICK_START.md`
- `deploy/prod/DEPLOY_TUTORIAL.md`

生产前置条件明确写的是 Linux：

```text
Linux 宿主机
已安装 podman compose 或 docker compose
/data 可写
```

生产 compose 渲染后主要结构：

```text
Compose
  ├─ mysql / nats / minio
  └─ main 容器（network_mode: host, privileged: true）
       ├─ Nginx
       ├─ core-server
       └─ Podman API
```

生产数据落点是宿主机固定目录：

```text
/data/kageos/mysql
/data/kageos/minio
/data/kageos/namespace
/data/kageos/data
/data/kageos/logs
/data/kageos/podman_storage
```

`main` 容器里安装 Podman，并启动：

```bash
podman system service --time=0 unix:///run/podman/podman.sock
```

并把宿主机 `/data/kageos/podman_storage` 挂到容器内：

```yaml
- /data/kageos/podman_storage:/var/lib/containers
```

这意味着生产的 Podman 存储虽然仍然重要，但它不是 macOS 那种隐藏在 Podman machine raw 里的 opaque VM 磁盘，而是 Linux 宿主机上的普通目录。

## Mac 本地客户端风险判断

如果目标是“用户像安装客户端一样在自己的 Mac 上本地运行 Kageos”，当前直接依赖 Podman machine 的方案风险偏高。

原因不是 Podman 不能用，而是产品化不可接受的几个点：

1. **多一层 VM 故障域**

   macOS 没有 Linux kernel，Podman 需要 Linux VM。Kageos 用户看到的是一个 Mac App 或命令，但真实数据可能在 VM raw 里。VM 损坏时，普通用户没有能力判断是 app 坏了、Podman 坏了、还是数据没了。

2. **关键数据默认落在 opaque raw 磁盘**

   当前 dev MySQL/MinIO 使用 named volume。macOS 上 named volume 在 Podman VM 内部。raw 文件一旦 XFS 损坏，所有 named volume、镜像、容器元数据会一起受影响。

3. **自动启动会放大损坏**

   代码里的 `InfraWatchdog` 会在 macOS 上自动 `podman machine start`。如果 machine 是健康的，这是自恢复；如果 machine 底层盘坏了，这会反复尝试启动一个坏 VM。

4. **用户容易误操作 `podman machine init`**

   本次事故里用户两个月前曾直接 init，结果从头开始。对普通用户而言，“连不上 Podman”时最容易跟着提示执行 init，而 init 的结果可能是新建空机器，让原数据更难定位。

5. **备份口径不清晰**

   对产品用户来说，“备份 Kageos”应该是一个明确动作。现在 Mac dev 口径下至少有：

   - Podman machine raw
   - Podman named volume
   - `.kageos/dev/env`
   - `.kageos/dev/config`
   - app namespace / data

   这不是普通客户端用户能理解和长期维护的模型。

## Linux 上会不会出现这种问题

Linux 上仍然可能出现磁盘损坏、断电、文件系统 corruption、硬盘坏块、错误删除等问题。任何持久化系统都不能声称“不会坏”。

但这次事故里的核心故障模式是 macOS Podman machine 特有或显著放大的：

| 风险点 | macOS Podman | Linux Podman |
|---|---|---|
| 是否需要 Linux VM | 需要 | 不需要 |
| 数据是否可能集中在 VM raw | 是 | 否，生产设计为 `/data/kageos` 目录 |
| Podman machine 是否可能损坏 | 是 | 不存在这层 |
| 排障复杂度 | 高：macOS + VM + XFS + gvproxy + SSH 转发 | 低：直接看 systemd、Podman、宿主机文件系统 |
| 备份方式 | raw 级 + 容器内导出 + host bind 混合 | 目录级备份 + 数据库/对象存储备份 |
| 用户误 init 风险 | 高 | 低 |

所以准确说：

- Linux 不是不会坏。
- 但 Linux 上不会出现“Podman machine raw 里面的 XFS 坏了，导致所有容器数据像凭空消失”这种 Mac/Windows VM 层问题。
- Kageos 生产文档要求 Linux 是合理的。
- 真正生产或客户长期运行，应该优先 Linux appliance / Linux server，而不是裸 Mac Podman Desktop。

## 项目还有救吗

有，而且方向很清楚。

### 可救的原因

1. 核心业务数据模型没有证明损坏。
2. 生产部署已经把数据收敛到 `/data/kageos`。
3. `kagectl uninstall` 默认还刻意保留 `podman_storage`，说明代码已有“运行时镜像/容器存储是重要资产”的意识。
4. 事故后恢复验证能看到容器、镜像、volume，说明项目的数据不依赖不可恢复的瞬时状态。

### 需要承认的问题

1. macOS 本地长期运行方案不够产品化。
2. 自动启动 Podman machine 缺少事故态保护。
3. 本地 dev 的 MySQL/MinIO 使用 named volume，不利于普通用户文件级备份。
4. 当前缺少“一键备份 / 一键诊断 / 禁止误 init / raw 损坏恢复说明”的用户级运维层。

## 建议路线

### P0：立刻补安全护栏

1. 不要在 shell 启动时自动 `podman machine start`。
2. Kageos 代码里所有 `podman machine start` 都必须加超时。
3. 启动失败时不要提示用户直接 `podman machine init`，而是提示：

   ```text
   不要 init。请先备份 Podman machine raw，再执行诊断。
   ```

4. `InfraWatchdog` 在连续失败后应进入 degraded 状态，不要继续尝试启动。
5. 增加诊断命令：

   ```bash
   kagectl doctor-dev
   kagectl podman-diagnose
   ```

   至少输出：

   - Podman machine 状态
   - machine raw 路径
   - Podman socket 状态
   - kageos dev volume 是否存在
   - 容器数量
   - 镜像数量
   - volume 数量
   - 最近启动日志位置

### P1：本地 dev 数据改为宿主机可见目录

本地开发的 MySQL / MinIO 不建议继续只用 named volume。建议允许：

```text
.kageos/dev/data/mysql
.kageos/dev/data/minio
.kageos/dev/namespace
.kageos/dev/data/logs
```

compose 改成 bind mount：

```yaml
volumes:
  - ../../../.kageos/dev/data/mysql:/var/lib/mysql
  - ../../../.kageos/dev/data/minio:/data
```

这样即使 Podman machine raw 坏了，也可以重建 VM，然后重新挂回宿主机目录。MySQL 在 macOS bind mount 上性能和一致性要验证，但“用户数据可被宿主机备份软件看到”这个收益很大。

### P1：增加备份命令

建议增加：

```bash
kagectl backup-dev
kagectl restore-dev
kagectl backup-prod
kagectl restore-prod
```

dev 至少备份：

- `.kageos/dev/env/kageos.env`
- `.kageos/dev/config`
- MySQL dump
- MinIO bucket 导出
- namespace
- app data
- Podman 容器/镜像清单

prod 至少备份：

- `.kageos/prod/kage.yaml`
- `.kageos/prod/generated/env/kageos.env`
- `/data/kageos/mysql` 或 MySQL dump
- `/data/kageos/minio`
- `/data/kageos/namespace`
- `/data/kageos/data`
- `/data/kageos/podman_storage`

### P2：本地客户端产品化不要直接暴露 Podman machine

如果 Kageos 要做“普通用户本地客户端”，建议三选一：

#### 方案 A：Linux appliance 优先

给用户一个受控 Linux appliance：

- 本地 Linux 小主机
- 或官方 VM 镜像
- 或云端私有实例

Kageos 在里面按生产模式跑，数据在 `/data/kageos`。Mac App 只是客户端壳，访问这个本地/局域网服务。

优点：最接近生产，风险最低。  
缺点：安装体验比纯 Mac App 重。

#### 方案 B：Mac 客户端 + 受控 VM，但必须做数据外置和快照

如果一定要 Mac 本地一键安装：

- 不让用户直接管理 Podman Desktop。
- App 自己管理 VM 生命周期。
- 数据目录放在 `~/Library/Application Support/Kageos/`。
- 定期自动备份 MySQL / MinIO / namespace。
- VM raw 只作为可重建运行时，不作为唯一数据源。

优点：用户体验好。  
缺点：工程量大，仍然有 VM 层。

#### 方案 C：本地客户端弱化容器依赖

把 Kageos 本地单用户版改成：

- SQLite / embedded DB
- 本地文件对象存储
- 只有用户 App 沙箱才临时用容器
- 用户 App 容器尽量无状态，可重建

优点：最像普通桌面软件。  
缺点：与生产架构差异变大，需要额外开发适配。

## 对当前代码的具体风险点

### 风险 1：`podman machine start` 没有独立超时

位置：

- `core/app-runtime/service/infra_watchdog.go`
- `core/app-runtime/service/container_service.go`

问题：

- 如果 `podman machine start` 卡住，调用方会被拖住。
- 本次事故正是 start 卡住型问题。

建议：

- 使用 `context.WithTimeout` 包裹，超时后进入 degraded。
- 连续失败次数达到阈值后停止自动恢复。

### 风险 2：错误提示容易引导用户 init

位置：

- `container_service.go` 中多处提示 `podman machine init`。

建议：

- 把提示改成事故安全文案：

```text
Podman machine 不可用。不要直接执行 podman machine init。
请先运行 kagectl podman-diagnose，并备份 machine raw。
```

### 风险 3：dev named volume 不利于备份

位置：

- `deploy/dev/compose/docker-compose.infra.yml`
- `deploy/dev/compose/docker-compose.dev.yml`

建议：

- dev 也支持宿主机 bind mount 数据目录。
- 至少让用户可选 `KAGEOS_DEV_DATA_ROOT`。

### 风险 4：生产 Podman storage 是核心资产

位置：

- `cmd/kagectl/templates.go`

```yaml
- /data/kageos/podman_storage:/var/lib/containers
```

建议：

- 文档里把 `/data/kageos/podman_storage` 明确列为备份对象。
- 备份策略区分：
  - 可重建镜像：可不强制备份
  - 用户 App 容器元数据 / app-base / build cache：建议备份或可重建说明

## 我的判断

1. **Kageos 生产部署应该继续以 Linux 为第一目标。**
2. **Mac Podman Desktop 更适合开发，不适合作为普通用户长期存放唯一数据的本地客户端底座。**
3. **如果要做 Mac 本地客户端，必须先做数据外置、备份恢复、诊断护栏，不然这次事故会在用户侧复现。**
4. **项目不是没救，而是现在到了需要把“开发环境容器化”升级成“产品级运行时管理”的阶段。**

## 官方背景资料

- Podman 官方 machine 文档说明，`podman machine` 用于管理 Podman 的虚拟机；macOS / Windows 上 Podman 需要虚拟机，因为容器核心能力依赖 Linux kernel。Linux 上 `podman machine` 是可选项，不是生产必需层：[podman-machine — Podman documentation](https://docs.podman.io/en/latest/markdown/podman-machine.1.html)。
- Podman 官方安装文档也说明，Mac 上每个 Podman machine 都由虚拟机承载，宿主机上的 `podman` CLI 是远程连接 VM 里的 Podman service：[Podman Installation Instructions](https://podman.io/docs/installation)。

## 后续行动清单

建议按这个顺序做：

1. 增加 `kagectl podman-diagnose`。
2. 改掉 `podman machine start` 无超时问题。
3. 改错误提示，禁止用户事故态直接 init。
4. dev compose 支持宿主机 bind mount 数据目录。
5. 增加 `backup-dev` / `restore-dev`。
6. 增加生产备份文档和 `backup-prod` / `restore-prod`。
7. 评估 Mac 客户端时，不把 Podman machine raw 作为唯一数据源。
