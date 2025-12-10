# Tab 和路由切换功能重构方案

## 一、核心设计原则

### 1. 单一数据源（Single Source of Truth）
- **路由是唯一的数据源**：URL 决定应用状态
- Tab 状态从路由派生，而不是独立维护
- 刷新时从路由恢复所有状态

### 2. 单向数据流
```
用户操作 → 更新路由 → 路由变化 → 更新 Tab 状态 → 更新 UI
```

### 3. 明确的数据流向
- **Tab 点击**：只更新路由，不直接更新 Tab 状态
- **路由变化**：解析路由，更新 Tab 状态（打开/激活 Tab）
- **刷新**：从路由恢复 Tab 状态，展开目录树

## 二、实现方案

### 2.1 Tab 点击处理

```typescript
// Tab 点击：只更新路由
const handleTabClick = (tab: any) => {
  if (tab.name) {
    const targetTab = tabs.value.find(t => t.id === tab.name)
    if (targetTab && targetTab.path) {
      const targetPath = `/workspace${targetTab.path.startsWith('/') ? targetTab.path : `/${targetTab.path}`}`
      router.replace({ path: targetPath, query: {} })
    }
  }
}
```

**原则**：
- 不调用 `activateTab`
- 只更新路由
- 路由变化会触发 `watch route.path`，进而更新 Tab 状态

### 2.2 路由变化处理

```typescript
// 监听路由变化，更新 Tab 状态
watch(() => route.path, async () => {
  const fullPath = extractWorkspacePath(route.path)
  
  // 解析路径，找到对应的 Tab
  const targetTab = tabs.value.find(t => {
    const tabPath = t.path?.replace(/^\//, '') || ''
    const routePath = fullPath?.replace(/^\//, '') || ''
    return tabPath === routePath
  })
  
  if (targetTab) {
    // Tab 已存在，激活它
    if (activeTabId.value !== targetTab.id) {
      applicationService.activateTab(targetTab.id)
    }
  } else {
    // Tab 不存在，从路由打开新 Tab
    await loadAppFromRoute()
  }
})
```

**原则**：
- 路由变化是唯一触发 Tab 状态更新的入口
- 如果 Tab 已存在，激活它
- 如果 Tab 不存在，从路由打开新 Tab

### 2.3 watch activeTabId

```typescript
// watch activeTabId：只处理数据保存和恢复，不处理路由
watch(() => stateManager.getState().activeTabId, async (newId, oldId) => {
  // 1. 保存旧 Tab 数据
  // 2. 恢复新 Tab 数据
  // 注意：不更新路由，路由更新由 handleTabClick 和 watch route.path 处理
})
```

**原则**：
- 只处理 Tab 数据的保存和恢复
- 不处理路由更新
- 避免与路由更新逻辑冲突

### 2.4 刷新时状态恢复

```typescript
// 刷新时：从路由恢复 Tab 状态
onMounted(async () => {
  await loadAppList()
  await loadAppFromRoute() // 从路由恢复 Tab 状态
})

// loadAppFromRoute：从路由解析并打开/激活 Tab
const loadAppFromRoute = async () => {
  const fullPath = extractWorkspacePath(route.path)
  // 1. 解析路径，找到对应的节点
  // 2. 检查 Tab 是否存在
  // 3. 如果存在，激活它；如果不存在，打开新 Tab
  // 4. 展开目录树到当前路径
}
```

**原则**：
- 刷新时从路由恢复所有状态
- 确保 Tab 状态和路由一致
- 自动展开目录树

## 三、关键点

### 3.1 避免循环更新
- Tab 点击 → 更新路由 → watch route.path → 激活 Tab
- 不会循环，因为路由更新是同步的，watch 只触发一次

### 3.2 避免时序问题
- 所有路由更新都通过 `router.replace`
- 所有 Tab 状态更新都通过 `activateTab`
- 两者分离，避免时序冲突

### 3.3 刷新时状态恢复
- 从路由解析路径
- 检查 Tab 是否存在（从 localStorage 恢复）
- 如果存在，激活它；如果不存在，打开新 Tab
- 展开目录树到当前路径

## 四、实现步骤

1. **简化 handleTabClick**：只更新路由
2. **重写 watch route.path**：从路由更新 Tab 状态
3. **简化 watch activeTabId**：只处理数据保存和恢复
4. **移除事件监听器中的路由更新逻辑**
5. **优化 loadAppFromRoute**：确保刷新时正确恢复状态

## 五、优势

1. **逻辑清晰**：单一数据源，单向数据流
2. **易于维护**：路由更新逻辑集中在一个地方
3. **避免时序问题**：路由和 Tab 状态更新分离
4. **刷新可靠**：从路由恢复状态，保证一致性

