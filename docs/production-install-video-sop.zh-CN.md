# kageos 国内服务器生产安装录屏 SOP

本文用于录制“从一台干净的国内云服务器安装 kageos 并完成首次登录”的视频教程。命令以 `https://kageos.com/install-prod.sh` 当前提供的生产安装器为准。

本次演示服务器为 `kageos-cn-small`。旧的 `instance-01`、持久化数据和专用镜像已经清理；KageOS 管理器、Podman、1Panel 和服务器系统均保留。重新安装必须由录制者亲手执行。

## 1. 视频目标和完成标准

观众看完后应当能够：

1. 准备一台具备公网访问能力的 Linux 云服务器。
2. 使用国内安装入口一键部署 kageos。
3. 使用管理命令确认实例健康。
4. 从浏览器打开 kageos，并以 `system` 用户完成首次登录。
5. 知道后续查看状态、日志、密码和执行更新的命令。

本次录制只有同时满足以下条件才算成功：

- `sudo kageos instance list` 显示一个运行中的实例。
- `sudo kageos verify instance-01` 通过。
- 服务器本机可以访问实例端口。
- 浏览器可以打开登录页。
- `system` 用户可以登录。
- 视频画面和终端回滚中没有泄露密码、SSH 密钥、Token 或云账号信息。

## 2. 录制前准备

### 2.1 服务器最低建议

- Ubuntu 22.04 或 24.04，x86_64。
- 2 核 CPU、4 GB 内存。
- 至少 20 GB 可用磁盘空间。
- 一个具有 `sudo` 权限的普通用户。
- 可以访问 Docker/Podman 镜像仓库和 `https://kageos.com`。

本次 `kageos-cn-small` 当前约有 40 GB 可用空间，满足录制条件。

### 2.2 云安全组

安装器默认从主机端口 `10001` 开始选择第一个空闲端口。录制前在云厂商控制台准备一条入站规则：

- 协议：TCP。
- 端口：`10001`。
- 来源：优先只允许录制电脑当前公网 IP。

如果安装结果使用了其他端口，应按安装摘要中的实际端口调整规则。不要为了省事开放所有端口。

### 2.3 录屏脱敏

开始录制前关闭通知、密码管理器弹窗、剪贴板历史和浏览器自动填充提示。以下内容不得出现在成片里：

- SSH 私钥、密钥文件内容或密码。
- `system` 用户密码。
- 云厂商 AccessKey、Token、账号 ID 和账单信息。
- 其他项目的环境变量、Cookie 和客户数据。

公网 IP 如果不希望公开，应提前设置终端和视频后期遮罩。安装器的成功摘要会输出登录密码，因此本 SOP 使用实时过滤命令隐藏 `Password:` 行。

## 3. 推荐视频结构

| 时间 | 画面 | 旁白重点 |
|---|---|---|
| 00:00–00:30 | 标题和最终效果 | 今天从干净服务器部署一个可登录的 kageos 实例。 |
| 00:30–01:30 | SSH 登录与环境检查 | 说明系统、资源、空闲端口和安全组。 |
| 01:30–02:00 | 安装器帮助 | 说明国内入口、容器化安装和数据目录。 |
| 02:00–05:00 | 执行安装 | 首次拉取和启动可能需要数分钟；等待部分可后期加速。 |
| 05:00–06:30 | 状态与健康验收 | 展示实例列表、状态、doctor 和 verify。 |
| 06:30–08:00 | 浏览器首次登录 | 密码在停录时读取；成片只展示被遮蔽的输入。 |
| 08:00–09:00 | 常用运维命令 | 状态、日志、重启、更新和备份。 |
| 09:00–09:30 | 总结 | 强调数据目录、安全组和备份。 |

## 4. 正式录制步骤

### 步骤 1：连接服务器

本次录制电脑已经配置 SSH 别名，可以执行：

```bash
ssh kageos-cn-small
```

面向普通观众时，将其解释为：

```text
ssh <用户名>@<服务器公网 IP>
```

不要把尖括号占位符原样粘贴执行。

建议旁白：

> 我现在连接的是一台 Ubuntu 云服务器。生产环境建议使用普通用户登录，并通过 sudo 完成安装，不要长期直接使用 root 远程登录。

### 步骤 2：执行只读预检

逐条执行，方便观众理解每个检查项：

```bash
hostnamectl --static
uname -m
free -h
df -h /
sudo -v
sudo kageos instance list
sudo ss -ltn | grep ':10001 ' || true
```

预期结果：

- 架构为 `x86_64`。
- 内存和磁盘满足最低要求。
- `sudo` 可用。
- 实例列表为空。
- `10001` 当前没有监听。

如果服务器尚未安装 `kageos` 管理器，`sudo kageos instance list` 报“command not found”是正常的；一键安装器会自动安装它。

### 步骤 3：展示安装器帮助

只读取脚本帮助，不执行安装：

```bash
curl -fsSL https://kageos.com/install-prod.sh | bash -s -- --help
```

建议旁白：

> 这个安装器会自动准备缺少的基础依赖，拉取已发布的容器镜像，创建独立实例，并安装统一的 kageos 管理命令。国内服务器加 `--cn`，会优先使用国内镜像路径。

### 步骤 4：执行国内生产安装

先让当前 shell 在管道任一环节失败时返回失败：

```bash
set -o pipefail
```

然后执行安装。下面的命令关闭容器内部密码输出，并把安装器最终摘要中的密码行替换为录屏占位文本：

```bash
curl -fsSL https://kageos.com/install-prod.sh | sudo env KAGEOS_AIO_PRINT_SECRETS=0 bash -s -- --cn 2>&1 | sed -E 's/^(Password:).*/\1 [录屏已隐藏]/'
```

不要在安装过程中关闭 SSH。首次拉取镜像和初始化 MySQL、MinIO、NATS 等服务可能需要数分钟。后期可以加速等待片段，但应保留开始拉取、等待首次启动和成功摘要三个关键画面。

若希望显式指定访问地址，可以在确认不会泄露地址后使用：

```text
curl -fsSL https://kageos.com/install-prod.sh | sudo env KAGEOS_AIO_PRINT_SECRETS=0 bash -s -- --base-url http://<服务器公网 IP> --cn
```

### 步骤 5：验收实例

安装完成后执行：

```bash
sudo kageos instance list
sudo kageos status instance-01
sudo kageos doctor instance-01
sudo kageos verify instance-01
curl -fsS -o /dev/null -w 'local_http=%{http_code}\n' http://127.0.0.1:10001/
```

预期：

- 实例名称为 `instance-01`，状态为运行中。
- `doctor` 没有阻断性错误。
- `verify` 通过。
- 本机 HTTP 请求返回正常状态码。

如果安装器选择的不是 `10001`，以 `sudo kageos instance list` 显示的端口为准，同时修改本机检查命令和云安全组规则。

### 步骤 6：停录并取得首次登录密码

这一段必须暂停录屏。确认 OBS、系统录屏和终端回滚录制均已停止后执行：

```bash
sudo kageos initial-password instance-01
```

把密码临时保存到可信的密码管理器，清空终端画面和剪贴板历史后再恢复录制。不要把密码写进脚本、文档、聊天或命令行参数。

### 步骤 7：浏览器首次登录

1. 打开安装摘要中的 URL，例如 `http://<服务器公网 IP>:10001`。
2. 用户名输入 `system`。
3. 输入刚才离线取得的密码。
4. 完成登录并展示首页或工作台。

密码输入框虽然通常会显示圆点，但仍应确认浏览器没有弹出带明文的密码保存提示。

建议旁白：

> 现在访问的是刚刚部署的自托管实例。初始管理员用户是 system，安装密码保存在服务器受限的数据目录中，可以通过管理命令在服务器本地读取。system 修改密码后，该安装密码不再代表当前密码。

### 步骤 8：展示常用管理命令

```bash
sudo kageos status instance-01
sudo kageos logs instance-01
sudo kageos restart instance-01
sudo kageos update instance-01
sudo kageos backup instance-01
sudo kageos instance list
```

录制时不必真的执行 `restart`、`update` 和 `backup`；可以只展示命令并解释用途。若要执行更新，先确认当前实例已经完成备份。

## 5. 常见问题与现场兜底

### 浏览器打不开，但本机检查正常

优先检查云安全组是否允许实际实例端口，而不是马上修改服务器全局防火墙：

```bash
sudo kageos instance list
sudo ss -ltnp | grep ':10001 '
```

如果实例实际端口变化，使用对应端口检查。只有明确理解影响时才修改 UFW、iptables 或安全组。

### 拉取镜像慢或失败

确认安装命令包含 `--cn`，然后检查 DNS 和 HTTPS 连通性：

```bash
curl -I https://kageos.com/install-prod.sh
```

不要在未判断原因时连续反复运行安装器。

### 首次启动等待时间很长

另开一个 SSH 窗口查看：

```bash
sudo kageos instance list
sudo kageos logs instance-01
```

保留错误现场。不要直接删除数据目录或执行容器全局清理。

### `10001` 已被占用

安装器会从 `10001` 开始自动选择空闲端口。安装后以实例列表和成功摘要为准：

```bash
sudo kageos instance list
```

### 安装失败后如何继续

先收集只读信息：

```bash
sudo kageos instance list
sudo kageos doctor instance-01
sudo kageos logs instance-01
df -h /
free -h
```

如果实例尚未注册，保留完整安装错误输出进行排查。不要使用 `podman system prune`、`docker system prune` 或通配符删除目录。

## 6. 成片发布前检查

- [ ] 视频里没有管理员密码、SSH 密钥、Token、Cookie 或云账号信息。
- [ ] 公网 IP、主机名和账号 ID 已按发布策略决定保留或遮罩。
- [ ] 安装命令来自 `https://kageos.com/install-prod.sh`，并包含国内参数 `--cn`。
- [ ] 视频明确说明安全组只开放必要端口和必要来源。
- [ ] 实例列表、`doctor`、`verify` 和浏览器登录均真实通过。
- [ ] 没有把安装过程剪辑成无法复现的“瞬间成功”。
- [ ] 没有展示或建议全局 prune、关闭防火墙等高风险操作。
- [ ] 视频描述区附上官网、安装命令和本 SOP 对应版本日期。

## 7. 建议视频简介

> 本视频演示如何在国内 Ubuntu 云服务器上，通过 kageos 官方生产安装器完成自托管部署、健康检查和首次登录。安装采用容器化方式，支持统一的状态、日志、更新和备份命令。生产使用前请配置域名、HTTPS、最小化安全组规则和独立备份。

文档核对日期：2026-08-25。
