# 可选：域名 → 本机 8999 / 8998（反代）

## 你想做的事

- **`geeleo.com`**（主域，示例）→ 反代到本机 **`127.0.0.1:8999`**（主站，与现在 Nginx 里 Web 站点一致）
- **`hub.geeleo.com`** → 反代到 **`127.0.0.1:8998`**（Hub 前端）

## 是否可行

**可以，而且天生适合「可选」**：**不解析域名 = 不用这套配置**；**解析过来 = 启用反代即可**。

这是标准「外层 Nginx 终结域名 + 内层仍是现有 Embedding 站点」：

- 外层：监听 **80/443**，`server_name` 为你的域名，`proxy_pass` 到 **8999 / 8998**
- 内层：保持现有 **`nginx-server.conf`**（8999、8998、静态、`/storage` 等反代 9090），**不用改**内层逻辑

**未把域名解析到本机时**：只要不 `sites-enabled` 启用域名配置，**没有任何影响**；用户继续用 `http://IP:8999` 即可。

## 解析到本机后需要一起对齐的配置（应用层）

否则会出现「页面能开，上传/预签名/copy_url 错」：

| 场景 | 建议 |
|------|------|
| 主站用 **HTTPS** | `deploy/config/local/` 里 **`hub.yaml`**：`public_host`、`os.base_url` 用 **`https://geeleo.com`**（含协议与端口若非常规） |
| 浏览器访问 MinIO 经主域路径 | **`app-storage.yaml`**：`storage.minio.cdn_domain` 与浏览器实际访问一致，如 **`https://geeleo.com`**（与 Nginx 里 `/ai-agent-os/` 反代一致） |
| Hub 子域 | 若 Hub 只从 **hub.geeleo.com** 打开，`hub.yaml` 里主站相关 URL 要按你产品期望改成子域或主域（与「试用跳转」一致即可） |

执行覆盖：

```bash
bash deploy/embedding/scripts/embedding.sh local
```

并重启 core/hub（或 `embedding.sh restart`）。

## 操作顺序建议

1. 本机确认 **`http://127.0.0.1:8999`**、**`:8998`** 正常（Embedding 已部署）。
2. **DNS** A/AAAA 指到服务器公网 IP。
3. **推荐**：直接执行 **`bash deploy/embedding/scripts/embedding.sh nginx`**（或 **`init` / `update`** 里已包含 nginx）。脚本会把 **[nginx-domain-proxy.example.conf](nginx-domain-proxy.example.conf)** 安装到 **`/etc/nginx/conf.d/ai-agent-os-domain.conf`** 并 reload。  
   - 自定义域名：在 **`deploy/config/local/nginx-domain-proxy.conf`** 放你自己的 `server` 配置（存在则优先于 example）。  
   - 不需要 80 反代：**`EMBEDDING_SKIP_NGINX_DOMAIN=1`** 再执行 nginx 命令。
4. 若手工维护：也可复制 example → `conf.d`，再 **`nginx -t` + reload**。
5. 按上表改 **local yaml**，`local` + 重启后端。

## 示例文件

- **[nginx-domain-proxy.example.conf](nginx-domain-proxy.example.conf)**：含 **HTTP 反代** + 注释里的 **HTTPS / certbot** 说明；**默认不安装**，按需手工启用。
