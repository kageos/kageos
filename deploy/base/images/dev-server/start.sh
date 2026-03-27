#!/bin/bash

# 构建开发服务器镜像
echo "构建 Linux 开发服务器镜像..."
podman build -f Dockerfile.dev-server -t my-linux-dev-server .

if [ $? -eq 0 ]; then
    echo "镜像构建成功！"
    
    # 停止并删除已存在的容器
    echo "停止已存在的容器..."
    podman stop my-linux-dev-server 2>/dev/null || true
    podman rm my-linux-dev-server 2>/dev/null || true
    
    # 启动新容器
    echo "启动 Linux 开发服务器..."
    podman run -d \
        --name my-linux-dev-server \
        -p 2222:22 \
        -p 8080:8080 \
        -p 3000:3000 \
        -p 5000:5000 \
        -v $(pwd)/../workspace:/workspace \
        my-linux-dev-server
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "=== Linux 开发服务器启动成功！ ==="
        echo "SSH 连接: ssh root@localhost -p 2222"
        echo "密码: password"
        echo "工作目录: /workspace"
        echo ""
        echo "查看日志: podman logs my-linux-dev-server"
        echo "进入容器: podman exec -it my-linux-dev-server /bin/bash"
        echo "停止容器: podman stop my-linux-dev-server"
        echo "================================="
    else
        echo "容器启动失败！"
        exit 1
    fi
else
    echo "镜像构建失败！"
    exit 1
fi



