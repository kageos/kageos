FROM ubuntu:22.04

# 设置非交互模式
ENV DEBIAN_FRONTEND=noninteractive

# 更新系统并安装基础工具
RUN apt update && apt install -y \
    openssh-server \
    curl \
    wget \
    git \
    vim \
    nano \
    htop \
    tree \
    unzip \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release \
    sudo \
    net-tools \
    iputils-ping \
    telnet \
    dnsutils \
    build-essential \
    gcc \
    g++ \
    make \
    cmake \
    gdb \
    valgrind \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev \
    && rm -rf /var/lib/apt/lists/*

# 安装 Node.js 18.x
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - \
    && apt install -y nodejs

# 安装 npm 全局工具
RUN npm install -g \
    yarn \
    pnpm \
    typescript \
    ts-node \
    nodemon \
    pm2 \
    @vue/cli \
    create-react-app \
    express-generator

# 配置 SSH
RUN mkdir /var/run/sshd \
    && echo 'root:password' | chpasswd \
    && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config

# 创建用户目录
RUN mkdir -p /root/.ssh \
    && chmod 700 /root/.ssh

# 设置工作目录
WORKDIR /workspace

# 暴露端口
EXPOSE 22 8080 3000 5000

# 创建启动脚本
RUN echo '#!/bin/bash' > /start.sh \
    && echo 'service ssh start' >> /start.sh \
    && echo 'echo "=== Linux 开发服务器 ==="' >> /start.sh \
    && echo 'echo "SSH 端口: 22 (映射到主机 2222)"' >> /start.sh \
    && echo 'echo "用户名: root"' >> /start.sh \
    && echo 'echo "密码: password"' >> /start.sh \
    && echo 'echo "工作目录: /workspace"' >> /start.sh \
    && echo 'echo "已安装工具: git, vim, nano, htop, python3, nodejs, npm, yarn, pnpm"' >> /start.sh \
    && echo 'echo "Node.js 版本: $(node --version)"' >> /start.sh \
    && echo 'echo "NPM 版本: $(npm --version)"' >> /start.sh \
    && echo 'echo "SSH 连接: ssh root@localhost -p 2222"' >> /start.sh \
    && echo 'echo "================================="' >> /start.sh \
    && echo 'tail -f /dev/null' >> /start.sh \
    && chmod +x /start.sh

CMD ["/start.sh"]



