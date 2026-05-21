# 海外版英文体验扫描清单

## 背景

根据 `plan.md`，项目第一阶段主要面向海外个人用户、独立开发者和中小企业。英文体验需要优先覆盖用户第一次使用时会经过的核心路径。

本清单先记录开发阶段扫描出的高优先级中文硬编码位置，后续可以按页面逐步迁移到 i18n。

## 优先级原则

优先处理用户可见内容，暂不处理代码注释、内部 README、测试用例中的中文。

优先级从高到低：

1. 登录、注册、忘记密码、测试用户入口
2. 工作空间首页、创建工作空间、切换工作空间
3. 服务树：创建目录、创建函数、创建文档、重命名、删除
4. Form/Table/Chart 的核心操作按钮、空状态、错误提示
5. 权限管理、操作日志、成员邀请
6. 内部调试弹窗、开发辅助页面

## P0 需要优先处理的文件

### 认证流程

- `web/src/architecture/presentation/features/auth/pages/LoginPage.vue`
- `web/src/architecture/presentation/features/auth/pages/RegisterPage.vue`
- `web/src/architecture/presentation/features/auth/pages/ForgotPasswordPage.vue`
- `web/src/architecture/presentation/features/auth/pages/CreateTestUserPage.vue`

这些页面是海外用户第一眼会看到的入口，不能出现中文表单校验、按钮、提示或错误消息。

### 核心工作区

- `web/src/architecture/presentation/views/WorkspaceView.vue`
- `web/src/architecture/presentation/views/FormView.vue`
- `web/src/architecture/presentation/components/WorkspaceHeader.vue`
- `web/src/architecture/presentation/components/WorkspaceFunctionRenderer.vue`
- `web/src/architecture/presentation/components/WorkspaceFunctionTabsPanel.vue`

这里影响创建、打开、提交和查看结果的主链路。

### 服务树和资源创建

- `web/src/architecture/presentation/components/ServiceTreePanel.vue`
- `web/src/architecture/presentation/components/WorkspaceCreateDocsDialog.vue`
- `web/src/architecture/presentation/components/FormDialog.vue`
- `web/src/architecture/presentation/components/PackageDetailContent.vue`
- `web/src/architecture/presentation/components/PackageDetailChildrenGrid.vue`

这里影响用户能否理解目录、函数、文档这些核心概念。

### 操作结果和可追溯能力

- `web/src/architecture/presentation/components/OperateLogSection.vue`
- `web/src/architecture/presentation/components/TeamAccessDialog.vue`
- `web/src/architecture/presentation/components/TeamAccessPanel.vue`

权限和操作日志是小团队协作的信任基础，英文版也要清晰。

## P1 需要跟进的文件

- `web/src/architecture/presentation/components/ChartRenderer.vue`
- `web/src/architecture/presentation/components/MiniWorkstationSessionDock.vue`
- `web/src/architecture/presentation/components/MiniWorkstationDisplayFieldCard.vue`
- `web/src/architecture/presentation/components/GlobalResourceSearchDialog.vue`
- `web/src/architecture/infrastructure/upload/index.ts`

这些会影响体验完整度，但可以在 P0 后处理。

## 迁移要求

1. 新增用户可见文案必须进入 i18n locale 文件。
2. 不要在 `.vue` template 或业务 composable 中新增中文硬编码。
3. 错误提示、空状态、按钮、表单校验都要覆盖。
4. 模板名称和模板说明后续应英文优先，中文可作为内部补充。
5. 对动态业务字段名称暂不强制翻译，因为它们来自用户自己创建的表单/表格。

## 建议执行顺序

1. 先处理认证页。
2. 再处理 Workspace 主链路。
3. 然后处理服务树资源创建和删除流程。
4. 最后处理权限、操作日志、上传和图表。

## 扫描命令

后续可以用下面命令继续扫描：

```bash
rg -n "[\\p{Han}]" web/src/architecture/presentation web/src/architecture/infrastructure --glob '*.vue' --glob '*.ts' --glob '!**/*.d.ts' --glob '!**/shared/i18n/**'
```

注意：扫描结果里会包含大量中文注释，处理时只看用户可见文案。
