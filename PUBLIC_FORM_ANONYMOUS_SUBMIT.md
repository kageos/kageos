# Public Form 匿名提交设计边界

> 用途：沉淀“未登录用户提交 Form”的产品和技术边界。本文只讨论 Public Form Submit，不扩展到匿名访问工作台、服务树、Table 或 Chart。

## 一句话结论

第一阶段只支持：

> 某个 Form 节点开启公开提交后，未登录用户通过公开链接访问这一个 Form，并完成一次提交。

不做：

- 匿名用户进入工作台。
- 匿名用户浏览服务树。
- 匿名用户查询 Table。
- 匿名用户访问 Chart。
- 泛化 guest 角色授权系统。
- 严格一人一次防重复。

这个能力应该被定义为 **Public Form Submit**，不是完整的 guest 权限体系。

## 为什么要做

很多有价值的业务表单面向外部人：

- NPS / 反馈收集。
- 活动报名。
- 联系我们。
- 外部工单提交。
- 客户需求提交。
- 文件收集表单。
- 简单问卷。

这些场景如果要求提交者注册登录，转化会很差。业界表单产品通常也是创建者登录，填写者免登录。

## 为什么不做完整 guest 权限

把未登录用户做成一个普通 guest 角色，然后给节点授权，看似统一，实际会放大边界：

- guest 能不能看服务树？
- guest 能不能读 Table？
- guest 能不能运行 Chart？
- guest 写 Table 算什么权限？
- guest 能不能上传文件？
- guest 能不能触发高成本 Form？
- guest 数据归属怎么判断？
- guest 是否能被搜索、授权、设为负责人？

这些问题会把 MVP 拖成完整 public resource permission 系统。

第一阶段不要做这个泛化。

## 推荐架构

## 当前实现状态

当前代码已经落地第一版 Public Share：

- 登录用户在 Form 节点的“公开链接”页签创建分享。
- 未登录用户访问 `/public/s/{share_id}` 查看单个公开 Form。
- 未登录用户提交 `/public/s/{share_id}/submit`。
- 匿名身份通过 `X-Public-Anonymous-Token` 传递，前端保存到 `localStorage`。
- 后端派生 `guest_anon_xxx` 并写入 `X-Request-User`，runtime / SDK 可通过 `ctx.GetRequestUser()` 读取。
- 分享记录表：`public_share`。
- 分享事件表：`public_share_event`。
- 配置项保持轻量：过期时间、最大提交次数。

这版没有接入完整 guest RBAC，也没有开放服务树、Table、Chart、Docs。

### 后台配置

登录用户在某个 Form 函数节点上开启：

```text
Create public share
```

系统生成公开记录：

```text
public_share
- id
- share_id
- tenant_user
- app
- full_code_path
- resource_type = form
- action = form.submit
- enabled
- created_by
- created_at
- expires_at
- max_uses
- use_count
```

`share_id` 必须不可猜，不能直接暴露内部 `full_code_path` 作为公开入口。

### 匿名访问

未登录用户访问页面：

```text
GET /public/s/{share_id}
```

页面内部读取公开 schema 的接口：

```text
GET /public/api/s/{share_id}
```

只返回这个 Form 的公开渲染信息：

- Form 名称。
- Form 描述。
- Form schema。
- 必要的公开配置。

不返回：

- 服务树。
- app 内其他节点。
- Table 数据。
- 工作台上下文。
- 内部权限信息。

### 匿名提交

未登录用户提交：

```text
POST /public/s/{share_id}/submit
```

当前实现中的提交 API 是：

```text
POST /public/api/s/{share_id}/submit
```

后端校验：

1. `share_id` 存在。
2. `enabled = true`。
3. 未过期。
4. 未超过最大提交数。
5. 绑定节点存在。
6. 绑定节点类型必须是 Form。
7. 该 Form 允许 public submit。
8. 未触发限流。

通过后，后端构造受限上下文，转发给原 Form 执行。

## 匿名身份模型

不要把所有匿名用户都写成同一个 `guest`，否则业务逻辑容易误判：

- 每个用户只能提交一次会误杀。
- 按创建人统计会混在一起。
- 审计看不出是否同一浏览器反复提交。

推荐模型：

```text
权限主体：guest
匿名 actor：guest_anon_xxx
来源：public_form
来源引用：share_id
```

权限判断上，它仍然只是 guest，只能做 public form submit。

审计和业务展示上，使用 `guest_anon_xxx` 区分匿名访问者。

## 稳定匿名 actor

目标：

> 同一个浏览器、同一台机器、同一个站点，访问同一个公开 Form 时生成稳定匿名 actor。

推荐做法：

1. 首次访问 public form 时，后端生成随机 `public_anon_seed`。
2. 前端保存到 `localStorage`，后续通过 `X-Public-Anonymous-Token` 传回。
3. 后端根据 `tenant_user + app + share_id + public_anon_seed` 派生 `guest_anon_xxx`。

示例：

```text
public_anon_seed = signed(random_128bit)
anon_actor_id = "guest_anon_" + HMAC(secret, tenant_user + app + share_id + seed)[0:16]
```

当前实现使用 Header：

```text
X-Public-Anonymous-Token: signed(payload).signature
```

后续如果要提高防篡改和抗 XSS 能力，可以把这层换成 `HttpOnly + Secure + SameSite=Lax` cookie；当前 MVP 先用 header 方案，便于前后端和 API 调试。

### 这个模型能保证什么

可以保证或基本保证：

- 同浏览器同站点稳定。
- 同浏览器同 Form 稳定。
- 审计上能区分不同匿名 actor。
- 不同 share 之间不直接共享同一个 actor id。

不能保证：

- 换浏览器仍然同一个人。
- 换设备仍然同一个人。
- 清 cookie 后仍然同一个人。
- 无痕模式关闭后仍然同一个人。
- 严格一人一次。

## 是否需要改权限系统

不需要大改完整权限系统。

需要做的是一个窄口：

> Public Form Gateway 做 share-level authorization。

也就是：

- 不把 guest 接入完整 RBAC。
- 不给 guest 节点授权入口。
- 不让 guest 访问普通 workspace API。
- 只允许 public submit endpoint 内部根据 share 配置调用绑定 Form。

普通权限系统只需要防止 `guest_anon_xxx` 被当成普通登录用户使用。

如果某些中间件必须识别用户身份，应明确：

```text
guest_anon_* => anonymous public actor
principal => guest
```

## Public Form 与普通 Form 的差异

Public Form 不是简单把普通 Form 裸露出去。

Public Form 必须有额外限制：

- 不允许访问服务树。
- 不允许查询 Table。
- 不允许调用其他函数。
- 不允许更新/删除。
- 不允许使用普通工作台 token。
- 不允许暴露内部错误细节。

Public Form 只允许：

- 读取公开 schema。
- 提交当前 share 绑定的 Form。
- 返回提交结果或成功文案。

## 表单类型限制

有些 Form 不适合开启匿名提交。

MVP 可以先做提示或禁止：

- 包含 `user/users/department/departments` 字段的 Form。
- 依赖 `ctx.GetRequestUser()` 做业务身份判断的 Form。
- 高成本 Form，例如 LLM、OCR、视频处理、Python 大任务。
- 有外部副作用的 Form，例如发 Slack、创建 GitHub issue、调用支付 API。
- 文件上传 Form 由字段 schema / 文件组件自己的策略控制，不放到 public share 配置里。

长期可以引入 `public_safe` 或 `public_risk_level` 配置，但第一阶段可以先用规则和提示控制。

## 防重复提交

匿名场景下不承诺严格防重复。

原因：

- 不登录就没有可靠身份。
- IP 会误伤公司、学校、门店等共享网络。
- Cookie 可清除。
- 浏览器可更换。
- 设备可更换。

MVP 只做防滥用，不做强防重复。

建议底线：

- 前端提交按钮防连点。
- 后端 rate limit。
- payload size 限制。
- share 可关闭。
- share 可过期。
- share 可设置最大提交数。
- 文件上传默认关闭或严格限制。

如果业务需要“一人一次”，后续再做：

- invite token。
- 邮箱验证码。
- 登录提交。
- 业务字段唯一校验。

## 防滥用策略

即使不做强防重复，也必须防滥用：

- IP / share 维度限流。
- 单次请求大小限制。
- 单个 share 每分钟提交限制。
- 单个 anon actor 每分钟提交限制。
- 文件上传大小、类型、数量限制。
- 高成本 Form 默认不允许 public。
- 错误响应不暴露内部路径、token、stack。

## 合规和隐私

匿名提交不等于没有个人数据。

表单可能收集：

- 姓名。
- 邮箱。
- 电话。
- 公司。
- IP。
- 文件。
- 案件、订单、反馈等敏感内容。

因此需要：

- Public Form 页面展示隐私说明或 consent text。
- owner 可配置提交说明。
- 不存原始 IP，优先存 hash。
- 匿名 actor 只用于提交记录、防滥用和审计。
- 不用于广告追踪。
- 设置提交记录和访问标识保留策略。
- 支持 owner 删除提交数据。

## 对外表述

推荐：

> Public Forms let external users submit a specific form without creating an account.

推荐：

> Anonymous submissions are designed for low-friction collection, not strict identity verification.

中文：

> 公开表单用于低摩擦收集外部提交，不承诺严格一人一次。需要严格身份唯一的场景，应使用登录或邀请链接。

## MVP 范围

第一版只做：

1. Form 节点开启/关闭匿名提交。
2. 生成 `share_id`。
3. 匿名访问单个 Form。
4. 匿名提交单个 Form。
5. signed token 生成稳定匿名 actor。
6. share-level authorization。
7. rate limit / payload limit。
8. 可选过期时间和最大提交数。

第一版不做：

- guest RBAC。
- public Table。
- public Chart。
- public Docs。
- public workspace。
- 严格防重复。
- invite token。
- captcha。
- OAuth。
- 外部用户账号系统。

## 后续扩展

如果 Public Form 验证有效，再考虑：

- Public Docs。
- Public Chart。
- Public read-only Table view。
- Invite token。
- CAPTCHA / Turnstile。
- 邮箱验证。
- Public file intake。
- Public Form analytics。

但这些都不属于当前 MVP。
