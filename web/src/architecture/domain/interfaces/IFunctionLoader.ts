/**
 * IFunctionLoader - 函数加载器接口
 * 
 * 职责：定义函数加载的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 根据 ID 加载函数详情
 * - 根据路径加载函数详情
 * - 缓存管理
 */

/**
 * 函数详情类型（简化，实际应该从 types 导入）
 */
export interface FunctionDetail {
  id?: number
  code?: string
  name?: string
  method?: string
  router?: string
  template_type?: 'form' | 'table'
  request?: any[]
  response?: any[]
  [key: string]: any
}

/**
 * 函数加载器接口
 */
export interface IFunctionLoader {
  /**
   * 根据 ID 加载函数详情
   * @param id 函数 ID
   * @returns Promise<FunctionDetail>
   */
  loadById(id: number): Promise<FunctionDetail>

  /**
   * 根据路径加载函数详情
   * @param path 函数路径（如：/workspace/tenant/app/service/function）
   * @returns Promise<FunctionDetail>
   */
  loadByPath(path: string): Promise<FunctionDetail>

  /**
   * 获取缓存的函数详情
   * @param id 函数 ID（可选）
   * @param path 函数路径（可选）
   * @returns FunctionDetail | null
   */
  getCached(id?: number, path?: string): FunctionDetail | null

  /**
   * 清空缓存
   */
  clearCache(): void
}

