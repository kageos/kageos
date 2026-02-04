# read_doc 与目录读取分析

## 结论（简要）

- **之前**：`read_doc` 传的是「路径」，但**不是**「传目录就自动读目录下所有文档」。实现是：路径必须在 `content/doc/文档目录.json` 里有一条 **full_code_path 完全匹配**；命中后只读**一个**文件（依次尝试 `prd.md`、`路径名.md`、`README.md`）。workspace 下 create-project、modify-project、execute、explain-project **未**在 文档目录.json 里登记，且实际文件是 `01-xxx.md`，所以传目录会「未找到」。
- **现在**：已做两处改动：
  1. **读目录 = 返回该目录下所有 .md**：在 `embed.go` 的 `getBuiltinDocContentByEntry` 里，单文件（prd.md / xxx.md / README.md）都读不到时，会尝试把该路径当作**目录**，列出 `content/builtin/<rel>/` 下所有 `.md`，按文件名排序后拼接返回。这样 `read_doc("/builtin/doc/workspace/modify-project")` 会返回该目录下的 `01-modify-project.md` 等全部 .md。
  2. **文档目录.json**：已登记 workspace 四个子目录（create-project、modify-project、execute、explain-project），便于在「可用文档」等系统消息里展示；且对 `/builtin/doc/workspace/` 下路径做了**回退**：即使未在 文档目录.json 里登记，也会按目录读并返回该目录下所有 .md。

## 两套路径别搞混

| 用途         | 路径位置                     | 谁在用                 | read_doc 能否读 |
|--------------|------------------------------|------------------------|-----------------|
| **模式配置** | `content/mode/dev/`          | mode_provider（system_prompt、first_assistant 等） | **不能**。mode 由 init 时从 embed 直接读 `content/mode/<code>/` 下 config.json 和 md，不经过 read_doc。 |
| **内置文档** | `content/builtin/doc/`       | read_doc(directory: "/builtin/doc/...") | **能**。read_doc 只读 `content/builtin/` 下且在 文档目录.json 登记（或 /builtin/doc/workspace/ 回退）的路径。 |

所以：**文档放在「正确位置」**  
- 若是给 **dev 模式**用的（system_prompt、first_assistant 等）→ 放在 `content/mode/dev/`，由模式加载，**不需要** read_doc。  
- 若是给 **read_doc** 用的（PRD、SOP、任务规范等）→ 放在 `content/builtin/doc/` 下对应子目录（如 workspace/modify-project），并保证该路径在 文档目录.json 登记或属于 `/builtin/doc/workspace/`，传目录时会返回该目录下**所有 .md**（如 01-xxx.md），按文件名顺序展示。

## 当前 read_doc 行为小结

- 传 **directory**（可逗号分隔多路径）。
- 路径以 `/builtin/` 开头时：  
  - 先在 文档目录.json 里找 **full_code_path 完全匹配**；  
  - 若未找到且路径以 `/builtin/doc/workspace/` 开头，则**按目录读**该路径下所有 .md 并拼接。  
- 命中后取内容：  
  - 先按**单文件**尝试：`<rel>/prd.md`、`<rel>.md`、`<rel>/README.md`；  
  - 若都读不到，再按**目录**：列出 `content/builtin/<rel>/` 下所有 .md，按文件名排序后拼接返回。
- 因此：**直接传目录（如 `/builtin/doc/workspace/modify-project`）会返回该目录下所有文档**（如 01-modify-project.md），不再只是「读一个文件」。
