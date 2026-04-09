/**
 * IFunctionLoader - 函数加载器接口
 * 
 * 职责：定义函数加载的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 根据路径加载函数详情
 * - 缓存管理
 */

import type { FunctionDetail as DomainFunctionDetail } from '../types'

export type { FunctionDetail } from '../types'

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
  loadByPath(path: string, funcType?: string): Promise<DomainFunctionDetail>

  /**
   * 获取缓存的函数详情
   * @param path 函数路径
   * @returns FunctionDetail | null
   */
  getCached(path: string): DomainFunctionDetail | null

  /**
   * 清空缓存
   */
  clearCache(): void
}
