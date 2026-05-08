# 意图：app.operate_test 应用操作和测试

## 使用条件

用户要求查表、提交表单、查图表、测试刚创建的应用、验证核心流程或操作已有函数。

## 执行前确认

开始操作前必须读取或确认：

1. 目标目录结构
2. 目标函数 schema

## 按需参考

- 测试时需要理解写操作副作用、Table 回调、当前用户/部门、消息或 Python 输出：`read_doc("/system/prompt/sdk/reference/runtime-capabilities")`
- 运行失败指向 build/schema/widget/路由问题：切换 `app.build_fix`；细节不足时读 `read_doc("/system/prompt/sdk/reference/build-validation")`

## 流程

1. 确认目标函数 `full_code_path`。
2. 确认 schema、必填字段、枚举、文件字段、筛选字段和默认值行为。
3. 设计测试用例。
4. Table 先 `run_table_search`；只有确认支持写入时才 create/update/delete。
5. Form 用 `run_form_submit`。
6. Chart 用 `run_chart_query`。
7. OnSelectFuzzy 用 `run_on_select_fuzzy`。

## 下一步

- 测试失败且是业务问题：切换 `app.modify`。
- 测试失败且是 build/schema 问题：切换 `app.build_fix`。
- 测试通过但要周期执行：切换 `schedule.task`。
