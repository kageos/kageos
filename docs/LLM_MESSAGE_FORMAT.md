# LLM 消息格式说明

本文档说明在代码生成时，传递给大模型的数据格式。

## 当前消息结构

### 1. System Message（系统消息）

**格式**：
```
{系统提示词模板}

{知识库内容}

当前 Package 上下文：{package_name}
```

**详细说明**：

1. **系统提示词模板**（`agent.SystemPromptTemplate`）
   - 来源：智能体配置中的"系统提示词"字段
   - 如果为空，使用默认值：`"你是一个专业的代码生成助手。"`
   - 示例：`plugins/excel2admin/docs/系统提示词.md` 的内容

2. **知识库内容**
   - 来源：智能体绑定的知识库中所有 `status = "completed"` 的文档
   - 格式：每个文档以 `## {文档标题}\n{文档内容}\n` 的形式拼接
   - 示例：
     ```
     ## SDK使用文档
     # Agent-App SDK 使用文档
     ...
     
     ## 系统提示词
     # Excel2Admin 智能体系统提示词
     ...
     ```

3. **Package 上下文**（新增）
   - 来源：根据 `req.TreeID` 查询 `service_tree` 表，获取 `Code` 字段
   - 格式：`当前 Package 上下文：{package_name}`
   - 示例：`当前 Package 上下文：crm`
   - **重要**：如果查询失败或 `TreeID` 为 0，则不添加此信息

**完整示例**：
```
你是一个专业的代码生成助手，专门负责将 Excel 数据转换为符合 Agent-App SDK 规范的 Go 代码。

## SDK使用文档
# Agent-App SDK 使用文档
...
## 系统提示词
# Excel2Admin 智能体系统提示词
...

当前 Package 上下文：crm
```

### 2. 历史消息（History Messages）

**格式**：
```json
[
  {
    "role": "user",
    "content": "用户消息1"
  },
  {
    "role": "assistant",
    "content": "AI回复1"
  },
  {
    "role": "user",
    "content": "用户消息2"
  }
]
```

**说明**：
- 从数据库 `agent_chat_messages` 表中加载，按 `created_at` 排序
- 只包含 `session_id` 匹配的历史消息
- 最后一条用户消息会被跳过（因为要用插件处理后的内容替换）

### 3. 当前用户消息（Current User Message）

**格式（非 plugin 类型）**：
```json
{
  "role": "user",
  "content": "{用户原始消息}"
}
```

**格式（plugin 类型）**：
```json
{
  "role": "user",
  "content": "{用户原始消息}\n\n{插件处理后的数据}"
}
```

**详细说明**：

1. **用户原始消息**（`req.Message.Content`）
   - 示例：`"帮我基于这个excel生成一个工单管理系统"`

2. **插件处理后的数据**（仅 plugin 类型）
   - 来源：调用 `excel2admin` 插件处理 Excel 文件后返回的 CSV 格式数据
   - 示例：
     ```
     工单标题,问题描述,优先级,工单状态,备注,附件
     工单1,描述1,低,待处理,备注1,附件1
     工单2,描述2,中,处理中,备注2,附件2
     工单标题允许20个字符内,最多500字符,分为低中高三个状态,分为待处理 处理中 已完成 已关闭 四种状态，默认待处理,备注可不填写,附件无限制上传文件类型，最大限制10MB 最多上传5个
     ```

**完整示例（plugin 类型）**：
```json
{
  "role": "user",
  "content": "帮我基于这个excel生成一个工单管理系统\n\n工单标题,问题描述,优先级,工单状态,备注,附件\n工单1,描述1,低,待处理,备注1,附件1\n..."
}
```

## 完整消息列表示例

```json
[
  {
    "role": "system",
    "content": "你是一个专业的代码生成助手...\n\n## SDK使用文档\n..."
  },
  {
    "role": "user",
    "content": "帮我基于这个excel生成一个工单管理系统\n\n工单标题,问题描述,优先级,工单状态,备注,附件\n工单1,描述1,低,待处理,备注1,附件1\n..."
  }
]
```

## ✅ 已实现的功能

### Package 上下文信息

**状态**：✅ 已实现

**实现方式**：
1. 在 `FunctionGenChat` 方法中，根据 `req.TreeID` 查询 `service_tree` 表
2. 获取 `ServiceTree.Code` 字段（这就是 package 名称）
3. 在 system message 中添加 package 上下文信息

**代码位置**：
- `core/agent-server/service/function_gen_service.go` 的 `FunctionGenChat` 方法（第 248-261 行）
- 在构建 system message 时添加（第 280-284 行）

**实现细节**：
```go
// 获取 Package 信息（根据 TreeID 查询 ServiceTree）
var packageName string
if req.TreeID > 0 {
    var serviceTree appModel.ServiceTree
    err := s.db.Where("id = ?", req.TreeID).First(&serviceTree).Error
    if err == nil && serviceTree.Code != "" {
        packageName = serviceTree.Code
        // 添加到 system message
        systemPromptContent.WriteString(fmt.Sprintf("\n\n当前 Package 上下文：%s", packageName))
    }
}
```

## 代码位置

- **消息构建逻辑**：`core/agent-server/service/function_gen_service.go` 的 `FunctionGenChat` 方法
- **System Message 构建**：第 247-270 行
- **用户消息构建**：第 272-332 行
- **LLM 调用**：第 380-391 行

