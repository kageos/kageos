# Web 主前端 Dockerfile
# 构建: docker build -t kageos-web:latest -f deploy/base/images/web/Dockerfile .

# ========== Stage 1: 构建 ==========
FROM node:20-slim AS builder

WORKDIR /src

# 先复制 package.json 利用缓存
COPY web/package.json web/package-lock.json* ./
RUN npm ci --prefer-offline

# 复制源码
COPY web/ ./

# 覆盖生产环境变量：Docker 部署通过 Nginx 反代，API 使用相对路径
RUN echo 'VITE_API_BASE_URL=' > .env.production && \
    echo 'VITE_APP_TITLE=Kageos' >> .env.production && \
    echo 'VITE_APP_ENV=production' >> .env.production

RUN npm run build-only

# ========== Stage 2: Nginx 托管 ==========
FROM nginx:stable-alpine

# 复制构建产物
COPY --from=builder /src/dist /usr/share/nginx/html

# 复制 Nginx 配置
COPY deploy/base/images/web/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
