# 工作台：实时读/写/更新 + Search-Replace 式编辑分析

目标：**读 Go 文件实时从盘读、更新实时生效、新增也实时**；**编辑不能是整文件覆盖**（太慢、要重写很久），需要 **search-replace 那种** 只改一段、实时更新。下面按「已有 / 缺口 / 要新增的接口」整理。

---

## 一、当前能力与缺口（结论表）

| 能力 | 是否实时 | 现状 | 缺口 |
|------|----------|------|------|
| **读 Go 文件** | ✅ 实时 | GetWorkspaceContext(file_source=runtime) → app-runtime ReadDirectoryFiles 从磁盘读 | 无 |
| **新增文件** | ✅ 实时 | write_go_file → AddFunctions → app-runtime CreateFunctions 直接写盘 | 无 |
| **更新/编辑文件** | ❌ 非预期 | 目前只有「整文件覆盖」：read 全量 → 改 → write_go_file 全量，耗时长、易出错 | 缺 **search-replace 式** 编辑，要 **实时** 只改一段 |
| **删除文件** | ❌ 未打通 | 无 delete_file；DeleteFunction 只删 DB 不删盘 | 缺「删节点 + 删磁盘」联动 + 工作台工具 |

所以：  
- **读、新增**：已经是你说的「实时从盘读 / 实时新增」。  
- **更新**：缺的是「**不整文件覆盖、类似 search replace、且实时**」的接口与工具。  
- **删除**：缺的是「删文件 + 删节点」的完整链路和工具。

---

## 二、读 Go 文件（已实时）

- **接口**：GetWorkspaceContext(full_code_path, **file_source=runtime**)
- **链路**：agent-server read_go_file → app-server GetWorkspaceContext(file_source=runtime) → app-runtime **ReadDirectoryFiles**（os.ReadFile 从盘读）
- **结论**：读到的就是当前磁盘内容，无需再改。

---

## 三、新增文件（已实时）

- **接口**：write_go_file → ServiceTreeAddFunctions → app-server AddFunctions → app-runtime **CreateFunctions**（os.WriteFile 写盘）
- **结论**：新增就是直接写盘，实时；无需新增接口。

---

## 四、更新/编辑：不要整文件覆盖，要 Search-Replace 式 + 实时

- **问题**：  
  - 整文件覆盖 = 先 read 全量 → 模型重写整份 → write_go_file 全量，耗 token、耗时间、易漏改。  
  - 你要的是：**类似 search replace**——只传「找哪段、改成啥」，服务端在**当前磁盘文件**上做替换并写回，**实时更新**。

- **需要的新能力（按层）**：

### 1）app-runtime：新增「按路径 + search-replace 改文件」接口（实时写盘）

- **建议名称**：`ReplaceInFile` / `PatchFileContent`（二选一即可）。
- **入参建议**：  
  - `user`, `app`  
  - `directory_path`（full_code_path，如 `/user/app/pkg1`）  
  - `file_name`（如 `handler` 或 `handler.go`）  
  - `search_string`：要被替换的原文（或首行/片段，用于定位）  
  - `replace_string`：替换后的内容  
  - `replace_all`：bool，是否替换全部出现（默认 true 更符合「改完所有」的直觉；false 则只替第一次）
- **逻辑**：  
  - 根据 user/app/directory_path/file_name 解析出**磁盘路径**（与 ReadDirectoryFiles / BatchWriteFiles 一致）。  
  - `os.ReadFile` 读**当前磁盘内容**（实时）。  
  - 在内存里做 `strings.Replace(content, search_string, replace_string, n)`（n 由 replace_all 决定）。  
  - `os.WriteFile` 写回磁盘 → **实时更新**。  
  - 可选：是否在本调用内触发编译/版本（可先只写盘，编译由现有 build_workspace 或后续统一触发）。
- **返回**：success、message；可选：替换次数、新文件行数等，便于工作台提示。

**可选扩展（同一接口或后续加）**：  
- 支持**多组** search-replace：`replacements: [{search, replace}, ...]`，一次调用改多处，减少工具调用次数。

### 2）app-server：新增「工作台用」的 search-replace 接口

- **作用**：对工作台暴露「按 full_code_path + file_name 做 search-replace」。
- **建议**：  
  - 例如 `POST /workspace/api/v1/files/replace`（或放在 service_tree 下，如 `/workspace/api/v1/service_tree/replace_file`）。  
  - 入参：`full_code_path`、`file_name`、`search_string`、`replace_string`、`replace_all`（可选）。  
  - 内部：根据 full_code_path 解析出 user/app、取目录对应 App 的 HostID，调 app-runtime 的 **ReplaceInFile**；返回成功/失败与简单信息。
- **结论**：需要新增 1 个 HTTP 接口，桥接到 runtime 的 ReplaceInFile。

### 3）agent-server：新增「search-replace 编辑」工具

- **建议名称**：`search_replace_file` / `replace_in_file`（与接口语义一致即可）。
- **参数建议**：  
  - `directory`（可选，默认当前工作目录）  
  - `file_name`（必填）  
  - `search_string`（必填）  
  - `replace_string`（必填）  
  - `replace_all`（可选，默认 true）
- **逻辑**：  
  - 解析出 full_code_path，调 app-server 的 replace 接口 → 即**实时更新磁盘**，无需模型输出整文件。
- **结论**：需要新增 1 个工具，并在提示词里说明「编辑用 search_replace_file，不要整文件重写」。

这样：**更新 = search-replace，实时写盘，不需要「重写整份、很久」**。

---

## 五、删除文件（仍需补齐）

- **需求**：删掉目录下某个 Go 文件，且**磁盘和 DB 一起删**（不能只删节点、盘上还留着）。
- **需要的新能力**：  
  1. **app-runtime**：新增「按目录 + 文件名删除磁盘文件」接口（如 `DeleteFile(user, app, directory_path, file_name)`），删完后可选触发编译。  
  2. **app-server**：DeleteFunction 时先调 app-runtime 删该文件，再删 DB；或单独提供「按 full_code_path + file_name 删文件」的接口给工作台。  
  3. **agent-server**：新增 `delete_file(directory, file_name)` 工具。

（这部分与之前「工作台目录文件操作工具分析」一致，不重复展开。）

---

## 六、需要新增的接口汇总（是否就是这几个）

按你的预期「读实时、更新实时、新增实时，编辑要 search-replace」整理，**需要新增的**就是下面这几类（读/新增已有，不列）：

| 层级 | 新增接口/能力 | 说明 |
|------|----------------|------|
| **app-runtime** | **ReplaceInFile**（或 PatchFileContent） | 按路径 + search_string + replace_string 改文件内容；读盘 → 替换 → 写盘，实时。 |
| **app-runtime** | **DeleteFile** | 按目录 + file_name 删磁盘文件（可与 DeleteFunction 联动）。 |
| **app-server** | **ReplaceFileContent**（或 SearchReplaceFile） | 接收 full_code_path + file_name + search/replace，调 runtime ReplaceInFile。 |
| **app-server** | **DeleteFunction 时调 runtime DeleteFile** | 或单独提供「按路径+文件名删」的 API 给工作台。 |
| **agent-server** | **search_replace_file** 工具 | 参数：directory, file_name, search_string, replace_string, replace_all；内部调 app-server replace 接口。 |
| **agent-server** | **delete_file** 工具 | 参数：directory, file_name；内部调 app-server 删节点（并触发 runtime 删文件）。 |

所以：  
- **编辑**：不再靠「整文件覆盖」，而是 **search-replace 接口 + search_replace_file 工具**，实现「只改一段、实时更新」。  
- **读**：已经实时从盘读（file_source=runtime）。  
- **新增**：已经实时写盘（write_go_file）。  
- **删除**：需要补齐 runtime DeleteFile + app-server 联动 + delete_file 工具。

如果你认可「就是这几个接口」，下一步可以按 **ReplaceInFile → app-server replace 接口 → search_replace_file 工具** 的顺序实现，然后再做 **DeleteFile + delete_file**。
