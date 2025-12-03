/**
 * TabStateManager - Tab 状态管理器
 * 
 * 职责：
 * - 管理每个 Tab 的路由状态（path + query）
 * - 支持 Tab 切换时保存和恢复路由状态
 * - 支持 Tab 关闭时清理状态
 */

export interface TabRouteState {
  path: string
  query: Record<string, any>
}

export class TabStateManager {
  private tabRouteStates = new Map<string, TabRouteState>()
  
  /**
   * 保存 Tab 的路由状态
   */
  saveTabRouteState(tabId: string, routeState: TabRouteState): void {
    this.tabRouteStates.set(tabId, {
      path: routeState.path,
      query: JSON.parse(JSON.stringify(routeState.query))  // 深拷贝
    })
  }
  
  /**
   * 获取 Tab 的路由状态
   */
  getTabRouteState(tabId: string): TabRouteState | null {
    const state = this.tabRouteStates.get(tabId)
    if (!state) return null
    
    return {
      path: state.path,
      query: JSON.parse(JSON.stringify(state.query))  // 深拷贝
    }
  }
  
  /**
   * 删除 Tab 的路由状态（Tab 关闭时）
   */
  deleteTabRouteState(tabId: string): void {
    this.tabRouteStates.delete(tabId)
  }
  
  /**
   * 清空所有 Tab 的路由状态
   */
  clearAll(): void {
    this.tabRouteStates.clear()
  }
  
  /**
   * 获取所有 Tab 的路由状态（用于调试）
   */
  getAllTabRouteStates(): Map<string, TabRouteState> {
    return new Map(this.tabRouteStates)
  }
  
  /**
   * 检查 Tab 是否有保存的路由状态
   */
  hasTabRouteState(tabId: string): boolean {
    return this.tabRouteStates.has(tabId)
  }
}

