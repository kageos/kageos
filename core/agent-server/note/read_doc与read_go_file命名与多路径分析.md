# read_doc / read_go_file 是否改为复数名（read_docs / read_go_files）分析

## 一、现状

| 工具 | 当前能力 | 多值支持 |
|------|----------|----------|
| **read_go_file** | 读工作区 .go 文件 | ✅ 已支持：`file_name` 可逗号分隔多文件（如 `a.go,b.go`），名称仍为 `read_go_file` |
| **read_doc** | 读文档（内置或工作区），`directory` 唯一定位 | ❌ 仅单路径，未支持多路径 |

## 二、要不要改名为复数？

### 结论：**不建议改名**（保持 `read_doc`、`read_go_file`）

理由简要如下。

1. **和“单工具 + 多值参数”的惯例一致**  
   常见做法是工具名用单数，通过参数表达“可多值”（如 `get_item(ids="1,2,3")`）。  
   - `read_go_file(directory, file_name="a.go,b.go")` 已经符合这一习惯，无需为多文件单独改名为 `read_go_files`。  
   - 若给 `read_doc` 增加多路径，用 `directory="/path1,/path2"` 即可，同样不必改名为 `read_docs`。

2. **改名成本高、收益小**  
   若改为 `read_docs` / `read_go_files`：
   - 需改：`ToolRegistry` 工具名、所有 prompt/文档里的工具名、`workspace_mode` 默认 `tool_names`、错误提示里“请用 read_doc”等文案。
   - 对已学过 `read_doc` / `read_go_file` 的模型来说，会多出一层“旧名→新名”的迁移成本。
   - 功能上只是“多传几个路径/文件”，用参数扩展就能表达，不需要靠改名表达。

3. **命名风格统一**  
   其他工具也是单数：`read_dir`、`write_doc`、`write_go_file`。保持 `read_doc`、`read_go_file` 更统一，且不会让人误以为“复数名才能传多个”。

因此：**read_doc 不必换成 read_docs，read_go_file 不必换成 read_go_files**；需要“多个”时，用参数支持即可。

## 三、read_doc 是否要支持多个路径？

### 建议：**要支持，用“逗号分隔多路径”即可，工具名仍为 read_doc**

- **用法示例**：`read_doc(directory: "/builtin/doc/sdk/xxx,/builtin/doc/case_catalog/yyy")`，一次返回多份文档内容（每份带标题，顺序与传入一致）。
- **实现要点**：
  - 对 `directory` 按逗号拆分、trim，得到多个路径；
  - 对每个路径分别调现有逻辑（builtin 用 `GetBuiltinDocContent`，工作区用 `GetDoc`）；
  - 按路径去重（同一 path 只算一次），再拼成一份结果（例如用 `## 文档名\n\n内容` 分段）。
- **与 read_go_file 一致**：都是“单工具名 + 单参数多值（逗号分隔）”，不引入新工具名或新参数名。

## 四、汇总建议

| 项目 | 建议 |
|------|------|
| read_go_file 是否改名为 read_go_files | **不改**。保持 `read_go_file`，多文件已用 `file_name="a.go,b.go"` 支持。 |
| read_doc 是否改名为 read_docs | **不改**。保持 `read_doc`。 |
| read_doc 是否支持多路径 | **支持**。扩展 `directory` 为逗号分隔多路径（如 `"/path1,/path2"`），实现上循环+去重+拼接即可。 |

按上述做法，命名一致、改造成本小，且能满足“一次读多个文档/多个文件”的需求。
