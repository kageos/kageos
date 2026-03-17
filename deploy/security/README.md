# 容器安全模块

## 背景

容器内运行的是 AI 生成的代码和第三方包，无法保证不执行删除操作。  
必须在 **内核层面** 禁止容器进程删除 `/app/code` 和 `/app/workplace`，  
只允许 runtime 在宿主机侧通过受控接口删除。

## 工作原理

1. **runtime 启动时** 自动检测宿主机 LSM 类型（AppArmor 或 SELinux），结果缓存。
2. **起容器时** 按检测结果只启用对应的一种安全策略，不同时启用两种。
3. **策略效果** 相同：容器内进程可以读写 code/workplace，但**不能删除**；宿主机上的 runtime 不受限制。

## 快速开始

### 第一步：确认宿主机 LSM 类型

查看 runtime 启动日志中的一行：
```
[LSM] 检测结果: apparmor (配置 lsm_mode=auto)
```
或
```
[LSM] 检测结果: selinux (配置 lsm_mode=auto)
```

### 第二步：安装对应策略

**AppArmor（Ubuntu / Debian 等）**
```bash
cd deploy/security/apparmor
sudo bash install.sh
```

**SELinux（RHEL / Fedora / CentOS 等）**
```bash
cd deploy/security/selinux
sudo bash install.sh /path/to/namespace
```

### 第三步：配置 app-runtime

在 `configs/dev/app-runtime.yaml` 或 `configs/prod/app-runtime.yaml` 中：

```yaml
container:
  lsm_mode: "auto"                          # 自动检测，通常不用改
  apparmor_profile: "ai-agent-os-app"        # AppArmor 环境填写；SELinux 环境留空
```

重启 runtime 即可生效。

## 目录结构

```
deploy/security/
  README.md              # 本文件
  apparmor/
    ai-agent-os-app      # AppArmor profile（路径型：deny unlink）
    install.sh            # 一键安装（拷贝 + 加载）
  selinux/
    ai-agent-os-app.te   # SELinux 策略源文件（类型型：自定义 type + 受限权限）
    install.sh            # 一键安装（编译 + 安装模块 + 打标签）
```

## 验证

**AppArmor**
```bash
# 宿主机上确认 profile 已加载
sudo aa-status | grep ai-agent-os-app

# 容器内确认 profile 生效
cat /proc/1/attr/current
# 应输出：ai-agent-os-app

# 容器内尝试删除（应被拒绝）
rm /app/code/some-file    # Permission denied
```

**SELinux**
```bash
# 宿主机上确认模块已安装
semodule -l | grep ai_agent_os_app

# 宿主机上确认标签
ls -lZ namespace/user/app/code/
# 应显示 ai_agent_os_data_t

# 容器内尝试删除（应被拒绝）
rm /app/code/some-file    # Permission denied
```

## 注意事项

- **两种 LSM 互斥**：一台宿主机只会有一种，runtime 按检测结果只启用一种，不会冲突。
- **宿主机不受限**：runtime 在宿主机上的删除操作（通过 DeleteApp / DeleteFile 等接口）不受 LSM 限制。
- **新文件自动受保护**：容器内新创建的文件继承父目录的安全属性，同样不能被删除。
- **无 LSM 时**：若检测为 none，runtime 会打 warning 日志，不阻塞启动，但内核级防删不生效。
