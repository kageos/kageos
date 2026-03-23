# Nginx（裸机静态 + 反代）

与 **Embedding** 一体维护的站点配置：

| 文件 | 说明 |
|------|------|
| **`nginx-server.conf`** | Web **8999**、Hub **8998**；静态根目录 **`/opt/ai-agent-os/...`**（由 **`embedding.sh nginx`** 同步）；API 等反代 **9090** |
| **`nginx-domain-proxy.example.conf`** | **可选**：`geeleo.com` → **8999**、`hub.geeleo.com` → **8998**（仅当 DNS 已指向本机时复制到 `sites-available` 并启用） |
| **`DOMAIN_PROXY.md`** | 域名反代可行性、与应用配置（`hub` / `app-storage`）对齐说明 |

**不要**把本目录当作 Nginx 的 `include` 路径直接挂载；应使用脚本复制到 `/etc/nginx/sites-available/` 并 reload。

总说明见 **[../README.md](../README.md)**。
