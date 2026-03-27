# Tini 升级说明

## 问题背景

在灰度发布场景下，当旧版本（PID 1）被关闭时，容器会认为主进程退出，导致容器自动重启。

### 问题流程
```
容器启动 → start.sh 启动 v4 (PID 1)
  ↓
更新：启动 v5（后台进程）
  ↓
v4 流量为 0，被关闭
  ↓
v4 是 PID 1 → 容器认为主进程退出
  ↓
容器退出并重启（--restart=always）
```

## 解决方案

使用 `tini` 作为 PID 1，解决以下问题：
1. ✅ **防止容器退出**：应用版本切换不会导致容器重启
2. ✅ **信号转发**：正确处理 SIGTERM/SIGINT 等信号
3. ✅ **僵尸进程回收**：自动回收僵尸进程
4. ✅ **优雅关闭**：支持应用优雅退出

### 新的进程结构
```
容器启动
  ↓
tini (PID 1) - 永不退出
  ↓
start.sh (PID 2) - 保持运行
  ↓
应用 v4 (PID 3) → 应用 v5 (PID 4)
  ↓
v4 退出 → tini 回收，容器继续运行
```

## 修改内容

### 1. Dockerfile
```dockerfile
# 安装 tini
RUN apk add --no-cache ca-certificates file tzdata tini

# 设置 tini 作为 entrypoint
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/start.sh"]
```

### 2. start.sh
```bash
# 启动应用（后台，不使用 exec）
./"releases/$BINARY_NAME" &
APP_PID=$!

# 保持脚本运行（让 tini 管理子进程）
while true; do
    sleep 3600
done
```

## 验证步骤

### 1. 查看进程树
```bash
# 进入容器
podman exec -it <container_name> sh

# 查看进程
ps aux
# 应该看到：
# PID 1: tini
# PID 2: /start.sh
# PID 3+: 应用进程
```

### 2. 测试版本切换
```bash
# 更新应用
curl -X POST http://localhost:9090/api/v1/app/update/xxx

# 查看容器状态
podman ps -a
# 容器应该保持运行状态，不会重启
```

### 3. 测试优雅关闭
```bash
# 停止容器
podman stop <container_name>

# 查看日志
podman logs <container_name>
# 应该看到应用收到 SIGTERM 并优雅退出
```

## 对比

| 特性 | 修改前 | 修改后 |
|------|--------|--------|
| PID 1 | 应用进程 | tini |
| 版本切换 | 容器重启 ❌ | 容器继续运行 ✅ |
| 信号处理 | 不处理 ❌ | 正确转发 ✅ |
| 僵尸进程 | 可能累积 ❌ | 自动回收 ✅ |
| 优雅关闭 | 不支持 ❌ | 支持 ✅ |

## 镜像信息

- **基础镜像**: alpine:latest
- **新增包**: tini (0.19.0-r3)
- **镜像大小**: 增加约 50KB
- **构建时间**: 无显著变化

## 生产建议

1. ✅ **保留 --restart=always**：异常情况下仍能自动重启
2. ✅ **监控进程数**：确认僵尸进程被正确回收
3. ✅ **测试信号处理**：验证优雅关闭流程
4. ✅ **灰度发布**：验证多版本共存场景

## 参考

- [Tini 官方文档](https://github.com/krallin/tini)
- [Docker 最佳实践](https://docs.docker.com/develop/dev-best-practices/)
- [PID 1 问题详解](https://blog.phusion.nl/2015/01/20/docker-and-the-pid-1-zombie-reaping-problem/)

