---
name: kageos
description: 统一编排 kageos 工作空间目录从需求设计、开发、本地检查、平台 build/update、真实业务验收、bundle 导出到 Hub 投稿和状态确认的完整交付闭环。用户要求从头做到发布、继续一次交付、查看交付状态，或不确定该使用 kageos-developer、kageos-operator、kageos-hub-publisher 中哪一个时使用。
---

# kageos

这是 kageos 套件的统一入口。名称只表示产品 `kageos`，不要扩写、改写或解释名称。

本 skill 负责编排，不替代三个专业 skill：

- `kageos-developer`：目录设计、实现、本地检查和平台 build/update。
- `kageos-operator`：在已构建目录上执行真实业务场景、清理合成数据并产出验证报告。
- `kageos-hub-publisher`：校验发布证据、准备媒体和元数据、获得最终确认后提交 Hub。

开始完整交付前，依次读取本 skill、当前阶段对应的专业 skill，以及该专业 skill 明确要求的 references。不要一次把三个专业 skill 的全部 references 都载入上下文。

## 意图路由

| 用户意图 | 路由 |
|---|---|
| 设计、开发、修改、修 bug、本地检查、平台构建 | `kageos-developer` |
| 操作现有目录、真实测试、验收、证明业务闭环 | `kageos-operator` |
| 发布已验收目录、更新 Hub 版本、查询投稿 | `kageos-hub-publisher` |
| 从设计做到发布、继续上次交付、查看全链路状态 | 本 skill 编排完整流程 |

用户只要求单阶段工作时，不得擅自扩展到后续写操作或发布。用户明确要求“全链路、闭环、做到发布”时，授权的是推进整个流程，不代表跳过 Operator 写入确认或 Publisher 最终提交确认。

## 完整流水线

严格按以下顺序推进：

```text
design
→ local_build
→ platform_build
→ operator_verify
→ bundle
→ publish_prepare
→ publish_submit
→ publish_status
```

阶段定义：

1. `design`：明确业务问题、目录、模板组合、主入口、暂不做和验收场景。
2. `local_build`：实现代码并通过 gofmt、go test、go build。
3. `platform_build`：执行真实 build/update，并从实时 service tree 证明新版本已经生效。
4. `operator_verify`：按真实 schema 执行业务场景、读回结果、验证自动化、清理测试数据，必须生成 `verified` 的 `kageos.operator-report.v1`。
5. `bundle`：在验收后导出并校验 `capability.bundle.v1`；记录文件 SHA-256。
6. `publish_prepare`：准备安全截图、说明、版本事实和最终 submission JSON，不提交。
7. `publish_submit`：向用户展示最终内容并获得紧邻提交动作的明确确认后才提交。
8. `publish_status`：查询 Hub，确认 submission ID、revision、目录身份、版本和真实状态。

任何阶段失败都记录为 `blocked`，保留已经通过的阶段和证据，从失败处修复后继续。不得为了让状态变绿而编辑证据文件。

## 交付记录

完整流程必须建立 `kageos.delivery-run.v1` JSON。先读 `references/delivery-run-contract.md`，然后用本 skill 的脚本创建和推进：

```bash
python3 scripts/delivery_run.py init \
  --output /tmp/example.delivery-run.json \
  --directory /user/app/package

python3 scripts/delivery_run.py record \
  --run /tmp/example.delivery-run.json \
  --stage design \
  --status passed \
  --note "最小目录设计已确认"

python3 scripts/delivery_run.py validate \
  --run /tmp/example.delivery-run.json

# 代码或平台状态在验收后发生变化时，使该阶段及后续证据失效
python3 scripts/delivery_run.py reset \
  --run /tmp/example.delivery-run.json \
  --from-stage platform_build \
  --reason "目录代码已更新"
```

在不同 Agent/客户端中，使用该 skill 自己目录下的脚本绝对路径或宿主提供的 skill-directory 变量；不要假定当前工作目录就是 skill 目录。

交付记录只保存阶段状态、非敏感说明、制品路径和哈希。严禁写入 token、Cookie、Authorization header、签名 URL、个人资料或客户数据。

## 强制门禁

- `local_build` 通过不代表平台已生效；必须单独完成 `platform_build`。
- 发布链路中的 Operator report 不是可选制品，必须落盘并渲染成人类可读的 Markdown/HTML。
- `operator_verify` 之后如果目录代码或平台版本改变，原报告作废，回到 `platform_build` 和 `operator_verify`。
- Bundle 必须在验收后导出。Publisher 上传前重新计算哈希并与交付记录一致。
- Publisher 不得接受 `blocked`、缺失、目录不一致或早于最新平台构建的 Operator report。
- `publish_prepare` 可以自动完成；`publish_submit` 必须获得最终确认。
- 发布成功不能只依赖提交接口返回；必须完成 `publish_status` 查询。

## 授权边界

完整流程有两个不可合并的确认点：

1. Operator 第一次真实写入前：展示准确调用、合成标记、副作用和清理动作，获得一次确认。
2. Publisher 最终提交前：展示名称、摘要、版本、截图说明和目标 Hub，获得一次确认。

代码编辑、本地检查、只读 discovery、报告渲染和投稿状态查询可在用户原始范围内连续推进。

## 完成标准

只有以下条件全部满足，才可以说“完整发布闭环已完成”：

- 八个阶段全部 `passed`；
- Operator report 为 `verified` 且清理成功；
- Bundle schema、目录身份、版本和 SHA-256 已校验；
- Hub 返回可追踪的 submission ID/revision；
- Hub 查询显示预期目录与版本处于已提交、审核中或已发布状态。

最终向用户返回：目录、平台构建事实、验证报告链接、Bundle 路径和哈希、Hub submission ID/revision/status，以及任何仍需人工处理的事项。
