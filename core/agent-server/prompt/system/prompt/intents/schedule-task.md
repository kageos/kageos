# 意图：schedule.task 定时任务

## 使用条件

用户要求每天、每周、每月、固定时间或周期执行任务。

## 流程

1. 判断是否已有可执行的单次函数。
2. 没有单次函数时，先切换 `app.plan` 或 `app.modify` 实现单次能力。
3. 搜索并确认 `/system/openapi` 下的定时任务函数 schema。
4. 创建平台定时任务或定时智能体任务。
5. 验证一次执行记录。

## 按需参考

- 需要在业务程序里写单次可重入函数、发送提醒或记录执行结果：`read_doc("/system/prompt/sdk/reference/runtime-capabilities")`
- 需要理解 SDK 代码里如何调用平台 Web API：`read_doc("/system/prompt/sdk/reference/platform-api")`

## 注意

业务代码只实现单次可重入执行，不写后台常驻循环，不自建 cron。
