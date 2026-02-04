# Table 更新工具（run_table_update）设计分析

## 一、现状

### 1. 后端 Table 更新接口（已有）

- **路由**：`PUT /api/v1/table/update/*full-code-path`
- **请求体**（OnTableUpdateRowReq 约定）：
  - `id`（必填）：要更新的行 ID
  - `updates`（必填）：本次要改的字段，如 `{ "status": "已处理", "title": "新标题" }`
  - `old_values`（可选）：更新前的整行数据，用于审计/乐观锁等；**不传时回调里 GetOldValues() 为空 map，后端不强制**
- **问题**：若调用方（大模型/执行工具）只有「id + 要改的字段」，要构造完整 body 就必须**先调列表接口按 id 查当前行**，再拼出 `old_values`，步骤多、易错，且大模型很难一次做对。

### 2. 执行侧

- 目前没有 `run_table_update` 工具，也没有 `pkg/apicall.TableUpdate`。
- 文档里写的是「编辑/删除：若后续提供 run_table_update…」。

---

## 二、设计目标

1. **调用方只传 id + updates**，不要求传 old_values。
2. **old_values 由工具内部**通过 table/search（eq=id）拉取当前行后自动填入，再调 table/update。
3. **批量更新**：和 run_table_create 一致，body 为 JSON 数组，每项一条更新；工具内循环「查 → 构造 body → 更新」，汇总返回。

---

## 三、run_table_update 设计（批量 + 内部自动拉 old_values）

### 3.1 入参

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| full_code_path | string | 是 | 表格函数完整路径（与 run_table_search / run_table_create 一致） |
| body | string | 是 | JSON 字符串。**数组**，每项为 `{ "id": <行ID>, "updates": { "field": "value", ... } }`；不传 old_values，由工具内部补全 |

**单条**也可写成数组只有一项：`[{ "id": 1, "updates": { "status": "已处理" } }]`。

### 3.2 工具内部流程（对每条）

1. 从 body 中解析出数组，遍历每一项 `{ id, updates }`。
2. **拉取当前行**：调用 table/search，`url_query = "eq=id:<id>&page=1&page_size=1"`，取返回的 `items[0]` 作为当前行（old_values）。
3. **若 items 为空**：本条视为「记录不存在」，记入 errors，不调 table/update。
4. **若有当前行**：构造完整 body  
   `{ "id": id, "updates": updates, "old_values": <当前行> }`  
   调用 `PUT table/update/{full_code_path}`。
5. 汇总：updated_count、data_list（或每条更新后的结果）、errors（失败条目的 index + 原因）。

### 3.3 返回

与 run_table_create 风格一致，例如：

```json
{
  "updated_count": 2,
  "failed_count": 0,
  "data_list": [ /* 每条更新后接口返回的 data/result */ ],
  "errors": []
}
```

若某条查不到或更新失败，则 failed_count 增加，errors 中追加 `{ "index": i, "error": "原因" }`。

### 3.4 要点小结

- **调用方**：只提供 `full_code_path` + body（数组，每项 `id` + `updates`），**不需要**也不传 old_values。
- **工具内部**：用列表接口 `eq=id` 查当前行 → 作为 old_values → 再调更新接口，保证审计/回调能拿到旧值。
- **批量**：body 为数组，一条或多条统一处理，和 run_table_create 一致，便于大模型「批量改状态」等场景。

---

## 四、实现清单

1. **pkg/apicall**  
   - 新增 `TableUpdate(ctx, fullCodePath string, body interface{}) (map[string]interface{}, error)`  
   - 内部：`PUT /workspace/api/v1/table/update{fullCodePath}`，body 原样传（map 或可序列化结构）。

2. **core/agent-server/service/tool_registry.go**  
   - 注册工具 `run_table_update`（description 写明：批量更新表格记录，body 为 JSON 数组，每项含 id、updates；工具内部按 id 查当前行作为 old_values 再调更新接口）。  
   - `CallTool` 增加 `case "run_table_update": return r.callRunTableUpdate(...)`。  
   - `callRunTableUpdate`：  
     - 解析 body 为 `[]interface{}`，遍历每项取 `id`、`updates`；  
     - 对每条：`TableSearch(ctx, fullCodePath, url.Values{"eq": []string{"id:"+idStr}, "page": "1", "page_size": "1"})`，从 result 中取 `items[0]` 为 old_values；  
     - 若无 items 或 len(items)==0，本条失败；否则构造 `{ id, updates, old_values }` 调 `apicall.TableUpdate`；  
     - 汇总 updated_count、data_list、errors 并返回。

3. **模式配置**  
   - 执行/开发等模式的 tool_names 中增加 `run_table_update`。

4. **文档**  
   - execute 文档（01-execute.md）：已增加「更新表格记录 → run_table_update」，说明 body 格式（数组、id+updates）；old_values 由 app-server 自动填充。  
   - 01-execute.md：「选对工具」「何时用什么工具」中增加「更新表格记录 → run_table_update」。

5. **table/search 返回结构**  
   - 以实际接口为准，取列表字段（一般为 `items`）的第一条作为当前行；若结构不同，在 callRunTableUpdate 里做一次适配即可。

---

## 五、能力下沉：app-server table/update 内自动填充 old_values（已实现）

- **实现位置**：`core/app-server/api/v1/standard_api.go` 的 `TableUpdate`。
- **逻辑**：解析 body 后，若存在 `id` 且 `old_values` 缺失或为空，则内部对该 full_code_path 发起一次 **table/search**（`eq=id:<id>&page=1&page_size=1`），取 `items[0]` 作为当前行，填入 `bodyData["old_values"]`，再写回 `c.Request.Body`，然后继续 `buildCallbackAppReq` 和 `RequestApp`。
- **效果**：上层（前端、agent 的 run_table_update）只需传 **id + updates**，不传 old_values 也会由平台自动补全，属于**能力下沉**，调用更方便。

## 六、结论

- 当前 table 更新接口**需要** body 里带 id、updates，old_values **可选**；若未传则由 **app-server 自动查表填充**（已实现）。
- **run_table_update** 工具只需传 id + updates（单条或批量数组），无需在工具内再查表；批量时循环调用 table/update 即可，每条请求都会在 app-server 侧自动带上 old_values。
