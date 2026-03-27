# Hub 前端 Dockerfile
# 构建: docker build -t ai-agent-os-hub-frontend:latest -f deploy/base/images/hub-frontend/Dockerfile .

# ========== Stage 1: 构建 ==========
FROM node:20-slim AS builder

WORKDIR /src

# 先复制 package.json 利用缓存
COPY enterprise_impl/hub/frontend/package.json enterprise_impl/hub/frontend/package-lock.json* ./
RUN npm ci --prefer-offline

# 复制源码并构建
COPY enterprise_impl/hub/frontend/ ./
RUN npm run build-only

# ========== Stage 2: Nginx 托管 ==========
FROM nginx:stable-alpine

# 复制构建产物
COPY --from=builder /src/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY deploy/base/images/hub-frontend/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
