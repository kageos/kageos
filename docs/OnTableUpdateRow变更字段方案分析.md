# OnTableUpdateRow 变更字段方案分析

## 📋 需求概述

**当前问题**：
- 每次更新传递全量数据：`{"id":2,"name":"802","type":"大型","capacity":120,...}`
- 不方便后端做日志审计
- 无法区分哪些字段真正变更了

**目标**：
- 只传递变更的字段
- 同时传递旧值（用于审计）
- 在平台侧或网关侧记录业务日志

## 🔍 可行性分析

### 1. 前端数据流分析

#### 当前实现
```typescript
// TableRenderer.vue
const handleDialogSubmit = async (data: Record<string, any>) => {
  if (dialogMode.value === 'update') {
    // currentRow.value 包含旧值（从 tableData 中获取）
    success = await handleUpdateRow(currentRow.value.id, data)  // data 是全量新值
  }
}

// FormDialog.vue
const handleSubmit = async () => {
  const submitData = formRendererRef.value.prepareSubmitDataWithTypeConversion()  // 全量新值
  emit('submit', submitData)
}
```

#### 关键发现
- ✅ **旧值已存在**：`currentRow.value` 或 `currentDetailRow.value` 包含完整的旧值
- ✅ **新值已存在**：`submitData` 包含完整的新值
- ✅ **可以对比**：前端可以对比旧值和新值，找出变更的字段

### 2. 数据结构修改方案

#### 方案 A：扩展 `OnTableUpdateRowReq`（推荐）

```go
type OnTableUpdateRowReq struct {
    ID        int                    `json:"id"`
    Updates   map[string]interface{} `json:"updates"`   // 只包含变更的字段
    OldValues map[string]interface{} `json:"old_values"` // 旧值（用于审计）
}

// 保持向后兼容
func (c *OnTableUpdateRowReq) GetId() int {
    // 现有逻辑保持不变
}

func (c *OnTableUpdateRowReq) GetUpdates() map[string]interface{} {
    // 现有逻辑保持不变（处理文件类型组件）
    // 如果 OldValues 为空，说明是旧版本，Updates 可能包含全量数据
}
```

**优点**：
- ✅ 向后兼容：如果 `OldValues` 为空，可以认为 `Updates` 是全量数据
- ✅ 清晰明确：旧值和新值分离，便于审计
- ✅ 不影响现有业务代码：`GetUpdates()` 方法保持不变

**缺点**：
- ⚠️ 需要前端修改：对比旧值和新值，只传递变更字段

#### 方案 B：保持现有结构，在网关层处理

```go
// 网关层（CallbackApp）对比旧值和新值
// 需要从数据库查询旧值，增加数据库查询开销
```

**缺点**：
- ❌ 需要查询数据库获取旧值（增加延迟）
- ❌ 网关层需要知道表结构（耦合度高）
- ❌ 无法处理计算字段的变更

### 3. 前端实现方案

#### 实现步骤

1. **在 `FormDialog` 或 `TableRenderer` 中对比旧值和新值**

```typescript
/**
 * 对比旧值和新值，找出变更的字段
 */
function getChangedFields(
  oldValues: Record<string, any>,
  newValues: Record<string, any>
): {
  updates: Record<string, any>,    // 只包含变更的字段
  oldValues: Record<string, any>    // 变更字段的旧值
} {
  const updates: Record<string, any> = {}
  const oldValuesChanged: Record<string, any> = {}
  
  // 遍历新值，找出变更的字段
  for (const key in newValues) {
    const newValue = newValues[key]
    const oldValue = oldValues[key]
    
    // 深度对比（处理对象、数组等）
    if (!isEqual(newValue, oldValue)) {
      updates[key] = newValue
      oldValuesChanged[key] = oldValue
    }
  }
  
  // 处理删除的字段（新值为 null/undefined，但旧值存在）
  for (const key in oldValues) {
    if (!(key in newValues) || newValues[key] === null || newValues[key] === undefined) {
      if (oldValues[key] !== null && oldValues[key] !== undefined) {
        updates[key] = null  // 或 undefined，取决于业务需求
        oldValuesChanged[key] = oldValues[key]
      }
    }
  }
  
  return { updates, oldValues: oldValuesChanged }
}
```

2. **修改 `handleUpdate` 方法**

```typescript
const handleUpdate = async (id: number, data: Record<string, any>, oldData: Record<string, any>): Promise<boolean> => {
  try {
    // 对比旧值和新值，找出变更的字段
    const { updates, oldValues } = getChangedFields(oldData, data)
    
    const updateData = {
      id,
      updates,      // 只包含变更的字段
      old_values: oldValues  // 变更字段的旧值
    }
    
    await tableUpdateRow(functionData.method, functionData.router, updateData)
    // ...
  }
}
```

3. **修改 `TableRenderer` 传递旧值**

```typescript
const handleDialogSubmit = async (data: Record<string, any>): Promise<void> => {
  if (dialogMode.value === 'update') {
    // currentRow.value 是旧值
    success = await handleUpdateRow(currentRow.value.id, data, currentRow.value)
  }
}
```

### 4. 网关侧日志记录

#### 在 `CallbackApp` 中记录审计日志

```go
func (a *App) CallbackApp(c *gin.Context) {
    // ... 现有逻辑 ...
    
    // 如果是 OnTableUpdateRow 回调，记录审计日志
    if callbackType == "OnTableUpdateRow" {
        var updateReq struct {
            ID        int                    `json:"id"`
            Updates   map[string]interface{} `json:"updates"`
            OldValues map[string]interface{} `json:"old_values"`
        }
        
        if err := json.Unmarshal(all, &updateReq); err == nil {
            // 记录审计日志
            auditLog := map[string]interface{}{
                "type":       "table_update",
                "router":     router,
                "method":     method,
                "id":         updateReq.ID,
                "updates":    updateReq.Updates,
                "old_values": updateReq.OldValues,
                "user":       contextx.GetRequestUser(c),
                "timestamp":  time.Now(),
            }
            
            // 发送到日志系统（如 Loki、ELK 等）
            a.auditLogger.Log(auditLog)
        }
    }
    
    // ... 继续现有逻辑 ...
}
```

### 5. 影响范围分析

#### 需要修改的文件

**前端**：
1. `web/src/composables/useTableOperations.ts` - 修改 `handleUpdate` 方法
2. `web/src/components/TableRenderer.vue` - 传递旧值
3. `web/src/components/FormDialog.vue` - 可选，如果在这里对比
4. `web/src/utils/objectDiff.ts` - 新增：深度对比工具函数

**后端 SDK**：
1. `sdk/agent-app/callback/table.go` - 修改 `OnTableUpdateRowReq` 结构
2. `sdk/agent-app/app/register.go` - 可选，如果需要特殊处理

**后端网关**：
1. `core/app-server/api/v1/app.go` - 添加审计日志记录

#### 向后兼容性

**方案 A（推荐）**：
- ✅ 完全向后兼容
- ✅ 如果 `OldValues` 为空，可以认为 `Updates` 是全量数据（旧版本行为）
- ✅ 现有业务代码无需修改（`GetUpdates()` 方法保持不变）

**方案 B**：
- ❌ 需要修改所有现有业务代码
- ❌ 破坏向后兼容性

### 6. 潜在问题和解决方案

#### 问题 1：深度对比的性能问题
**解决方案**：
- 使用高效的深度对比库（如 `lodash.isEqual`）
- 对于大对象，可以考虑只对比顶层字段（根据业务需求）

#### 问题 2：计算字段的变更
**问题**：某些字段是计算字段（如 `status`），不在数据库中，但需要记录变更
**解决方案**：
- 前端对比时，包含所有字段（包括计算字段）
- 后端在 `GetUpdates()` 中过滤掉计算字段（如果不需要更新）

#### 问题 3：文件类型组件的变更
**问题**：文件类型组件是复杂对象，如何判断是否变更？
**解决方案**：
- 对比文件的 URL 或 ID（而不是整个对象）
- 在 `GetUpdates()` 中已经处理了文件类型组件的序列化

### 7. 实施建议

#### 阶段 1：前端实现（不影响后端）
1. 实现 `getChangedFields` 工具函数
2. 修改 `handleUpdate` 方法，对比旧值和新值
3. 传递 `old_values` 字段（即使后端暂时不使用）

#### 阶段 2：后端 SDK 修改
1. 修改 `OnTableUpdateRowReq` 结构，添加 `OldValues` 字段
2. 保持 `GetUpdates()` 方法向后兼容
3. 更新文档和示例

#### 阶段 3：网关审计日志
1. 在 `CallbackApp` 中记录审计日志
2. 集成日志系统（Loki、ELK 等）
3. 添加日志查询和分析功能

### 8. 总结

**可行性**：✅ **完全可行**

**优势**：
- ✅ 前端已有旧值和新值，可以轻松对比
- ✅ 向后兼容，不影响现有业务代码
- ✅ 便于审计和日志记录
- ✅ 减少网络传输（只传递变更字段）

**风险**：
- ⚠️ 需要前端实现深度对比逻辑（但可以使用现有库）
- ⚠️ 需要测试各种边界情况（null、undefined、对象、数组等）

**建议**：
- ✅ 采用**方案 A**（扩展 `OnTableUpdateRowReq`）
- ✅ 分阶段实施，先前端，再后端，最后网关
- ✅ 保持向后兼容，确保平滑升级

