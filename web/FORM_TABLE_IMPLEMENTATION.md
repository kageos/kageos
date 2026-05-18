# Form 和 Table 模版渲染实现文档

> 状态：草案
> 更新时间：2026-05-17
> 负责人窗口：事项 8 / codex/document-archive-cleanup

## 📋 已实现功能

### ✅ Form 函数渲染 (FormRenderer)

#### 支持的 Widget 类型
- ✅ **input** - 文本输入框（支持 prepend/append、password 模式）
- ✅ **number** - 数字输入框（支持 min/max/step/precision）
- ✅ **text_area** - 多行文本框（支持 rows 配置）
- ✅ **select** - 下拉选择（支持单选/多选）
- ✅ **datetime** - 时间选择器（支持自定义格式）
- ✅ **switch** - 开关
- ✅ **checkbox** - 多选框组
- ✅ **radio** - 单选框组

#### 表单验证
- ✅ `required` - 必填验证
- ✅ `min=N` - 最小长度验证
- ✅ `max=N` - 最大长度验证
- ✅ `oneof=选项1,选项2` - 枚举值验证

#### 功能特性
- ✅ 根据 `request` 字段自动渲染表单
- ✅ 支持字段渲染默认值（`widget.config.render_default`）
- ✅ 支持字段描述（`desc` 字段）
- ✅ 提交时调用 `/workspace/api/v1/form/submit{full_code_path}` 接口
- ✅ 支持 GET/POST/PUT 等不同的 HTTP 方法
- ✅ 自动显示执行结果（基于 `response` 字段）
- ✅ 结果支持 datetime 字符串时间展示
- ✅ 支持表单重置功能

### ✅ Table 函数渲染 (TableRenderer + FormDialog)

#### 列表功能
- ✅ 根据 `schema.table.fields` 字段自动渲染表格列
- ✅ 根据 `hide.scenes` 控制列显示
  - 不配置 `hide` - 显示
  - `list` - 显示
  - `create` - 不显示（只在新增表单展示）
  - `update` - 不显示（只在编辑表单展示）
- ✅ 时间字段自动格式化显示
- ✅ 支持分页（page、page_size）
- ✅ 支持排序（点击列头排序）

#### 搜索功能
- ✅ 根据 Request 筛选字段自动生成搜索表单
- ✅ `eq` - 精确匹配
- ✅ `like` - 模糊查询
- ✅ `in` - 包含查询（下拉选择）
- ✅ `gte/lte` - 时间范围查询

#### CRUD 操作
- ✅ **新增功能**
  - 根据 `schema.callbacks` 判断是否显示"新增"按钮
  - 点击后弹出表单对话框
  - 根据 `hide.scenes` 控制字段显示（`hide.scenes` 不包含 `create` 或未配置 `hide` 的字段会进入新增表单）
  - 调用 `/workspace/api/v1/table/create{full_code_path}`

- ✅ **编辑功能**
  - 根据 `schema.callbacks` 判断是否显示"编辑"按钮
  - 点击后弹出表单对话框，预填当前数据
  - 根据 `hide.scenes` 控制字段显示（`hide.scenes` 不包含 `update` 或未配置 `hide` 的字段会进入编辑表单）
  - 调用 `/workspace/api/v1/table/update{full_code_path}`

- ✅ **删除功能**
  - 根据 `schema.callbacks` 判断是否显示"删除"按钮
  - 点击后二次确认
  - 调用 `/workspace/api/v1/table/delete{full_code_path}`

## 🎯 使用方式

### Form 函数
当函数详情的 `template_type` 为 `"form"` 时：
1. 根据 `request` 字段渲染表单
2. 用户填写表单后点击"提交"
3. 调用 `/workspace/api/v1/form/submit{full_code_path}` 提交数据
4. 如果有 `response` 字段，自动显示执行结果

### Table 函数
当函数详情的 `template_type` 为 `"table"` 时：
1. 自动调用 `/workspace/api/v1/table/search{full_code_path}` 获取列表数据
2. 根据 `schema.table.fields` 字段渲染表格
3. 根据 Request 筛选字段生成搜索表单
4. 根据 `schema.callbacks` 字段决定显示哪些操作按钮：
   - `OnTableAddRow` → 显示"新增"按钮
   - `OnTableUpdateRow` → 显示"编辑"按钮
   - `OnTableDeleteRows` → 显示"删除"按钮

## 🔐 展示场景 (hide.scenes)

| 值 | 列表显示 | 新增时 | 编辑时 | 说明 |
|---|---|---|---|---|
| 不配置 `hide` | ✅ | ✅ | ✅ | 前端三个场景都展示 |
| `create,update` | ✅ | ❌ | ❌ | 前端仅在列表展示，不进入新增/编辑表单 |
| `list,update` | ❌ | ✅ | ❌ | 前端仅在新增表单展示 |
| `list,create` | ❌ | ❌ | ✅ | 前端仅在编辑表单展示 |

`table` / `form` 是容器组件，不作为 table 列表列渲染；即便误配 `hide.scenes=list`，前后端都会静默忽略，避免影响应用注册。

## 📝 API 接口

### Form / Table 标准接口
```typescript
// Form 提交
POST /workspace/api/v1/form/submit{full_code_path}

// Table 查询
GET /workspace/api/v1/table/search{full_code_path}
```

### Table 写操作
**重要说明**：
- 前端只调用标准 Table 接口；后端内部转成 `_callback` 请求。
- 写操作是否可用只看 `schema.callbacks`。

```typescript
// 新增记录
POST /workspace/api/v1/table/create{full_code_path}
Body: { field1: value1, field2: value2, ... }

// 更新记录
PUT /workspace/api/v1/table/update{full_code_path}
Body: { id: 记录ID, updates: { field1: value1, ... } }

// 删除记录
DELETE /workspace/api/v1/table/delete{full_code_path}
Body: { ids: [id1, id2, ...] }
```

**示例**：
```
POST /workspace/api/v1/table/create/beiluo/testapi21/crm/crm_ticket.table
Content-Type: application/json

Body:
{
  "title": "测试工单",
  "status": "待处理",
  "priority": "高"
}
```

**设计原因**：
- 前端对接稳定的标准接口，不感知底层 `_callback` 包装。
- 服务端统一校验 `schema.callbacks`，避免只读表被误写。
- Table 更新缺少 `old_values` 时，服务端会按 id 查询当前行并自动补齐。

## 🧪 测试步骤

### 测试 Form 函数
1. 在服务目录中点击一个 `template_type` 为 `form` 的函数
2. 观察表单是否正确渲染（根据 `request` 字段）
3. 填写表单并提交
4. 观察是否正确显示执行结果（根据 `response` 字段）

**示例**：斐波那契数列计算器
- 函数路径：`/luobei/test10/tools/tools_fibonacci`
- 输入：起始位置、结束位置、分隔符
- 输出：斐波那契数列、数列和

### 测试 Table 函数
1. 在服务目录中点击一个 `template_type` 为 `table` 的函数
2. 观察表格是否正确加载数据
3. 测试搜索功能（精确查询、模糊查询、时间范围等）
4. 测试排序功能（点击列头）
5. 测试分页功能
6. 如果有 `OnTableAddRow` 回调，测试新增功能
7. 如果有 `OnTableUpdateRow` 回调，测试编辑功能
8. 如果有 `OnTableDeleteRows` 回调，测试删除功能

**示例**：工单管理
- 函数路径：`/beiluo/testapi21/crm/crm_ticket`
- 包含完整的 CRUD 功能

## 📦 新增文件

1. `/web/src/components/FormRenderer.vue` - Form 函数渲染器
2. `/web/src/components/FormDialog.vue` - 表单对话框（用于 Table 的新增/编辑）
3. 更新 `/web/src/components/TableRenderer.vue` - 完善 CRUD 功能
4. 更新 `/web/src/api/function.ts` - 添加回调接口
5. 更新 `/web/src/views/Workspace/index.vue` - 集成 FormRenderer

## 🎨 组件结构

```
Workspace (工作区)
├── ServiceTreePanel (左侧服务目录树)
└── Function Renderer (中间渲染区)
    ├── TableRenderer (Table 类型)
    │   ├── 工具栏（新增按钮）
    │   ├── 搜索栏
    │   ├── 数据表格（编辑、删除按钮）
    │   ├── 分页器
    │   └── FormDialog (新增/编辑对话框)
    │
    └── FormRenderer (Form 类型)
        ├── 表单区域
        └── 结果展示区域
```

## 🚀 下一步优化建议

1. **批量操作**：支持批量删除多条记录
2. **导出功能**：支持导出 Excel
3. **高级搜索**：支持更复杂的查询条件组合
4. **字段校验**：支持更多的 validation 规则（如邮箱、手机号等）
5. **自定义渲染**：支持自定义列渲染（如标签、徽章等）
6. **图片上传**：支持 `file_upload` widget 类型
7. **富文本编辑器**：支持富文本内容编辑

## 💡 注意事项

1. **时间字段格式**：时间字段统一使用 `datetime` 字符串时间
2. **分页数据结构**：后端应返回 `{ items: [], paginated: { current_page, page_size, total_count, total_pages } }` 结构
3. **回调接口**：前端调用标准 Table 接口，后端内部按 schema callbacks 转发回调
4. **ID 字段**：Table 的编辑和删除功能依赖于每行数据必须有 `id` 字段
5. **展示场景判断**：未配置 `hide` 表示列表/新增/编辑均展示；配置 `hide.scenes` 后只在指定场景展示

---

**实现完成时间**：2025-10-30
**实现功能**：Form 函数渲染 + Table 完整 CRUD 功能
