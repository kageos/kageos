---
id: sop.explain-project
name: explain-project
description: 解释项目、分析代码、回答“这个目录/函数/系统是干什么的”时使用。该 skill 默认无副作用，不应写文件、构建或发布。
triggers:
  - 看一下项目
  - 解释
  - 分析
  - 讲清楚
  - 这个项目是干嘛的
  - 梳理
modes:
  - qa
  - execute
  - dev
  - modify
  - agent
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - web_search
  - fetch_url_content
completion:
  - 已读取当前目录结构或目标文件
  - 回答包含系统职责、核心链路和关键文件
  - 未执行写文件、构建、删除、发布等副作用操作
---

# 解释项目 SOP

## 使用条件

用户要求“看一下”“分析一下”“讲清楚”“说说看法”，且没有明确要求修改代码或执行副作用操作时使用本 skill。

## 流程

1. 先用 `read_dir` 看当前目录和函数结构。
2. 根据问题读取相关 Go 文件或文档。
3. 如果问题涉及 SDK 或平台机制，读取 `required_docs`。
4. 用清晰结构说明：这是什么、入口在哪里、数据/调用链路是什么、风险和建议是什么。
5. 不写文件、不 build、不删除、不发布。

## 输出建议

- 先给结论。
- 再讲关键链路。
- 最后给风险和下一步建议。
- 如果信息不足，说明还需要读取哪些文件，而不是编造。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
