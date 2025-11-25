# FormRenderer 命名规范化重构

## 🎯 重构原因

用户反馈：`SimpleFormRenderer` 这个名字"低俗"，不符合完全重写的定位。

**核心理念**：
- ✅ 我们是**完全重写**，不是"简化版"
- ✅ 新的渲染引擎应该使用正式名称 `FormRenderer`
- ✅ 旧系统应该被标记为 `Legacy`（遗留）

---

## 📝 重构内容

### 1. 文件重命名

| 旧文件路径 | 新文件路径 | 说明 |
|-----------|-----------|------|
| `src/components/FormRenderer.vue` | `src/components/LegacyFormRenderer.vue` | 旧渲染器，标记为遗留代码 |
| `src/core/renderers/SimpleFormRenderer.vue` | `src/core/renderers/FormRenderer.vue` | 新渲染器，使用正式名称 |

### 2. 更新的文件

#### 代码文件
- ✅ `src/views/Workspace/index.vue`
  - 更新 import：`FormRenderer from '@/core/renderers/FormRenderer.vue'`
  - 更新模板：直接使用 `<FormRenderer :function-detail="functionDetail" />`
  - 移除旧的 props（`fields`, `response-fields`, `method`, `router`, `mode`）

- ✅ `src/views/Test/FormRendererTest.vue`
  - 更新 import 和组件引用

- ✅ `src/core/renderers/FormRenderer.vue`
  - 更新所有控制台日志：`[SimpleFormRenderer]` → `[FormRenderer]`

#### 文档文件
- ✅ `src/core/README.md`
  - 更新组件介绍和使用示例

- ✅ `docs/新旧渲染系统集成方案.md`
  - 更新系统对比说明
  - 更新代码示例

---

## 🚀 使用方式变化

### 旧方式（已废弃）

```vue
<FormRenderer
  :fields="functionDetail.request || []"
  :response-fields="functionDetail.response || []"
  :method="functionDetail.method"
  :router="functionDetail.router"
  mode="form"
/>
```

### 新方式（推荐）

```vue
<FormRenderer
  :function-detail="functionDetail"
/>
```

**优点**：
- ✅ 简洁：只需传递一个完整的 `functionDetail` 对象
- ✅ 完整：自动解析 `request`, `response`, `method`, `router` 等所有字段
- ✅ 扩展性：未来添加新字段无需修改 props

---

## 📊 当前状态

### 新系统（`FormRenderer.vue`）

**位置**：`src/core/renderers/FormRenderer.vue`

**功能**：
- ✅ 支持嵌套结构（`children`）
- ✅ 支持 Widget 系统（`InputWidget`, `SelectWidget`, `ListWidget` 等）
- ✅ 支持回调系统（`OnSelectFuzzy` 等）
- ✅ 支持快照/分享功能
- ✅ OOP 架构，易于扩展

**使用位置**：
- `src/views/Workspace/index.vue` - form 类型函数
- `src/views/Test/FormRendererTest.vue` - 测试页面

### 旧系统（`LegacyFormRenderer.vue`）

**位置**：`src/components/LegacyFormRenderer.vue`

**状态**：
- ⚠️ 已标记为遗留代码
- ⚠️ 建议逐步迁移到新系统
- ⚠️ 未来将被完全移除

**使用位置**：
- 无（已被新系统替代）

---

## 🎯 下一步计划

### 短期（今天）
1. ✅ 完成命名规范化
2. ⬜ 创建 `NumberWidget`
3. ⬜ 测试收银台功能

### 中期（本周）
1. ⬜ 补充所有缺失的 Widget
2. ⬜ 完善回调系统
3. ⬜ 实现聚合计算

### 长期（下周）
1. ⬜ 完全移除 `LegacyFormRenderer`
2. ⬜ 完善文档和测试
3. ⬜ 性能优化

---

## ✅ 重构验证

### 验证清单

- [x] 文件重命名成功
- [x] 所有 import 更新完成
- [x] 控制台日志更新完成
- [x] 文档更新完成
- [x] 无 `SimpleFormRenderer` 遗留引用
- [ ] 功能测试通过
  - [ ] 测试页面正常
  - [ ] Workspace 正常
  - [ ] 收银台功能正常

### 测试步骤

1. **测试页面**
   ```
   http://localhost:5174/test/form-renderer
   ```
   - 点击"切换测试数据"
   - 验证基础表单渲染
   - 验证提交/分享功能

2. **Workspace**
   ```
   http://localhost:5174/workspace/luobei/testcmp
   ```
   - 选择任意 form 类型函数
   - 验证渲染正常

3. **收银台**
   ```
   http://localhost:5174/workspace/luobei/testcmp/tools/cashier_desk
   ```
   - 验证嵌套结构渲染
   - 验证 List 添加/删除行
   - 验证 Select 回调（待实现）

---

## 💡 命名规范建议

### 通用原则

1. **核心组件**：使用正式名称，不加修饰词
   - ✅ `FormRenderer`
   - ✅ `TableRenderer`
   - ❌ `SimpleFormRenderer`
   - ❌ `BasicTableRenderer`

2. **遗留代码**：使用 `Legacy` 前缀
   - ✅ `LegacyFormRenderer`
   - ✅ `LegacyTableRenderer`

3. **实验性功能**：使用 `Experimental` 前缀
   - ✅ `ExperimentalFileUploader`

4. **特定场景**：使用具体描述
   - ✅ `InlineFormRenderer`（行内表单）
   - ✅ `ModalFormRenderer`（弹窗表单）

---

## 📄 相关文档

- [新旧渲染系统集成方案](新旧渲染系统集成方案.md)
- [核心架构 README](../../src/core/README.md)
- [架构总览](架构总览.md)

