# 多阶段构建 Dockerfile
# 用于构建 AI Agent OS 服务

# 构建阶段
FROM golang:1.23-alpine AS builder
# 注意：Go 1.24 可能还未发布，使用 1.23 或 latest

# 安装必要的构建工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建单实例部署版本（统一启动所有服务）
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ai-agent-os-single \
    -ldflags="-w -s" \
    ./core/cmd/main

# 构建微服务部署版本（每个服务独立）
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app-server \
    -ldflags="-w -s" \
    ./core/app-server/cmd/app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/agent-server \
    -ldflags="-w -s" \
    ./core/agent-server/cmd/app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app-runtime \
    -ldflags="-w -s" \
    ./core/app-runtime/cmd/app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/app-storage \
    -ldflags="-w -s" \
    ./core/app-storage/cmd/app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api-gateway \
    -ldflags="-w -s" \
    ./core/api-gateway/cmd/app

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/control-service \
    -ldflags="-w -s" \
    ./core/control-service/cmd/app

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin /app/bin

# 复制配置文件
COPY --from=builder /app/configs /app/configs

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口（所有服务的端口）
EXPOSE 9090 9091 9092 9093 9095 9096

# 默认命令（单实例部署）
CMD ["./bin/ai-agent-os-single"]

