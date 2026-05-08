# 意图：app.modify 应用修改

## 使用条件

用户要求修改已有应用、已有目录、已有函数或已有代码。

## 按需参考

- 程序里发送消息、取当前用户/部门、事务、副作用顺序、Python 运行时、Table 回调高级能力：`read_doc("/system/prompt/sdk/reference/runtime-capabilities")`
- 需要在 SDK 代码里调用平台 Web API：`read_doc("/system/prompt/sdk/reference/platform-api")`
- build 或启动期 schema/widget/路由校验失败：切换 `app.build_fix`；细节不足时读 `read_doc("/system/prompt/sdk/reference/build-validation")`

## 二级分类

修改前必须先判断具体类型：

- 字段改名
- 新增字段
- 删除字段
- 修改 widget
- 新增或修改 select 选项
- 修改搜索条件
- 新增 OnSelectFuzzy
- 新增 Table 回调逻辑
- 新增消息通知
- 新增 link 跳转
- 新增 Form/Table/Chart
- 修改 Chart 指标
- 修业务 bug

## 流程

1. 读取目录和相关 Go 文件。
2. 判断修改类型并读取专项文档。
3. 小改优先 `search_replace_file`，大改或新增文件用 `write_go_file`。
4. 写入/替换工具返回文件级非阻断诊断时，若本轮还有计划文件未写完，先记录并继续写完；完整落盘后再批量修复诊断并统一 `build_workspace`。
5. build 成功后切换 `app.operate_test` 验证受影响路径。

## 注意

不要做无关重构，不要修改 `init_.go`，不要重造平台能力。
