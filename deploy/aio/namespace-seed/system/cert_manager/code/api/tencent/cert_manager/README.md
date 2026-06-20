# 腾讯云证书管家

这是一个小而专的证书管理应用：只对接腾讯云 DNSPod，使用 Let's Encrypt ACME DNS-01 自动签发和续期证书。系统负责申请、巡检、提醒、归档和下载证书文件，不负责把证书部署到用户服务器。

当前目录遵循“厂商父目录 + 能力模块目录”的复制规则：`code/api/tencent/cert_manager` 中 `tencent` 表示厂商，`cert_manager` 保持能力模块自身英文 code。路由前缀为 `/tencent/cert_manager`。

## 核心流程

1. 在 `腾讯云配置` 中保存 SecretId、SecretKey 和 ACME 邮箱。
2. 在 `证书域名管理` 中维护需要管理的域名、SAN、负责人、通知人和自动续期开关。
3. 用户通过 `申请/续期证书` 提交申请，系统自动创建 `_acme-challenge` TXT 记录。
4. 系统轮询 DNS，确认 TXT 生效后提交 ACME 验证。
5. 签发成功后保存 `cert.pem`、`chain.pem`、`fullchain.pem`、`private.key` 和 ZIP 证书包到 files 组件。
6. `证书自动续期巡检` 每天 03:00 检查启用自动续期的域名，未签发或即将过期时自动续期。

## 腾讯云密钥权限

建议创建专用 CAM 子用户或角色的 SecretId/SecretKey，只授予 DNSPod 需要的最小权限。当前实现会调用这些 DNSPod API：

- `DescribeDomainList`: 验证密钥可用性。
- `DescribeRecordList`: 查找托管域名。
- `CreateRecord`: 创建 ACME DNS-01 TXT 记录。
- `DeleteRecord`: 签发完成后清理 TXT 记录。

系统会保存 Secret 密文，不会在列表中展示明文。

## 功能入口

- `configs.table`: 腾讯云配置列表
- `config.form`: 新增或更新腾讯云配置
- `config_status.form`: 检查腾讯云 Secret 与 ACME Directory
- `domains.table`: 证书域名管理
- `requests.table`: 证书申请与续期记录
- `issue.form`: 手动申请或续期证书
- `auto_renew_sweep.form`: 自动续期巡检，内置每日 03:00 定时任务
- `assets.table`: 证书资产库

## 设计边界

- 只支持腾讯云 DNSPod 自动化，后续接入其他厂商建议复制该目录做专版实现。
- 不自动部署证书，不碰用户服务器、Ingress、负载均衡或 CDN 配置。
- 签发后默认删除 ACME TXT 记录，避免 DNS 中长期留下临时 challenge。
- 私钥会作为证书资产保存，下载和分发由用户自行控制。
