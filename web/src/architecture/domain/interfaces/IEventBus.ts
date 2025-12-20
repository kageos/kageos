/**
 * IEventBus - 事件总线接口
 * 
 * 职责：定义事件总线的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 组件间通信
 * - 跨层级通信
 * - 解耦组件依赖
 */

/**
 * 事件总线接口
 */
export interface IEventBus {
  /**
   * 触发事件
   * @param event 事件名称
   * @param payload 事件数据（可选）
   */
  emit(event: string, payload?: any): void

  /**
   * 监听事件
   * @param event 事件名称
   * @param handler 事件处理函数
   * @returns 取消监听的函数
   */
  on(event: string, handler: (payload?: any) => void): () => void

  /**
   * 取消监听事件
   * @param event 事件名称
   * @param handler 事件处理函数
   */
  off(event: string, handler: (payload?: any) => void): void

  /**
   * 监听事件（仅触发一次）
   * @param event 事件名称
   * @param handler 事件处理函数
   */
  once(event: string, handler: (payload?: any) => void): void
}

/**
 * 事件类型定义（使用 camelCase，更易读）
 * 命名规范：模块名:动作名，例如 workspace:node-clicked
 */
export enum WorkspaceEvent {
  nodeClicked = 'workspace:node-clicked',           // 节点点击
  appSwitched = 'workspace:app-switched',           // 应用切换
  serviceTreeLoaded = 'workspace:service-tree-loaded', // 服务树加载完成
  functionLoaded = 'workspace:function-loaded',     // 函数加载完成
  // 🔥 Tab 功能已删除，以下事件已废弃
  // tabSwitched = 'workspace:tab-switched',
  // tabSwitchedComplete = 'workspace:tab-switched-complete'
}

export enum FormEvent {
  initialized = 'form:initialized',                 // 表单初始化完成
  fieldValueUpdated = 'form:field-value-updated',   // 字段值更新
  validated = 'form:validated',                     // 表单验证完成
  submitted = 'form:submitted',                     // 表单提交
  responseReceived = 'form:response-received'       // 响应数据接收
}

export enum TableEvent {
  dataLoaded = 'table:data-loaded',                 // 表格数据加载完成
  searchChanged = 'table:search-changed',           // 搜索条件变化
  sortChanged = 'table:sort-changed',              // 排序变化
  pageChanged = 'table:page-changed',              // 分页变化
  rowAdded = 'table:row-added',                     // 行新增
  rowUpdated = 'table:row-updated',                // 行更新
  rowDeleted = 'table:row-deleted'                 // 行删除
}

export enum RouteEvent {
  // 路由更新请求事件
  updateRequested = 'route:update-requested',        // 请求更新路由
  updateCompleted = 'route:update-completed',        // 路由更新完成
  
  // 路由变化事件（由路由管理器监听 Vue Router 变化后发出）
  pathChanged = 'route:path-changed',                // 路径变化
  queryChanged = 'route:query-changed',              // 查询参数变化
  routeChanged = 'route:route-changed'               // 路由变化（path + query）
}

