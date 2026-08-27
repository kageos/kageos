# kageos 备份与恢复

当前生产单机部署提供手动、可校验、可恢复的 `kagectl` 备份闭环。它适合默认的 bundled MySQL 与 bundled MinIO 部署。外部数据库或外部对象存储应使用供应商提供的一致性备份和恢复机制，`kagectl` 会明确拒绝混合执行，避免制造不完整备份。

## 能力边界

备份包含：

- 私有生产配置 `.kageos/prod/kage.yaml`，包括恢复所需密钥；
- `<storage.root>/mysql`；
- `<storage.root>/minio`；
- `<storage.root>/namespace`；
- `<storage.root>/data`；
- `<storage.root>/tls`。

备份不包含可重新生成或拉取的 `.kageos/prod/generated`、平台普通日志和 `podman_storage`。恢复后，`kagectl` 会重新渲染部署文件并启动、验证实例。

当前使用一致性物理备份。创建备份时会停止生产 Compose 栈，归档完成并通过逐文件 SHA-256 校验后再启动服务。这样牺牲一段与数据量相关的停机时间，换取 MySQL、MinIO、工作空间源码和本地数据处于同一个一致性时间点。

物理数据要求恢复目标使用与备份相同的 kageos 主镜像、用户应用基础镜像、MySQL 镜像和 MinIO 镜像。恢复命令会在写入前检查这四项；需要恢复旧版本备份时，应先把当前实例配置切换到 manifest 记录的镜像版本。

## 创建备份

从仓库根目录、使用拥有该部署和 Compose 的部署用户执行：

```bash
go run ./cmd/kagectl backup
```

默认输出：

```text
.kageos/prod/backups/kageos-backup-<UTC timestamp>.tar.gz
.kageos/prod/backups/kageos-backup-<UTC timestamp>.tar.gz.sha256
```

也可以指定另一块已挂载磁盘：

```bash
go run ./cmd/kagectl backup --output /mnt/kageos-backups
```

如果 `--output` 以 `.tar.gz` 结尾，它会作为准确文件名使用。命令拒绝覆盖已有文件。归档权限为 `0600`，因为其中包含数据库、用户文件、连接器数据、JWT 和其他恢复密钥。

输出路径必须位于 `<storage.root>` 之外，防止备份过程把正在生成的归档再次收入自身。它仍可能与生产数据处于同一块磁盘；这种情况下只能用于逻辑回滚，不能替代异盘或异机副本。

备份完成后应复制到另一块磁盘、另一台服务器或受控对象存储。同一服务器、同一磁盘上的副本可以用于升级回滚和误操作恢复，但不能防止整机或磁盘损坏。

## 查看和校验

列出默认目录中的备份：

```bash
go run ./cmd/kagectl backup list
```

离线校验归档结构、必需数据集、逐文件大小和 SHA-256：

```bash
go run ./cmd/kagectl backup verify \
  .kageos/prod/backups/kageos-backup-20260827T120000Z.tar.gz
```

校验不会连接或修改正在运行的实例。不要只依赖旁边的 `.sha256` 文件；`backup verify` 还会检查内部 manifest 与每一个归档条目。

## 恢复预检

先执行 dry-run：

```bash
go run ./cmd/kagectl restore \
  /mnt/kageos-backups/kageos-backup-20260827T120000Z.tar.gz \
  --dry-run
```

dry-run 会：

- 完整读取和校验归档；
- 检查 manifest schema；
- 检查备份是否来自一致性停机快照；
- 检查 MySQL 与 MinIO 镜像版本；
- 显示将被替换的数据范围；
- 不停止服务、不写入任何数据。

## 执行恢复

恢复会覆盖当前实例数据，必须显式使用 `--force`：

```bash
go run ./cmd/kagectl restore \
  /mnt/kageos-backups/kageos-backup-20260827T120000Z.tar.gz \
  --force
```

执行顺序：

1. 完整校验归档并解压到与 `storage.root` 同一文件系统的临时目录；
2. 停止生产 Compose 栈；
3. 将当前 `mysql`、`minio`、`namespace`、`data` 和 `tls` 原子移动到回滚目录；
4. 安装备份数据并恢复私有配置；目标机器的 `storage.root` 路径保持不变；
5. 重新渲染配置，启动平台并运行完整分层验证；
6. 验证失败时自动恢复原数据和原配置，再尝试启动原实例；
7. 验证成功后默认删除回滚目录，避免长期占用双份磁盘。

需要在成功后暂时保留回滚目录时：

```bash
go run ./cmd/kagectl restore <archive> --force --keep-rollback
```

这会显著增加磁盘占用，应在人工验收后及时移走或清理。

## 磁盘空间

创建备份是流式 tar+gzip，不会先复制完整数据到 staging，但最终归档仍需要与压缩后数据量相当的目标空间。恢复需要先解压一份完整数据，并在切换期间保留原数据作为回滚，因此目标文件系统必须能短时间容纳新旧两份数据。

恢复前建议至少保留：

```text
归档解压后大小 + 当前在线数据大小 + 20% 安全余量
```

若容量不足，不要使用同盘恢复。先扩容，或在新服务器安装相同版本的 kageos 后执行恢复。

## 当前未包含

- 定时备份策略和 UI 控制面；
- S3/OSS/COS 自动上传；
- 增量备份和 MySQL 时间点恢复；
- 外部 MySQL、外部 MinIO 的统一备份；
- 无停机跨数据面快照；
- 全局 `kageos backup <instance>` 多实例管理器包装。

这些能力可以在手动备份/恢复经过真实生产演练后，复用 `timer-scheduler` 和实例管理器逐步增加。
