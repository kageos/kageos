# Podman 自动启动工具

## 概述

这是一个用于解决 macOS 系统重启后 Podman 机器不自动启动问题的工具集。在 macOS 上，Podman 使用虚拟机来运行容器，重启后需要手动启动 Podman 机器，这个工具可以自动完成这个过程。

## 问题背景

在 macOS 上使用 Podman 时，经常会遇到以下问题：
- 电脑重启后，`podman ps` 命令报错
- 需要手动运行 `podman machine start` 才能使用 Podman
- 没有自动启动机制，每次重启都需要手动操作

## 解决方案

本工具提供了完整的自动启动解决方案：
1. **自动启动脚本**：检查并启动 Podman 机器
2. **macOS 服务**：系统启动时自动运行脚本
3. **Shell 集成**：终端启动时自动检查 Podman 状态
4. **便捷命令**：提供简化的 Podman 操作命令

## 文件说明

```
podman-auto-start/
├── README.md                 # 说明文档
├── podman-auto-start.sh     # 主启动脚本
├── install.sh               # 安装脚本
└── uninstall.sh             # 卸载脚本
```

### podman-auto-start.sh
- **用途**：核心启动脚本，检查并启动 Podman 机器
- **功能**：
  - 检查 Podman 是否安装
  - 检查机器是否运行
  - 自动创建和启动机器
  - 验证连接状态
  - 提供详细的日志输出

### install.sh
- **用途**：一键安装自动启动服务
- **功能**：
  - 检查系统环境
  - 创建必要目录
  - 安装启动脚本
  - 创建 macOS 服务
  - 配置 shell 集成
  - 提供便捷命令别名

### uninstall.sh
- **用途**：完全卸载自动启动服务
- **功能**：
  - 卸载 macOS 服务
  - 删除相关文件
  - 清理 shell 配置
  - 清理日志文件

## 使用方法

### 安装

```bash
# 进入工具目录
cd podman-auto-start

# 运行安装脚本
./install.sh
```

### 手动使用

```bash
# 手动运行启动脚本
~/bin/podman-auto-start.sh

# 使用便捷命令
pps        # podman ps
pstart     # podman machine start
pstop      # podman machine stop
pstatus    # podman machine list
prestart   # 重启 Podman 机器
```

### 卸载

```bash
# 运行卸载脚本
./uninstall.sh
```

## 工作原理

### 1. 自动启动机制
- 使用 macOS 的 `launchd` 服务系统
- 系统启动时自动运行启动脚本
- 延迟 10 秒启动，确保系统完全加载

### 2. Shell 集成
- 在 shell 配置文件中添加检查逻辑
- 每次打开终端时自动检查 Podman 状态
- 如果机器未运行，自动启动

### 3. 错误处理
- 检查 Podman 是否安装
- 处理端口冲突问题
- 提供详细的错误信息和日志

## 日志文件

- **标准输出**：`/tmp/podman-auto-start.log`
- **错误输出**：`/tmp/podman-auto-start.error.log`

查看日志：
```bash
# 查看启动日志
tail -f /tmp/podman-auto-start.log

# 查看错误日志
tail -f /tmp/podman-auto-start.error.log
```

## 服务管理

### 查看服务状态
```bash
launchctl list | grep podman
```

### 手动启动服务
```bash
launchctl load ~/Library/LaunchAgents/com.podman.auto-start.plist
```

### 手动停止服务
```bash
launchctl unload ~/Library/LaunchAgents/com.podman.auto-start.plist
```

## 故障排除

### 1. 服务未自动启动
- 检查服务是否加载：`launchctl list | grep podman`
- 查看错误日志：`cat /tmp/podman-auto-start.error.log`
- 手动运行脚本：`~/bin/podman-auto-start.sh`

### 2. 端口冲突
- 检查端口占用：`lsof -i :53808`
- 重启 Podman 机器：`podman machine stop && podman machine start`

### 3. 权限问题
- 确保脚本有执行权限：`chmod +x ~/bin/podman-auto-start.sh`
- 检查服务文件权限

## 系统要求

- macOS 系统
- Podman 已安装
- Bash shell 支持

## 注意事项

1. **数据安全**：卸载时会创建 shell 配置文件备份
2. **端口冲突**：如果遇到端口冲突，脚本会自动处理
3. **权限要求**：安装过程需要用户权限
4. **兼容性**：支持 zsh 和 bash shell

## 更新日志

- **v1.0.0**：初始版本，提供基本的自动启动功能
- 支持 macOS 系统
- 提供完整的安装和卸载功能
- 集成 shell 自动检查
- 提供便捷命令别名

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个工具。

## 许可证

MIT License


