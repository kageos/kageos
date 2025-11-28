/**
 * ApiClientImpl - API 客户端实现
 * 
 * 职责：实现 IApiClient 接口，提供 HTTP 请求功能
 * 
 * 特点：
 * - 基于现有的 request 工具实现
 * - 支持 GET、POST、PUT、DELETE 方法
 * - 可以轻松替换为其他实现（如 WebSocket API 客户端）
 */

import { get, post, put, del } from '@/utils/request'
import type { IApiClient } from '../../domain/interfaces/IApiClient'

/**
 * API 客户端实现（基于 axios）
 */
export class ApiClientImpl implements IApiClient {
  /**
   * GET 请求
   */
  async get<T>(url: string, params?: any): Promise<T> {
    return get<T>(url, params)
  }

  /**
   * POST 请求
   */
  async post<T>(url: string, data?: any): Promise<T> {
    return post<T>(url, data)
  }

  /**
   * PUT 请求
   */
  async put<T>(url: string, data?: any): Promise<T> {
    return put<T>(url, data)
  }

  /**
   * DELETE 请求
   */
  async delete<T>(url: string): Promise<T> {
    return del<T>(url)
  }
}

