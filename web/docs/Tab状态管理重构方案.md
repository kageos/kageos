# Tab 状态管理重构方案

## 一、问题分析

### 1.1 核心问题

**Tab 切换时状态丢失**，具体表现为：
- 用户在 Tab A 中设置搜索条件（如"紧急"）
- 切换到 Tab B
- 再切换回 Tab A 时，搜索条件丢失

### 1.2 问题根本原因

当前实现中，状态保存和恢复的时机混乱，导致状态在切换过程中被覆盖或丢失：

**问题 1：保存时机不正确**
```typescript
// 当前实现：只在 handleTabClick 中保存
const handleTabClick = (tab: any) => {
  // 保存当前 Tab 的状态
  saveCurrentTabState()
  // 切换路由
  router.replace({ path: targetPath, query: {} })
}
```

**致命缺陷**：
- 用户通过**服务目录切换**时，不会触发 `handleTabClick`
- 用户通过**点击 Tab 标签**时，才会触发 `handleTabClick`
- 导致服务目录切换时，状态根本没有保存

**问题 2：恢复时机不正确**
```typescript
// 当前实现：在 watch activeTabId 中恢复
watch(() => stateManager.getState().activeTabId, (newId, oldId) => {
  // 恢复新 Tab 数据
  if (newTab.data) {
    serviceFactoryInstance.getTableStateManager().setState(newTab.data)
  }
})
```

**致命缺陷**：
- `watch activeTabId` 是**异步触发**的，晚于组件挂载
- `TableView` 的 `onMounted` 会立即调用 `initializeTable()`
- `initializeTable()` 从 `TableStateManager` 获取状态时，状态还没有被恢复
- 导致每次都是空状态，丢失了搜索参数

**问题 3：状态覆盖冲突**
```typescript
// useTableInitialization.ts
const initializeTable = async () => {
  // 从 TableStateManager 获取状态
  const currentState = stateManager.getState()
  
  // 如果 URL 没有 query 参数，重置状态
  if (!hasQueryParams) {
    stateManager.setState({
      searchForm: {},  // 清空搜索表单
      sorts: defaultSorts,
      pagination: { currentPage: 1, pageSize: 20, total: 0 }
    })
  }
  
  // 同步到 URL
  syncToURL()
  
  // 加载数据
  await loadTableData()
}
```

**致命缺陷**：
- `initializeTable()` 会重置状态，即使 `watch activeTabId` 已经恢复了状态
- 因为执行顺序是：`onMounted` → `initializeTable()` → `watch activeTabId` 触发
- 导致恢复的状态立即被覆盖

### 1.3 问题示例（用户操作流程）

**场景：通过服务目录切换**

1. 用户在"工单管理"（Tab A）中设置搜索条件："紧急"
2. 用户点击服务目录中的"会议室管理"（Tab B）
3. 触发流程：
   - `handleNodeClick()` → `router.replace()` → 路由变化
   - `watch activeTabId` 触发 → 尝试恢复 Tab B 的状态
   - **问题**：Tab A 的状态从未被保存（因为 `handleTabClick` 没触发）
4. 用户再点击服务目录中的"工单管理"（Tab A）
5. 触发流程：
   - `handleNodeClick()` → `router.replace()` → 路由变化
   - `watch activeTabId` 触发 → 尝试恢复 Tab A 的状态
   - **问题**：Tab A 没有保存的数据（`tab.data` 为空）
   - `initializeTable()` 重置状态 → 搜索条件"紧急"丢失

**场景：通过 Tab 标签切换**

1. 用户在"工单管理"（Tab A）中设置搜索条件："紧急"
2. 用户点击"会议室管理"（Tab B）标签
3. 触发流程：
   - `handleTabClick()` → 保存 Tab A 的状态 → `router.replace()`
   - `watch activeTabId` 触发 → 恢复 Tab B 的状态
   - **问题**：`TableView` 的 `onMounted` 比 `watch activeTabId` 更早执行
   - `initializeTable()` 发现 `TableStateManager` 还是空的（因为 `watch` 还没触发）
   - `initializeTable()` 重置状态 → Tab B 的状态丢失
4. 用户再点击"工单管理"（Tab A）标签
5. 触发流程：
   - `handleTabClick()` → 保存 Tab B 的状态（此时已被重置） → `router.replace()`
   - `watch activeTabId` 触发 → 恢复 Tab A 的状态
   - **问题**：同样的时序问题，Tab A 的状态也可能被 `initializeTable()` 覆盖

### 1.4 核心矛盾

**时序冲突**：
- `watch activeTabId` 是异步的，晚于 `TableView.onMounted`
- `TableView.onMounted` 立即调用 `initializeTable()`
- `initializeTable()` 需要从 `TableStateManager` 获取状态
- 但 `TableStateManager` 的状态还没有被 `watch activeTabId` 恢复

**触发源冲突**：
- 有两种切换方式：服务目录切换、Tab 标签切换
- `handleTabClick` 只能处理 Tab 标签切换
- 服务目录切换不会触发 `handleTabClick`，导致状态无法保存

## 二、重构目标

### 2.1 功能目标

1. **状态持久化**：Tab 切换时，完整保存当前 Tab 的状态（搜索表单、排序、分页、数据）
2. **状态恢复**：切换回 Tab 时，完整恢复之前保存的状态
3. **状态隔离**：每个 Tab 的状态独立，不会相互污染
4. **统一入口**：无论通过服务目录切换还是 Tab 标签切换，都能正确保存和恢复状态

### 2.2 技术目标

1. **时序正确**：保证状态在正确的时机被保存和恢复
2. **单一职责**：每个函数只负责一件事（保存、恢复、初始化）
3. **可维护性**：逻辑清晰，易于调试和扩展

## 三、重构方案

### 3.1 核心思路

**原则 1：提前恢复状态**
- 在 `TableView.onMounted` 之前，先恢复状态到 `TableStateManager`
- `initializeTable()` 直接从 `TableStateManager` 获取状态，无需特殊处理

**原则 2：统一保存时机**
- 在 `watch activeTabId` 中保存旧 Tab 的状态（**同步执行**）
- 无论是服务目录切换还是 Tab 标签切换，都会触发 `watch activeTabId`

**原则 3：简化初始化逻辑**
- `initializeTable()` 只负责：
  1. 从 `TableStateManager` 获取状态
  2. 同步状态到 URL
  3. 加载数据
- 不再判断 URL 参数、不再重置状态、不再从 Tab 数据恢复

### 3.2 实现方案

#### 3.2.1 修改 `useWorkspaceTabs.ts`

**关键修改点 1：在 `watch activeTabId` 中同步保存和恢复**

```typescript
const setupTabDataWatch = () => {
  watch(() => stateManager.getState().activeTabId, (newId, oldId) => {
    console.log('[useWorkspaceTabs] watch activeTabId 触发', { oldId, newId })
    
    // 🔥 步骤 1：同步保存旧 Tab 的状态（必须在恢复新 Tab 之前）
    if (oldId) {
      const oldTab = tabs.value.find(t => t.id === oldId)
      if (oldTab && oldTab.node) {
        const detail = stateManager.getFunctionDetail(oldTab.node)
        if (detail?.template_type === 'table') {
          // 从 TableStateManager 获取当前状态并保存
          const tableStateManager = serviceFactoryInstance.getTableStateManager()
          const currentState = tableStateManager.getState()
          
          oldTab.data = {
            searchForm: { ...currentState.searchForm },
            searchParams: { ...currentState.searchParams },
            sorts: [...currentState.sorts],
            hasManualSort: currentState.hasManualSort,
            pagination: { ...currentState.pagination },
            data: [...currentState.data],
            loading: false,
            sortParams: currentState.sortParams
          }
          
          console.log('[useWorkspaceTabs] 保存旧 Tab 状态', {
            tabId: oldId,
            searchForm: oldTab.data.searchForm,
            searchFormKeys: Object.keys(oldTab.data.searchForm || {}),
            sorts: oldTab.data.sorts,
            pagination: oldTab.data.pagination
          })
        } else if (detail?.template_type === 'form') {
          const currentState = serviceFactoryInstance.getFormStateManager().getState()
          oldTab.data = {
            data: Array.from(currentState.data.entries()),
            errors: Array.from(currentState.errors.entries()),
            submitting: currentState.submitting
          }
        }
      }
    }
    
    // 🔥 步骤 2：立即恢复新 Tab 的状态（在 TableView.onMounted 之前）
    if (newId) {
      const newTab = tabs.value.find(t => t.id === newId)
      if (newTab && newTab.data && newTab.node) {
        const detail = stateManager.getFunctionDetail(newTab.node)
        if (detail?.template_type === 'table') {
          // 立即恢复到 TableStateManager
          serviceFactoryInstance.getTableStateManager().setState({
            searchForm: newTab.data.searchForm || {},
            searchParams: newTab.data.searchParams || {},
            sorts: newTab.data.sorts || [],
            hasManualSort: newTab.data.hasManualSort || false,
            pagination: newTab.data.pagination || {
              currentPage: 1,
              pageSize: 20,
              total: 0
            },
            data: newTab.data.data || [],
            loading: false,
            sortParams: newTab.data.sortParams || null
          })
          
          console.log('[useWorkspaceTabs] 恢复新 Tab 状态', {
            tabId: newId,
            searchForm: newTab.data.searchForm,
            searchFormKeys: Object.keys(newTab.data.searchForm || {}),
            sorts: newTab.data.sorts,
            pagination: newTab.data.pagination
          })
        } else if (detail?.template_type === 'form') {
          serviceFactoryInstance.getFormStateManager().setState({
            data: new Map(newTab.data.data),
            errors: new Map(newTab.data.errors),
            submitting: newTab.data.submitting
          })
        }
      } else {
        // 新 Tab 没有保存的数据，重置为默认状态
        if (newTab?.node) {
          const detail = stateManager.getFunctionDetail(newTab.node)
          if (detail?.template_type === 'table') {
            serviceFactoryInstance.getTableStateManager().setState({
              data: [],
              loading: false,
              searchParams: {},
              searchForm: {},
              sortParams: null,
              sorts: [],
              hasManualSort: false,
              pagination: {
                currentPage: 1,
                pageSize: 20,
                total: 0
              }
            })
            console.log('[useWorkspaceTabs] 新 Tab 没有保存数据，重置状态', { tabId: newId })
          }
        }
      }
    }
  })
}
```

**关键修改点 2：移除 `handleTabClick` 中的保存逻辑**

```typescript
const handleTabClick = (tab: any) => {
  // ... 提取 tabId 的逻辑不变 ...
  
  const targetTab = tabs.value.find(t => t.id === tabId)
  if (!targetTab || !targetTab.path) {
    console.warn('[useWorkspaceTabs] handleTabClick: 未找到对应的 tab', {
      tabId,
      availableTabs: tabs.value.map(t => ({ id: t.id, path: t.path }))
    })
    return
  }
  
  console.log('[useWorkspaceTabs] handleTabClick: 处理 Tab 点击', {
    tabId,
    currentActiveTabId: activeTabId.value
  })
  
  // 🔥 直接切换路由，保存逻辑由 watch activeTabId 统一处理
  const tabPath = targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`
  const targetPath = `/workspace${tabPath}`
  
  router.replace({ path: targetPath, query: {} }).catch((err) => {
    console.error('[useWorkspaceTabs] handleTabClick: 路由更新失败', err)
  })
}
```

**关键修改点 3：移除 `saveCurrentTabState` 函数（不再需要）**

#### 3.2.2 修改 `useTableInitialization.ts`

**关键修改点：简化 `initializeTable()` 逻辑**

```typescript
const initializeTable = async (): Promise<void> => {
  const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
  const functionId = functionDetailValue?.id
  const router = functionDetailValue?.router
  
  if (isInitializing.value) {
    Logger.warn('useTableInitialization', '正在初始化中，跳过', { functionId, router })
    return
  }
  
  if (isMounted && !isMounted.value) {
    Logger.warn('useTableInitialization', '组件已卸载，跳过初始化', { functionId, router })
    return
  }
  
  isInitializing.value = true
  
  try {
    // 🔥 步骤 1：从 TableStateManager 获取状态（已由 watch activeTabId 恢复）
    const currentState = stateManager.getState()
    
    Logger.debug('useTableInitialization', '开始初始化', {
      functionId,
      router,
      searchForm: currentState.searchForm,
      searchFormKeys: Object.keys(currentState.searchForm || {}),
      sorts: currentState.sorts,
      pagination: currentState.pagination
    })
    
    // 🔥 步骤 2：同步状态到 URL
    if (!isSyncingToURL.value) {
      isSyncingToURL.value = true
      await nextTick()
      syncToURL() // 完整同步所有参数（分页、排序、搜索）
      await nextTick()
      isSyncingToURL.value = false
    }
    
    // 🔥 步骤 3：加载数据
    if (isMounted && !isMounted.value) {
      Logger.warn('useTableInitialization', '组件在初始化过程中已卸载，取消加载数据', { functionId, router })
      return
    }
    
    Logger.debug('useTableInitialization', '开始加载数据', { functionId, router })
    await loadTableData()
    Logger.debug('useTableInitialization', '数据加载完成', { functionId, router })
  } finally {
    isInitializing.value = false
    Logger.debug('useTableInitialization', 'initializeTable 完成', { functionId, router })
  }
}
```

**关键修改点：简化 `watch route.query` 逻辑**

```typescript
watch(() => route.query, async (newQuery: any, oldQuery: any) => {
  const functionDetailValue = 'value' in functionDetail ? functionDetail.value : functionDetail
  const functionId = functionDetailValue?.id
  const router = functionDetailValue?.router
  
  // 检查路由是否匹配当前函数
  const currentPath = extractWorkspacePath(route.path)
  const expectedPath = (router || '').replace(/^\/+/, '')
  const pathMatches = currentPath === expectedPath || currentPath.startsWith(expectedPath + '?')
  
  if (!pathMatches) {
    Logger.debug('useTableInitialization', '路由不匹配当前函数，忽略 URL 变化', {
      functionId,
      router,
      currentPath,
      expectedPath
    })
    return
  }
  
  if (isMounted && !isMounted.value) {
    Logger.warn('useTableInitialization', '组件已卸载，忽略 URL 变化', { functionId, router })
    return
  }
  
  if (isSyncingToURL.value || isRestoringFromURL.value || isInitializing.value) {
    Logger.debug('useTableInitialization', '正在同步或初始化中，忽略 URL 变化', {
      functionId,
      router,
      isSyncingToURL: isSyncingToURL.value,
      isRestoringFromURL: isRestoringFromURL.value,
      isInitializing: isInitializing.value
    })
    return
  }
  
  // 🔥 从 URL 恢复状态到 TableStateManager
  isRestoringFromURL.value = true
  try {
    restoreFromURL()
    
    // 重新加载数据
    if (isMounted && !isMounted.value) {
      Logger.warn('useTableInitialization', '组件在 URL 恢复过程中已卸载，取消加载数据', { functionId, router })
      return
    }
    
    const currentPathAfterRestore = extractWorkspacePath(route.path)
    const pathMatchesAfterRestore = currentPathAfterRestore === expectedPath || currentPathAfterRestore.startsWith(expectedPath + '?')
    if (!pathMatchesAfterRestore) {
      Logger.debug('useTableInitialization', '路由在恢复过程中已变化，取消加载数据', {
        functionId,
        router,
        currentPathAfterRestore,
        expectedPath
      })
      return
    }
    
    Logger.debug('useTableInitialization', 'URL 变化后开始加载数据', { functionId, router })
    await loadTableData()
  } finally {
    isRestoringFromURL.value = false
  }
}, { deep: true })
```

### 3.3 流程图

#### 3.3.1 Tab 切换流程（服务目录切换）

```
用户点击服务目录 "会议室管理"
    ↓
handleNodeClick()
    ↓
router.replace('/workspace/luobei/demo/crm/meeting_room_list')
    ↓
路由变化
    ↓
watch activeTabId 触发（同步执行）
    ↓
步骤 1：保存旧 Tab 状态到 tab.data
    - 从 TableStateManager 获取当前状态
    - 保存到 oldTab.data（深拷贝）
    ↓
步骤 2：恢复新 Tab 状态到 TableStateManager
    - 从 newTab.data 获取保存的状态
    - 恢复到 TableStateManager
    ↓
TableView.onMounted
    ↓
initializeTable()
    ↓
步骤 1：从 TableStateManager 获取状态（已恢复）
步骤 2：同步状态到 URL
步骤 3：加载数据
```

#### 3.3.2 Tab 切换流程（Tab 标签切换）

```
用户点击 Tab 标签 "会议室管理"
    ↓
handleTabClick()
    ↓
router.replace('/workspace/luobei/demo/crm/meeting_room_list')
    ↓
（后续流程与服务目录切换完全相同）
```

### 3.4 关键改进点

1. **统一保存时机**：
   - 无论是服务目录切换还是 Tab 标签切换，都在 `watch activeTabId` 中保存
   - 保证所有切换方式都能正确保存状态

2. **提前恢复状态**：
   - 在 `watch activeTabId` 中立即恢复状态到 `TableStateManager`
   - `TableView.onMounted` 执行时，状态已经恢复好了

3. **简化初始化逻辑**：
   - `initializeTable()` 不再判断 URL 参数、不再重置状态
   - 只负责：获取状态 → 同步到 URL → 加载数据

4. **同步执行保存**：
   - `watch activeTabId` 是同步执行的（Vue 3 的 watch 默认是同步的）
   - 保证保存和恢复在正确的时机完成

## 四、测试用例

### 4.1 基础功能测试

**测试用例 1：Tab 标签切换**
1. 在"工单管理"中搜索"紧急"
2. 点击"会议室管理"Tab
3. 点击"工单管理"Tab
4. **预期**：搜索条件"紧急"仍然存在

**测试用例 2：服务目录切换**
1. 在"工单管理"中搜索"紧急"
2. 点击服务目录中的"会议室管理"
3. 点击服务目录中的"工单管理"
4. **预期**：搜索条件"紧急"仍然存在

**测试用例 3：混合切换**
1. 在"工单管理"中搜索"紧急"
2. 点击"会议室管理"Tab
3. 点击服务目录中的"工单管理"
4. **预期**：搜索条件"紧急"仍然存在

### 4.2 边界条件测试

**测试用例 4：新打开的 Tab**
1. 点击服务目录中的"投票记录查询"（未打开过）
2. **预期**：显示默认状态（无搜索条件，第 1 页，默认排序）

**测试用例 5：关闭后重新打开的 Tab**
1. 在"工单管理"中搜索"紧急"
2. 关闭"工单管理"Tab
3. 点击服务目录中的"工单管理"
4. **预期**：显示默认状态（搜索条件已清空）

**测试用例 6：多个 Tab 相互切换**
1. 在"工单管理"中搜索"紧急"
2. 在"会议室管理"中搜索"可用"
3. 在"投票记录查询"中搜索"进行中"
4. 依次切换回"工单管理" → "会议室管理" → "投票记录查询"
5. **预期**：每个 Tab 的搜索条件都保持正确

### 4.3 性能测试

**测试用例 7：快速切换**
1. 在两个 Tab 之间快速切换 10 次
2. **预期**：状态不丢失，无重复请求

**测试用例 8：大量 Tab**
1. 打开 10 个不同的 Tab
2. 在每个 Tab 中设置不同的搜索条件
3. 依次切换所有 Tab
4. **预期**：所有 Tab 的状态都正确保存和恢复

## 五、实施计划

### 5.1 实施步骤

1. **备份当前代码**
   - 创建新分支：`refactor/tab-state-management-v2`
   - 提交当前代码状态

2. **重构 `useWorkspaceTabs.ts`**
   - 修改 `setupTabDataWatch()`：在 `watch activeTabId` 中同步保存和恢复
   - 简化 `handleTabClick()`：移除保存逻辑
   - 移除 `saveCurrentTabState()` 函数

3. **重构 `useTableInitialization.ts`**
   - 简化 `initializeTable()`：移除 URL 判断、状态重置逻辑
   - 简化 `watch route.query`：只负责从 URL 恢复状态

4. **测试验证**
   - 运行所有测试用例
   - 修复发现的问题

5. **代码审查**
   - 审查代码质量
   - 确保符合规范

6. **合并到主分支**
   - 提交到 `refactor/new-architecture`
   - 推送到远程

### 5.2 回滚方案

如果重构失败，可以通过以下步骤回滚：

```bash
# 切换回原分支
git checkout refactor/new-architecture

# 删除重构分支
git branch -D refactor/tab-state-management-v2
```

## 六、总结

### 6.1 核心改进

1. **统一保存时机**：所有切换方式都在 `watch activeTabId` 中保存
2. **提前恢复状态**：在 `TableView.onMounted` 之前恢复状态
3. **简化初始化逻辑**：`initializeTable()` 不再处理状态恢复

### 6.2 预期效果

- ✅ Tab 切换时状态不丢失
- ✅ 服务目录切换和 Tab 标签切换行为一致
- ✅ 每个 Tab 的状态独立，不会相互污染
- ✅ 代码逻辑清晰，易于维护

### 6.3 风险评估

**风险等级：中**

- ✅ 重构涉及核心逻辑，但改动范围可控
- ✅ 有完整的测试用例覆盖
- ✅ 有回滚方案

**建议**：
- 在测试环境充分测试后再部署到生产环境
- 逐步灰度发布，观察用户反馈

