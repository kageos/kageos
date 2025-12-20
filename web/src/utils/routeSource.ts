/**
 * 路由更新来源常量
 * 统一管理所有路由更新的 source 字符串，避免硬编码
 */
export const RouteSource = {
  // 工作空间相关
  WORKSPACE_NODE_CLICK: 'workspace-node-click',
  WORKSPACE_NODE_CLICK_PACKAGE: 'workspace-node-click-package',
  WORKSPACE_ROUTING_CLEAR_LINK_TYPE: 'workspace-routing-clear-link-type',
  
  // 表格相关
  TABLE_DETAIL_OPEN: 'table-detail-open',
  TABLE_DETAIL_NAVIGATE: 'table-detail-navigate',
  TABLE_DETAIL_CLOSE: 'table-detail-close',
  TABLE_DETAIL_CLEANUP: 'table-detail-cleanup',
  TABLE_DETAIL_CLEANUP_INVALID_ID: 'table-detail-cleanup-invalid-id',
  TABLE_DETAIL_CLEANUP_NOT_FOUND: 'table-detail-cleanup-not-found',
  TABLE_DETAIL_CLEANUP_FUNCTION_CHANGE: 'table-detail-cleanup-function-change',
  TABLE_SYNC: 'table-sync',
  TABLE_LINK_CLICK: 'table-link-click',
  TABLE_CREATE_DIALOG_CLOSE: 'table-create-dialog-close',
  TABLE_ADD_DIALOG_OPEN: 'add-dialog-open',
  
  // 其他
  AGENT_SELECT: 'agent-select',
  BACK_TO_LIST: 'back-to-list',
} as const

/**
 * 路由更新来源类型
 */
export type RouteSourceType = typeof RouteSource[keyof typeof RouteSource]

