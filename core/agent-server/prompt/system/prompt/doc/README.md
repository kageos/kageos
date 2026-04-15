# 公用文档

本目录存放所有模式共用的文档与配置，对应树上的 `/system/prompt/doc`。

## 文件说明

- **workspace-env-template.md**：工作台环境信息模板（纯数据），占位符由代码填充。
- **目录索引**：不再单独落盘为 JSON，运行时根据 `/system/prompt` 树上的目录与文档元数据动态生成。

## 与 mode 的关系

- **doc/**：公用内容，所有工作台模式共享。
- **mode/<code>/**：各模式个性化（system_prompt、first_assistant、config.json 等）。
