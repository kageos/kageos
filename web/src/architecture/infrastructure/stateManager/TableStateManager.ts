/**
 * TableStateManager - 表格状态管理实现
 * 
 * 职责：基于响应式对象实现表格状态管理
 * 
 * 特点：
 * - 使用 StateManagerImpl 作为基础
 * - 提供表格特定的状态管理
 */

import { StateManagerImpl } from './StateManagerImpl'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { TableState } from '../../domain/types'

/**
 * 表格状态管理实现
 */
export class TableStateManager extends StateManagerImpl<TableState> implements IStateManager<TableState> {
  constructor(initialState?: Partial<TableState>) {
    const defaultState: TableState = {
      data: [],
      loading: false,
      searchParams: {},
      searchForm: {}, // 🔥 新增：搜索表单数据
      sortParams: null,
      sorts: [], // 🔥 新增：排序列表
      hasManualSort: false, // 🔥 新增：是否手动排序
      pagination: {
        currentPage: 1,
        pageSize: 10,
        total: 0
      }
    }

    super({
      ...defaultState,
      ...initialState
    } as TableState)

    // 注意：这里不需要设置 watch，因为 state 已经是响应式的
    // 如果需要响应式，可以在使用时通过 computed 或 watch 实现
  }

  /**
   * 获取表格数据
   */
  getData() {
    return this.getState().data
  }

  /**
   * 获取加载状态
   */
  isLoading() {
    return this.getState().loading
  }

  /**
   * 获取分页信息
   */
  getPagination() {
    return this.getState().pagination
  }
}
