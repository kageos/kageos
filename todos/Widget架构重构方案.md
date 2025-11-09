# Widget 架构重构方案

## 一、问题说明

### 1.1 当前问题

#### 核心问题：递归更新导致 "Maximum recursive updates exceeded" 错误

**问题表现**：
- 在提交表单后，当响应参数中包含 `table` 类型的字段时，会出现递归更新错误
- `ResponseTableWidget.render()` 被频繁调用，每次生成新的 VNode
- 即使使用了 `toRaw`、`v-memo`、`computed` 缓存等方案，问题仍然存在

**问题根源**：
1. **函数式渲染 vs 声明式组件的冲突**
   - `widget.render()` 每次调用都返回新的 VNode 对象
   - Vue 的 `component :is` 期望组件是稳定的
   - 每次 VNode 变化都触发重新渲染

2. **Widget 实例生命周期管理混乱**
   - Widget 实例被缓存在 `allWidgets` Map 中
   - 但每次 `render()` 都创建新的 VNode
   - 状态在实例中，但渲染在函数中

3. **响应式数据滥用**
   - 过度使用响应式数据（ref、computed）
   - 在 render 函数中读取响应式数据
   - watch 监听器链式触发

### 1.2 架构设计问题

#### 问题1: 混合使用类和函数式渲染
- Widget 是类实例，但通过 `render()` 方法返回 VNode
- 每次渲染都调用 `render()`，返回新的 VNode
- Vue 无法正确追踪组件实例的生命周期

#### 问题2: 响应式追踪混乱
- 在 render 过程中读取响应式数据
- 即使使用 `toRaw`，问题仍然存在（因为 computed 本身是响应式的）
- watch 监听器在 render 过程中设置，导致链式触发

#### 问题3: 状态管理分散
- Widget 内部状态（如抽屉状态）使用 ref
- FormRenderer 中也有状态（如 responseData）
- 状态分散，难以管理和追踪

## 二、重构前的架构逻辑

### 2.1 当前架构概览

```
FormRenderer (Vue 组件)
  ├─ renderField() - 渲染请求参数字段
  │   └─ WidgetBuilder.create() - 创建 Widget 实例
  │       └─ widget.render() - 返回 VNode
  │
  ├─ renderResponseField() - 渲染响应参数字段
  │   └─ new ResponseTableWidget() - 创建 Widget 实例
  │       └─ widget.render() - 返回 VNode
  │
  └─ getResponseFieldVNode() - 缓存 VNode
      └─ computed(() => renderResponseField())
```

### 2.2 Widget 类设计

#### BaseWidget 基类
```typescript
abstract class BaseWidget {
  // 核心方法
  abstract render(): any  // 主渲染方法（请求参数）
  
  // 不同场景的渲染方法
  renderTableCell(value?: FieldValue): any  // 表格单元格渲染
  renderForResponse(): any  // 响应参数渲染
  renderForDetail(value?: FieldValue): any  // 详情渲染
  renderSearchInput(searchType: string): any  // 搜索输入渲染
}
```

#### 不同场景的渲染
1. **请求参数渲染** (`render()`)
   - 可编辑的表单字段
   - 支持 onChange 回调
   - 需要 formManager

2. **响应参数渲染** (`renderForResponse()`)
   - 只读展示
   - 不需要 onChange
   - 可能不需要 formManager

3. **表格单元格渲染** (`renderTableCell()`)
   - 表格中的单元格展示
   - 可能是简化显示
   - 临时 Widget（不需要 formManager）

4. **详情渲染** (`renderForDetail()`)
   - 详情抽屉中的展示
   - 可能是完整展示
   - 临时 Widget

5. **搜索输入渲染** (`renderSearchInput()`)
   - 搜索表单中的输入框
   - 根据搜索类型返回不同的输入组件
   - 临时 Widget

### 2.3 Widget 工厂模式

```typescript
class WidgetFactory {
  private widgetMap: Map<string, typeof BaseWidget>
  private responseWidgetMap: Map<string, typeof BaseWidget>
  
  getWidgetClass(type: string): typeof BaseWidget
  getResponseWidgetClass(type: string): typeof BaseWidget | null
}
```

**设计原则**：
- 依赖倒置：通过工厂模式创建 Widget，不直接依赖具体实现
- 扩展性：新增组件只需注册到工厂，无需修改其他代码

### 2.4 聚合计算功能

#### 当前实现
```typescript
// TableWidget 中的聚合计算
private recalculateStatistics(): void {
  const allRows = this.getAllRowsData()
  const result: Record<string, any> = {}
  
  for (const [label, expression] of Object.entries(this.statisticsConfig.value)) {
    const value = ExpressionParser.evaluate(expression, allRows)
    result[label] = value
  }
  
  this.statisticsResult.value = result
}
```

**特点**：
- 在 TableWidget 内部实现
- 使用 ExpressionParser 解析表达式
- 支持 sum、count、avg、min、max 等聚合函数
- 支持乘法聚合：`sum(价格,*数量)`
- 支持 List 层聚合：`list_sum(字段)`

### 2.5 当前架构的优缺点

#### 优点
1. ✅ **依赖倒置原则**：通过工厂模式，FormRenderer 不直接依赖具体 Widget
2. ✅ **扩展性好**：新增组件只需注册到工厂
3. ✅ **功能完整**：支持多种渲染场景
4. ✅ **聚合计算**：支持复杂的表达式计算

#### 缺点
1. ❌ **递归更新问题**：函数式渲染导致递归更新
2. ❌ **生命周期管理混乱**：Widget 实例和 VNode 分离
3. ❌ **响应式追踪混乱**：在 render 过程中读取响应式数据
4. ❌ **状态管理分散**：状态分散在 Widget 和 FormRenderer 中

## 三、重构后的架构设计

### 3.1 核心设计理念

#### 设计原则
1. **组件化**：将 Widget 改为 Vue 组件，利用 Vue 的生命周期管理
2. **状态集中**：使用 Pinia Store 集中管理状态
3. **依赖倒置**：保持工厂模式，不违背依赖倒置原则
4. **场景分离**：不同渲染场景使用不同的组件或 props

### 3.2 新架构概览

```
FormRenderer (Vue 组件)
  ├─ RequestFieldRenderer (Vue 组件)
  │   └─ <component :is="getRequestWidgetComponent(field)" />
  │       └─ InputWidgetComponent.vue
  │       └─ TableWidgetComponent.vue
  │       └─ FormWidgetComponent.vue
  │
  ├─ ResponseFieldRenderer (Vue 组件)
  │   └─ <component :is="getResponseWidgetComponent(field)" />
  │       └─ ResponseTableWidgetComponent.vue
  │       └─ ResponseFormWidgetComponent.vue
  │
  └─ ResponseDataStore (Pinia Store)
      └─ 集中管理响应数据
```

### 3.3 Widget 组件化设计

#### 方案A: 完全组件化（推荐）

**每个 Widget 对应一个 Vue 组件**：

```
widgets/
  ├─ InputWidget.vue
  ├─ TableWidget.vue
  ├─ FormWidget.vue
  ├─ ResponseTableWidget.vue
  └─ ResponseFormWidget.vue
```

**组件接口设计**：
```vue
<!-- InputWidget.vue -->
<script setup lang="ts">
interface Props {
  field: FieldConfig
  value: FieldValue
  formManager?: ReactiveFormDataManager
  formRenderer?: FormRendererContext
  depth?: number
  // 场景标识
  mode?: 'edit' | 'response' | 'table-cell' | 'detail' | 'search'
}

const props = withDefaults(defineProps<Props>(), {
  mode: 'edit',
  depth: 0
})

// 根据 mode 决定渲染方式
const renderContent = computed(() => {
  switch (props.mode) {
    case 'edit':
      return renderEditMode()
    case 'response':
      return renderResponseMode()
    case 'table-cell':
      return renderTableCellMode()
    case 'detail':
      return renderDetailMode()
    case 'search':
      return renderSearchMode()
  }
})
</script>
```

#### 方案B: 混合方案（渐进式）

**保留 Widget 类，但提供 Vue 组件包装**：

```
widgets/
  ├─ InputWidget.ts (类)
  ├─ InputWidgetComponent.vue (组件包装)
  ├─ TableWidget.ts (类)
  └─ TableWidgetComponent.vue (组件包装)
```

**组件包装设计**：
```vue
<!-- InputWidgetComponent.vue -->
<script setup lang="ts">
import { InputWidget } from './InputWidget'

const props = defineProps<{...}>()

// 创建 Widget 实例
const widget = new InputWidget({...})

// 使用 Widget 的方法，但通过组件渲染
const renderResult = computed(() => {
  switch (props.mode) {
    case 'edit':
      return widget.render()
    case 'response':
      return widget.renderForResponse()
    case 'table-cell':
      return widget.renderTableCell()
    // ...
  }
})
</script>
```

### 3.4 不同场景的渲染方案

#### 场景1: 请求参数渲染（可编辑）
```vue
<RequestFieldRenderer
  :field="field"
  :value="formManager.getValue(field.code)"
  :form-manager="formManager"
  :form-renderer="formRendererContext"
  mode="edit"
/>
```

#### 场景2: 响应参数渲染（只读）
```vue
<ResponseFieldRenderer
  :field="field"
  :value="responseDataStore.data?.[field.code]"
  mode="response"
/>
```

#### 场景3: 表格单元格渲染
```vue
<!-- 在 TableWidgetComponent 中 -->
<el-table-column>
  <template #default="{ row }">
    <TableCellRenderer
      :field="field"
      :value="row[field.code]"
      mode="table-cell"
    />
  </template>
</el-table-column>
```

#### 场景4: 详情渲染
```vue
<!-- 在详情抽屉中 -->
<DetailRenderer
  :field="field"
  :value="detailValue"
  mode="detail"
/>
```

#### 场景5: 搜索输入渲染
```vue
<!-- 在搜索表单中 -->
<SearchInputRenderer
  :field="field"
  :search-type="searchType"
  mode="search"
/>
```

### 3.5 状态管理设计

#### Pinia Store 设计

```typescript
// stores/responseData.ts
export const useResponseDataStore = defineStore('responseData', () => {
  const data = ref<any>(null)
  const renderTrigger = ref(0)
  
  function setData(newData: any) {
    data.value = newData
    renderTrigger.value++
  }
  
  return { data, renderTrigger, setData }
})

// stores/formData.ts
export const useFormDataStore = defineStore('formData', () => {
  const data = reactive<Map<string, FieldValue>>(new Map())
  
  function setValue(fieldPath: string, value: FieldValue) {
    data.set(fieldPath, value)
  }
  
  function getValue(fieldPath: string): FieldValue {
    return data.get(fieldPath) || { raw: null, display: '', meta: {} }
  }
  
  return { data, setValue, getValue }
})
```

### 3.6 工厂模式保持

```typescript
// factories/WidgetComponentFactory.ts
class WidgetComponentFactory {
  private componentMap: Map<string, Component>
  private responseComponentMap: Map<string, Component>
  
  getRequestComponent(type: string): Component
  getResponseComponent(type: string): Component | null
  getTableCellComponent(type: string): Component
  getDetailComponent(type: string): Component
  getSearchInputComponent(type: string): Component
}
```

**设计原则**：
- 保持依赖倒置原则
- 新增组件只需注册到工厂
- 不修改其他代码

### 3.7 聚合计算功能保持

#### 重构后的实现

```typescript
// 在 TableWidgetComponent.vue 中
const statisticsResult = computed(() => {
  if (!statisticsConfig.value) return {}
  
  const allRows = getAllRowsData()
  const result: Record<string, any> = {}
  
  for (const [label, expression] of Object.entries(statisticsConfig.value)) {
    result[label] = ExpressionParser.evaluate(expression, allRows)
  }
  
  return result
})
```

**特点**：
- 使用 `computed` 自动计算
- 当数据变化时自动重新计算
- 不需要手动调用 `recalculateStatistics()`
- 性能更好（Vue 的 computed 有缓存）

### 3.8 组件注册机制

```typescript
// factories/WidgetComponentFactory.ts
export const widgetComponentFactory = new WidgetComponentFactory()

// 注册请求参数组件
widgetComponentFactory.registerRequestComponent('input', InputWidgetComponent)
widgetComponentFactory.registerRequestComponent('table', TableWidgetComponent)
widgetComponentFactory.registerRequestComponent('form', FormWidgetComponent)

// 注册响应参数组件
widgetComponentFactory.registerResponseComponent('table', ResponseTableWidgetComponent)
widgetComponentFactory.registerResponseComponent('form', ResponseFormWidgetComponent)

// 注册表格单元格组件
widgetComponentFactory.registerTableCellComponent('input', InputTableCellComponent)
widgetComponentFactory.registerTableCellComponent('form', FormTableCellComponent)
```

## 四、重构能否解决问题？

### 4.1 递归更新问题

#### ✅ 可以解决

**原因**：
1. **组件实例稳定**：Vue 组件实例不会每次渲染都重新创建
2. **Props 传递**：通过 props 传递数据，Vue 会自动优化更新
3. **生命周期管理**：Vue 的生命周期可以更好地控制响应式追踪
4. **隔离响应式**：组件内部的响应式数据不会影响父组件

**实现要点**：
- 使用 `v-memo` 进一步优化
- 使用 `toRaw` 读取 props（如果需要）
- 使用 `shallowRef` 存储数据
- 通过 emit 通知父组件状态变化

### 4.2 扩展性问题

#### ✅ 可以保持

**原因**：
1. **工厂模式保持**：仍然使用工厂模式，不违背依赖倒置原则
2. **组件注册机制**：新增组件只需注册，无需修改其他代码
3. **场景分离**：不同场景使用不同的组件或 props，清晰明确

**新增组件步骤**：
1. 创建组件文件（如 `NewWidgetComponent.vue`）
2. 在工厂中注册组件
3. 完成（无需修改其他代码）

### 4.3 聚合计算功能

#### ✅ 可以保持并优化

**原因**：
1. **功能保持**：聚合计算逻辑不变，只是从类方法改为 computed
2. **性能优化**：使用 Vue 的 computed，有自动缓存
3. **响应式优化**：当数据变化时自动重新计算，不需要手动调用

**实现方式**：
```typescript
// 在 TableWidgetComponent.vue 中
const statisticsResult = computed(() => {
  // 聚合计算逻辑
  return ExpressionParser.evaluate(expression, allRows)
})
```

### 4.4 多种渲染场景

#### ✅ 可以支持

**方案**：
1. **方案A**：每个场景一个组件（如 `InputEditComponent.vue`、`InputResponseComponent.vue`）
2. **方案B**：一个组件，通过 `mode` prop 区分场景（推荐）

**推荐方案B**：
- 代码复用性更好
- 维护成本更低
- 扩展性更好

## 五、重构实施计划

### 5.1 第一阶段：基础组件化

1. **创建基础组件**
   - `InputWidgetComponent.vue`
   - `TableWidgetComponent.vue`
   - `FormWidgetComponent.vue`

2. **创建响应组件**
   - `ResponseTableWidgetComponent.vue`
   - `ResponseFormWidgetComponent.vue`

3. **创建工厂**
   - `WidgetComponentFactory.ts`

4. **测试基础功能**
   - 请求参数渲染
   - 响应参数渲染

### 5.2 第二阶段：状态管理

1. **创建 Pinia Store**
   - `responseData.ts`
   - `formData.ts`

2. **迁移状态**
   - 将 `responseData` 迁移到 Store
   - 将 `formManager` 的部分功能迁移到 Store

3. **测试状态管理**
   - 确保状态更新正常
   - 确保组件响应正常

### 5.3 第三阶段：场景支持

1. **实现表格单元格渲染**
   - `TableCellRenderer.vue`

2. **实现详情渲染**
   - `DetailRenderer.vue`

3. **实现搜索输入渲染**
   - `SearchInputRenderer.vue`

4. **测试所有场景**
   - 确保所有场景都能正常工作

### 5.4 第四阶段：聚合计算

1. **迁移聚合计算**
   - 将 `recalculateStatistics()` 改为 `computed`

2. **测试聚合计算**
   - 确保计算结果正确
   - 确保性能正常

### 5.5 第五阶段：优化和清理

1. **性能优化**
   - 使用 `v-memo` 优化渲染
   - 使用 `shallowRef` 优化状态

2. **代码清理**
   - 移除旧的 Widget 类（如果不再需要）
   - 清理未使用的代码

3. **文档更新**
   - 更新开发文档
   - 更新组件使用文档

## 六、风险评估

### 6.1 技术风险

#### 风险1: 组件化可能引入新问题
**缓解措施**：
- 先实现一个简单的组件测试
- 逐步迁移，不一次性重构所有组件
- 充分测试每个阶段

#### 风险2: 状态管理可能影响现有功能
**缓解措施**：
- 保持向后兼容
- 逐步迁移状态
- 充分测试

### 6.2 时间风险

#### 风险: 重构工作量大
**缓解措施**：
- 分阶段实施
- 优先解决核心问题（递归更新）
- 其他功能逐步迁移

### 6.3 兼容性风险

#### 风险: 可能影响现有功能
**缓解措施**：
- 保持接口兼容
- 充分测试
- 准备回滚方案

## 七、总结

### 7.1 重构目标

1. ✅ **解决递归更新问题**：通过组件化彻底解决
2. ✅ **保持扩展性**：保持工厂模式，不违背依赖倒置原则
3. ✅ **支持多种场景**：通过 mode prop 支持不同渲染场景
4. ✅ **保持核心功能**：聚合计算等功能保持不变

### 7.2 重构优势

1. **彻底解决递归更新问题**
2. **更好的性能**：利用 Vue 的自动优化
3. **更好的可维护性**：组件化架构更清晰
4. **更好的扩展性**：新增组件更容易

### 7.3 重构挑战

1. **工作量大**：需要重构大量代码
2. **测试工作**：需要充分测试所有场景
3. **学习成本**：团队需要适应新架构

### 7.4 建议

**建议采用方案A（完全组件化）**，因为：
1. 可以彻底解决递归更新问题
2. 架构更清晰，更符合 Vue 3 的最佳实践
3. 虽然工作量大，但长期收益更大

**实施策略**：
1. 先实现一个简单的组件测试（如 `InputWidgetComponent.vue`）
2. 验证方案可行性
3. 如果成功，逐步迁移其他组件
4. 如果失败，分析原因并调整

