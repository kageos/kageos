---
name: kageos-hub-publisher
description: 使用范围受限的 Sona 个人访问 token、R2/S3 直传、AI 辅助元数据和带可编辑说明的浏览器截图，把已经真实验收的 kageos 工作空间目录打包并发布到 kageos Hub。在 kageos-operator 之后，用户要求发布、投稿、更新或查询 Hub 目录状态时使用。
---

# kageos Hub 发布

发布真实证据，不发布未经测试的 Demo。执行前读取 `references/publishing-sop.md`。

## 必要输入

- 状态为 `verified` 的 Operator 报告。
- 目录 `full_code_path` 和可导出的 `capability.bundle.v1`。
- 通过完整 `kageos` 流程发布时，当前 `kageos.delivery-run.v1` 记录。
- Hub 基础地址，默认 `https://hub.kageos.ai/api/v1`。
- 具有 `uploads:write` 和 `hub:publish` scopes 的 `HUB_PUBLISH_TOKEN`。
- 用户可选提供的工作空间浏览器地址。

不得记录或嵌入 token。不得索要 R2 凭证；所有文件都通过 Sona upload intent 后直接 PUT 到 R2/S3。

## 工作流

1. 拒绝缺失、过期、blocked、目录不匹配或早于最新平台构建的 Operator 报告。完整交付时先校验交付记录，并要求 `operator_verify` 及之前阶段全部通过。
2. 导出并校验确定性目录 Bundle。不得把 `SKILL.md` 当作 kageos 工作空间目录制品。
3. 如果有浏览器地址，使用可用的 Browser 或 Chrome 控制能力：
   - 打开准确地址并操作已经验收的业务场景；
   - 在一致桌面尺寸下截取 2–6 张关键视口；
   - 截取产生结果后的状态，不截空白配置页；
   - 检查 token、邮箱、电话、客户名称、私有地址和无关标签页；
   - 丢弃不安全截图，换用合成数据重新执行；
   - 为每张截图写简洁、可编辑的说明，解释用户价值和可见证据。
4. 使用 `scripts/hub_publish.py upload` 上传 Bundle。截图先写 SOP 规定的媒体清单，再运行 `scripts/hub_publish.py prepare`；它通过 Sona intent 上传本地文件，生成有序 gallery，并把选中图片加入 `description_html`。
5. 可以调用 Hub AI assist，根据 Bundle 事实和 Operator 证据生成草稿；不得把建议当事实，也不得虚构能力。
6. 检查最终 submission JSON。第一张 gallery 图片是目录封面；每项都必须有 `url`、`kind`、有意义的 `alt` 和 `caption`。正文截图必须解释可见结果，不能重复展示空 UI。
7. 向用户展示最终名称、摘要、版本事实、截图说明和目标 Hub，并在紧邻提交动作前获得明确确认。
8. 使用 `scripts/hub_publish.py submit` 投稿，再查询 `status`，直到列表中出现该 submission。完整交付时把 `publish_submit` 和 `publish_status` 证据写入交付记录，不得保存凭证或签名 URL。

## 更新版本

继续使用同一目录 code/namespace。Hub 分配下一个公开版本；Bundle 的 release version 作为来源证据保留，并填写真实更新说明。更新不得创建第二个目录身份。

## 停止条件

以下任一情况都停止，不提交：验收不是 `verified`、包校验失败、截图泄露敏感信息、token scopes 不足、上传完成失败，或用户尚未确认最终投稿内容。
