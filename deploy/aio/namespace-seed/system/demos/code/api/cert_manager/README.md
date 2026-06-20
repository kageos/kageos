# Smart Certificate Manager

智能证书管理是一个面向运维、安全和平台团队的 KageOS SDK demo，用来管理域名证书资产、证书文件、到期巡检、续期任务和负责人提醒。

这个 demo 的边界很明确：系统可以保存证书、私钥和证书包，支持下载和审计；可以读取公网 443 证书做巡检；可以在证书临近过期时提醒负责人并创建续期任务；但不会把证书自动部署到服务器、网关、CDN 或负载均衡。

## MVP 范围

- 维护域名资产：域名、环境、负责人、通知人、验证方式、DNS 服务商、部署目标、提前提醒天数。
- 使用 `files` 组件保存证书文件、私钥文件和证书包。
- 导入证书时解析证书有效期、签发者、SAN、序列号、SHA256 指纹和域名匹配结果。
- 手动扫描公网 TLS 证书，记录巡检结果。
- 内置每日证书到期巡检定时任务，开箱即用。
- 发现即将过期、已过期或检查失败时提醒负责人。
- 可选自动创建续期任务，续期结果上传后进入证书资产库并标记为待部署。

## 不做的事

- 不直接部署证书。
- 不重载 Nginx、网关、CDN 或负载均衡。
- 不在 demo 内直接持有云厂商 DNS API 密钥。
- 不保证公网探测能覆盖所有内网、CDN 多边缘节点或 SNI 特殊场景。
- 不替代正式 CA 合规审计，证书签发和部署仍应由受控流程完成。

## 路由

- `GET /cert_manager/cert_domain_list.table`：证书域名管理。
- `GET /cert_manager/cert_asset_list.table`：证书资产库。
- `POST /cert_manager/cert_import.form`：导入证书文件。
- `GET /cert_manager/cert_scan_record_list.table`：证书巡检记录。
- `POST /cert_manager/cert_public_scan.form`：扫描公网证书。
- `POST /cert_manager/cert_expiry_sweep.form`：证书到期巡检。
- `GET /cert_manager/cert_renewal_task_list.table`：证书续期任务。
- `POST /cert_manager/cert_renewal_create.form`：创建证书续期任务。
- `POST /cert_manager/cert_renewal_result.form`：登记续期结果。

## 内置定时任务

`cert_expiry_sweep.form` 的 `FormTemplate` 已配置 `Schedules`：

- Code：`cert_expiry_daily_sweep`
- Cron：`0 9 * * *`
- Body：`CertExpirySweepReq{WarningDays: 30, Notify: true, AutoCreateRenewal: true}`

安装后每天 09:00 自动扫描启用巡检的域名。开启自动续期任务的域名，如果进入即将过期或已过期状态，会自动生成一条续期任务。

## 推荐流程

1. 在“证书域名管理”录入域名和负责人。
2. 通过“扫描公网证书”初始化当前公网证书状态。
3. 使用每日到期巡检自动提醒风险。
4. 风险出现后创建或自动生成续期任务。
5. 管理员或受控执行器在外部完成证书申请。
6. 在“登记续期结果”上传新证书、私钥和证书包。
7. 平台团队下载证书文件，走已有发布流程部署。

## 后续可扩展

- 接入 ACME 客户端，把“自动续期任务”推进到自动申请证书，但仍不自动部署。
- 为 DNS-01 增加按租户隔离的 DNS Provider 凭证托管。
- 增加审批流，私钥下载前需要负责人或安全团队确认。
- 增加证书变更通知和指纹漂移监控。
- 增加导出清单，支持安全审计和资产盘点。
