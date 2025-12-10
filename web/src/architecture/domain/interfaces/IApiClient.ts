/**
 * IApiClient - API 客户端接口
 * 
 * 职责：定义 API 调用的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - HTTP 请求
 * - 函数执行
 * - 数据获取
 */

/**
 * API 客户端接口
 */
export interface IApiClient {
  /**
   * GET 请求
   * @param url 请求 URL
   * @param params 查询参数（可选）
   * @returns Promise<T>
   */
  get<T>(url: string, params?: any): Promise<T>

  /**
   * POST 请求
   * @param url 请求 URL
   * @param data 请求体数据（可选）
   * @returns Promise<T>
   */
  post<T>(url: string, data?: any): Promise<T>

  /**
   * PUT 请求
   * @param url 请求 URL
   * @param data 请求体数据（可选）
   * @returns Promise<T>
   */
  put<T>(url: string, data?: any): Promise<T>

  /**
   * DELETE 请求
   * @param url 请求 URL
   * @returns Promise<T>
   */
  delete<T>(url: string): Promise<T>
}

