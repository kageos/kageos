# Nginx（裸机静态 + 反代）

与 **Embedding** 一体维护的站点配置：

| 文件 | 说明 |
|------|------|
| **`nginx-server.conf`** | Web **8999**、Hub **8998**；`root` 占位为 `/opt/ai-agent-os/...`，由 **`embedding.sh nginx`** / **`deploy/server-deploy.sh`** 安装时 `sed` 成仓库内真实 `web/dist`、`hub-frontend/dist` |

**不要**把本目录当作 Nginx 的 `include` 路径直接挂载；应使用脚本复制到 `/etc/nginx/sites-available/` 并 reload。

总说明见 **[../README.md](../README.md)**。
