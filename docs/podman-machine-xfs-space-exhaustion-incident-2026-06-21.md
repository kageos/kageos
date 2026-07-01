# Podman Machine XFS 损坏事故复盘：宿主机空间耗尽

> 日期：2026-06-21  
> 环境：macOS + Podman Desktop / Podman machine `libkrun`  
> 影响范围：本地开发容器运行时不可用，`podman images` / `podman ps` / 本地容器启动失败  
> 结论级别：本地运行时重大故障复盘

## 结论

本次事故的根因可以基本确认：macOS 宿主机磁盘空间严重不足，事故发生时只剩约 `5G` 可用空间，导致 Podman machine 的 raw 虚拟磁盘写入异常，进而造成 VM 内 XFS 根分区元数据损坏。

直接损坏点不是 Kageos 代码，也不是用户主动删除容器数据，而是：

1. macOS 上 Podman 需要一个 Linux VM 承载容器。
2. 该 VM 的磁盘文件是宿主机上的 raw 文件。
3. 宿主机空间耗尽时，VM 内部仍会继续写 journal、容器层、overlay、数据库和日志。
4. 底层写入失败或不完整后，VM 内 XFS 文件系统出现元数据损坏。
5. Podman CLI 表面报的是 socket / SSH 连接失败，但真正问题在 VM 内部文件系统。

最终通过离线 `xfs_repair` 修复 XFS 后，Podman machine 和容器运行时恢复。

## 现象

用户侧看到的报错：

```text
Cannot connect to Podman. Please verify your connection to the Linux system using `podman system connection list`,
or try `podman machine init` and `podman machine start` to manage a new Linux VM

Error: unable to connect to Podman socket:
failed to connect: ssh: handshake failed:
read tcp 127.0.0.1:55888->127.0.0.1:55851: read: connection reset by peer
```

关键表现：

- `podman machine list` 一度显示 machine 正在运行或正在启动。
- `podman images` 卡住或失败。
- `krunkit` CPU 长时间异常升高。
- `gvproxy` / SSH 端口表面存在，但连接被 reset。
- Podman Desktop / CLI 都无法稳定访问 VM 内的 Podman socket。

这些现象容易误导为 Podman socket、SSH key、connection profile 或 gvproxy 故障。实际根因在 VM 磁盘文件系统。

## 影响

本次影响的是本地开发环境：

- Podman CLI 无法列出镜像和容器。
- Kageos 本地容器依赖不可用。
- 已有镜像、容器、volume 一度看起来像消失。
- 若误执行 `podman machine init`，可能创建新空 VM，增加恢复难度。

修复后验证：

```bash
podman images
podman ps -a
podman run --rm docker.io/library/alpine:latest echo final-podman-ok
```

均恢复正常。

## 关键证据

### 1. 宿主机空间耗尽

事故发生时，用户观察到宿主机只剩约 `5G` 可用空间，微信等应用已经提示磁盘空间严重不足。

释放部分空间后，诊断时宿主机 Data 卷约为：

```text
/System/Volumes/Data: 460Gi, used 402Gi, available 34Gi, capacity 93%
```

修复完成并进一步清理后约为：

```text
/System/Volumes/Data: 460Gi, used 309Gi, available 127Gi, capacity 71%
```

这说明真正事故窗口里的空间压力比诊断时更严重。

### 2. VM 日志出现 I/O error

Podman machine 串口日志里出现大量 journald 写入错误：

```text
systemd-journald: Failed to open /var/log/journal/...: Input/output error
systemd-journald: Failed to rotate ... system.journal: Input/output error
```

`Input/output error` 是文件系统或底层块设备异常的强信号，不是普通 Podman connection 配置问题。

### 3. XFS dry-run 确认元数据损坏

离线挂载 raw 盘后，确认分区结构：

```text
/dev/vda2  EFI-SYSTEM  vfat
/dev/vda3  boot        ext4
/dev/vda4  root        xfs
```

`xfs_repair -n /dev/vda4` 发现真实损坏：

```text
Metadata CRC error
bad directory block magic
corrupt block 0 in directory inode ...
entry "tmp" ... references free inode
disconnected dir inode ...
```

因此可以确认是 XFS 元数据损坏。

## 直接原因与根因

### 直接原因

Podman machine 的 XFS 根分区 `/dev/vda4` 损坏，导致 VM 内服务不稳定，进而使宿主机上的 Podman CLI 无法通过 SSH/gvproxy 连接到 VM 内 Podman socket。

### 根因

宿主机磁盘空间严重不足，只剩约 `5G` 可用空间。Podman machine 的 raw 磁盘仍在承载大量写入，底层空间耗尽后触发 VM 内 XFS 写入失败或不完整，最终损坏文件系统元数据。

### 促成因素

1. **macOS Podman 多一层 VM 故障域**

   macOS 上 Podman 并非原生 Linux 容器，容器数据实际在 Linux VM 内。宿主机空间问题会间接伤到 guest 文件系统。

2. **VM raw 磁盘较大且集中承载状态**

   当前 Podman machine raw 文件虚拟大小为 `100GiB`，实际占用约 `80GiB`。镜像、容器、volume 和 VM 系统状态都集中在这个磁盘里。

3. **本地镜像和容器历史较多**

   修复后 `podman system df` 显示：

   ```text
   Images         312   30.61GB   27.45GB reclaimable
   Containers      68   651.1MB   651.1MB reclaimable
   Local Volumes   11   15.46GB   5.746GB reclaimable
   ```

   镜像可回收空间较大，长期未清理会放大宿主机空间压力。

4. **自动启动逻辑会放大故障噪音**

   排障过程中发现 GoLand 中一个调试态程序持续触发 `podman machine start`，会反复拉起损坏 VM。此前存在的 `com.podman.auto-start` LaunchAgent 也有自动启动 Podman 的意图。

   这不是根因，但在事故态会增加重试、卡顿和日志噪音。

5. **Podman 官方提示容易诱导误操作**

   报错建议 `podman machine init`。如果用户在未备份 raw 磁盘前 init 新 machine，可能让原有数据位置更难定位。

## 恢复过程

### 1. 释放宿主机空间

用户先手动删除文件，将宿主机可用空间从约 `5G` 恢复到 `30G+`。这是后续修复能成功的前提。

### 2. 停止卡死的 Podman VM 进程

确认存在异常进程：

```text
krunkit CPU 异常升高
gvproxy 存在但 SSH handshake reset
podman images 卡住或失败
```

停止 `krunkit` / `gvproxy`，确保 raw 磁盘离线，避免带电修复。

### 3. 备份 raw 磁盘

修复前对 raw 磁盘做 APFS clone 备份：

```text
$HOME/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw.backup-before-xfs-repair-20260621-120941
```

原始 raw：

```text
$HOME/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw
```

### 4. 使用 Alpine rescue VM 离线修复

macOS/Homebrew 没有可直接使用的 `xfsprogs`，因此启动临时 Alpine ARM64 rescue VM，将 Podman raw 磁盘作为块设备接入。

识别到：

```text
/dev/vda4 LABEL="root" TYPE="xfs"
```

先 dry-run：

```bash
xfs_repair -n /dev/vda4
```

第一次正式修复时，`xfs_repair` 提示需要先 replay log：

```text
The filesystem has valuable metadata changes in a log which needs to be replayed.
Mount the filesystem to replay the log, and unmount it before re-running xfs_repair.
```

因此先加载 XFS 模块并临时挂载：

```bash
modprobe xfs
mount -o nouuid /dev/vda4 /mnt/root
sync
umount /mnt/root
```

然后正式修复：

```bash
xfs_repair /dev/vda4
```

注意：本次没有使用 `xfs_repair -L` 强制丢弃日志。日志先通过挂载回放，再进行修复，风险低于直接 `-L`。

### 5. 校验修复结果

修复后再次只读检查：

```bash
xfs_repair -n /dev/vda4
```

并只读挂载确认文件系统可读：

```text
/dev/vda4  99.4G  67.9G  31.6G  68%  /mnt/root
```

### 6. 启动 Podman machine 并验证

启动后最终状态：

```text
podman-machine-default  libkrun  Currently running  6 CPU  8GiB memory  100GiB disk
```

功能验证：

```bash
podman images
podman ps -a
podman run --rm docker.io/library/alpine:latest echo final-podman-ok
```

结果：

```text
final-podman-ok
```

VM 内磁盘：

```text
/dev/vda4  100G  70G  30G  71%  /
/dev/vda4  100G  70G  30G  71%  /var
```

当前 boot 未再出现新的 XFS I/O error。journald 检测到历史 journal 文件不干净并自动替换：

```text
File /var/log/journal/.../system.journal corrupted or uncleanly shut down, renaming and replacing.
```

这是预期的恢复后清理行为。

## 排障中的额外发现

### 1. GoLand 调试进程反复拉起 Podman

排障时发现一个 GoLand 调试态进程链反复启动 Podman：

```text
dlv -> debugserver -> ___11go_build_... -> podman machine start
```

这会导致刚停下的 VM 又被拉起，使离线修复无法安全进行。修复前必须先停止这类自动启动源。

### 2. Podman auto-start LaunchAgent 配置损坏

发现本地 LaunchAgent：

```text
$HOME/Library/LaunchAgents/com.podman.auto-start.plist
```

存在 XML 问题：

- DOCTYPE 行多了反斜杠。
- `<string>` 中的 `&&` 未转义为 `&amp;&amp;`。

同时脚本：

```text
$HOME/bin/podman-auto-start.sh
```

没有给 launchd 环境补 `/opt/podman/bin`，导致 launchd 环境下可能找不到 `podman`。

已修复：

- plist 通过 `plutil -lint`。
- 脚本通过 `bash -n`。
- 脚本增加 `/opt/podman/bin` 到 `PATH`。

本次没有立即加载该 LaunchAgent，避免自动启动 `nats-server` 改变当前容器状态。

## 经验教训

### 1. `ssh handshake failed` 不一定是 SSH 问题

在 macOS Podman 环境里，错误链路是：

```text
podman CLI -> SSH/gvproxy -> Linux VM -> VM 内 podman.socket
```

只要 VM 内部文件系统或 systemd 服务异常，外层就可能表现成 SSH handshake reset、connection refused 或 socket 连接失败。

### 2. 宿主机空间不足会损坏 guest 文件系统

Podman machine raw 盘不是一个安全隔离边界。宿主机空间耗尽时，guest 内 XFS 仍可能被写坏。

### 3. 事故态第一原则是备份 raw，禁止 init

看到 Podman 提示 `podman machine init` 时，不要立即执行。

优先顺序应该是：

1. 停止自动启动源。
2. 停止 VM。
3. 备份 raw 磁盘。
4. 离线检查文件系统。
5. 只有确认无恢复价值时，才考虑 init 新 machine。

### 4. 自动重启要有健康检查边界

健康 VM 可以自动启动。损坏 VM 不应该被无限自动重启。

自动启动逻辑应先检查：

- 宿主机可用空间。
- Podman machine 是否处于反复 starting。
- 近期串口日志是否存在 `Input/output error`、`XFS ... corruption`。
- 是否有正在运行的 `xfs_repair` / rescue 流程。

## 后续改进项

### P0：宿主机磁盘空间护栏

建议在本地开发工具或启动脚本中加入硬性检查：

| 阈值 | 行为 |
| --- | --- |
| 可用空间 `< 80G` | 提示清理，标记为 warning |
| 可用空间 `< 50G` | 启动前强提示，建议停止大构建 |
| 可用空间 `< 20G` | 禁止自动启动 Podman machine / 禁止大规模 build |
| 可用空间 `< 10G` | 进入事故态，提示立即释放空间并停止 VM |

### P0：新增 `kagectl doctor`

建议诊断项：

```bash
df -h /
df -h /System/Volumes/Data
podman machine list
podman system connection list
podman system df
podman info
```

并额外检查：

- `krunkit` 是否异常高 CPU。
- `gvproxy` 是否监听当前 SSH port。
- Podman machine log 是否出现 `Input/output error`。
- 是否存在多个自动启动源同时拉起 Podman。

### P1：事故态文案

当检测到 Podman machine 无法连接时，不要直接建议用户 init。应提示：

```text
不要立即执行 podman machine init。
先备份 Podman raw 磁盘，再做离线检查。
```

### P1：本地数据备份策略

对开发环境至少提供：

- raw 盘备份位置说明。
- MySQL dump。
- MinIO 数据导出。
- Podman volume 列表。
- 一键导出关键 dev 数据。

### P1：清理策略

当前 `podman system df` 显示镜像可回收空间较大：

```text
Images reclaimable: 27.45GB
Containers reclaimable: 651.1MB
Volumes reclaimable: 5.746GB
```

建议提供安全清理脚本，至少分级：

```bash
podman image prune
podman container prune
podman volume prune
podman system prune
```

其中 `volume prune` 必须默认禁用或二次确认，因为可能删除开发数据。

### P2：长期产品化方向

对于面向普通用户的 macOS 本地客户端，不建议把关键数据默认压在 Podman machine 的 opaque raw 磁盘中。

更稳妥方向：

- 关键业务数据落宿主机显式目录。
- 容器运行时状态和业务数据分离。
- 提供明确的备份/恢复入口。
- 对 macOS Podman machine 仅作为开发或轻量运行时，不作为唯一持久化边界。

## 快速应急手册

如果再次出现类似错误：

```text
ssh: handshake failed
connection reset by peer
Cannot connect to Podman socket
```

优先执行：

```bash
df -h /System/Volumes/Data
podman machine list
ps aux | rg -i 'podman|krunkit|gvproxy'
```

如果宿主机可用空间很低：

1. 先释放宿主机空间，目标至少 `50G+`，最好 `80G+`。
2. 停止自动启动 Podman 的进程。
3. 停止 `krunkit` / `gvproxy`。
4. 备份 raw：

   ```bash
   RAW="$HOME/.local/share/containers/podman/machine/libkrun/podman-machine-default-arm64.raw"
   cp -c "$RAW" "$RAW.backup-before-repair-$(date +%Y%m%d-%H%M%S)"
   ```

5. 再做离线 XFS 检查和修复。

禁止在未备份前执行：

```bash
podman machine init
```

## 本次事故最终状态

- Podman machine 已恢复运行。
- `podman images` 正常。
- `podman ps -a` 正常。
- `podman run --rm alpine echo final-podman-ok` 正常。
- VM 内 `/` 和 `/var` 可用约 `30G`。
- 宿主机清理后可用约 `127GiB`。
- 修复前 raw 备份已保留。
