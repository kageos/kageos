# 删除 generate_code 工具分析

## 一、generate_code 当前作用

1. **唯一作用**：模型调用 `generate_code(file_name)` 后，返回一句「已记录将生成 xxx.go，请在本轮或下一轮输出 markdown 代码块，输出完成后调用 write_package_code」。
2. **真实写文件**：发生在 `write_package_code` 里，与 generate_code 无直接耦合；generate_code 只是给「从历史里找代码」的兜底逻辑提供一个**锚点**（那条 tool 消息 + 下一条 assistant 的 content）。

## 二、write_package_code 目前拿代码的两种方式

| 方式 | 说明 |
|------|------|
| **推荐** | 本条消息里用 `<var><key>名</key><value>代码</value></var>` 定义变量，参数里传 `$source_code: "名"`，由变量解析得到 source_code。 |
| **兜底** | 未传 `$source_code` 时，从会话历史里找「最后一条内容含“已记录将生成”的 tool 消息」，再取下一条 assistant 的 content，从中解析 markdown 代码块。 |

兜底依赖「先调过 generate_code」才会出现那条 tool 消息；若删除 generate_code，这个锚点就没了。

## 三、删除 generate_code 的替代方案

- **不再**用「generate_code 之后的 assistant 消息」做兜底。
- **新兜底**：当没有 `$source_code` 时，从**本条 assistant 消息正文**（即含本次 tool_calls 的那条消息的 content）里解析 markdown 代码块。
- 这样模型可以：**同一条消息**里先写 ```go ... ```，再调用 `write_package_code(file_name)`（不传 `$source_code`），系统从本条消息里挖代码块即可，无需先调 generate_code。

## 四、结论

**可以删除 generate_code**。删除后：

- 推荐流程不变：`<var>` + `$source_code`。
- 兜底改为：从**本条消息**里解析代码块；若解析不到则报错，提示用 `<var>` 或在本条消息中写 markdown 代码块。

语义更简单：写代码只有两种方式——要么变量引用，要么本条消息里的代码块，不再依赖「先声明再下一条消息出代码」的隐式约定。
