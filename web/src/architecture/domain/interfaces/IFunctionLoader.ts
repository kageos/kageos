/**
 * IFunctionLoader - 函数加载器接口
 * 
 * 职责：定义函数加载的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 根据路径加载函数详情
 * - 缓存管理
 */

import type { FieldConfig } from '@/core/types/field'

/**
 * 函数详情类型
 */
export interface FunctionDetail {
  id?: number
  app_id?: number
  tree_id?: number
  code?: string
  name?: string
  description?: string
  method?: string
  router?: string
  has_config?: boolean
  create_tables?: string
  callbacks?: string | string[]
  template_type?: 'form' | 'table' | 'chart' | string
  request?: FieldConfig[]
  response?: FieldConfig[]
  permissions?: Record<string, boolean>
  created_by?: string
  created_at?: string
  updated_at?: string
  [key: string]: any
}

/**
 * 函数加载器接口
 */
export interface IFunctionLoader {
  /**
   * 根据路径加载函数详情
   * @param path 函数路径（如：/workspace/tenant/app/service/function）
   * @param funcType 函数类型：table、form、chart（可选，默认为 table）
   * @returns Promise<FunctionDetail>
   */
  loadByPath(path: string, funcType?: string): Promise<FunctionDetail>

  /**
   * 获取缓存的函数详情
   * @param path 函数路径
   * @returns FunctionDetail | null
   */
  getCached(path: string): FunctionDetail | null

  /**
   * 清空缓存
   */
  clearCache(): void
}
