/**
 * ErrorHandler - 统一错误处理
 * 
 * 核心目标：
 * 1. 统一错误日志格式
 * 2. 统一错误消息提示
 * 3. 统一降级处理
 * 4. 便于错误追踪和统计
 */

import { ElMessage } from 'element-plus'

/**
 * 错误处理选项
 */
export interface ErrorHandleOptions {
  /** 是否显示错误消息（默认 false） */
  showMessage?: boolean
  
  /** 错误消息文本（可选，默认自动提取） */
  message?: string
  
  /** 降级返回值（默认 undefined） */
  fallbackValue?: any
  
  /** 是否重新抛出错误（默认 false） */
  rethrow?: boolean
  
  /** 日志级别（默认 'error'） */
  logLevel?: 'error' | 'warn' | 'info'
}

/**
 * ErrorHandler - 统一错误处理类
 */
export class ErrorHandler {
  /**
   * 处理 Widget 相关错误
   * 
   * @param context - 错误上下文（如：'TableWidget.renderCellByWidget'）
   * @param error - 错误对象
   * @param options - 错误处理选项
   * @returns 降级返回值
   * 
   * @example
   * ```typescript
   * try {
   *   const widget = createWidget()
   * } catch (error) {
   *   return ErrorHandler.handleWidgetError('TableWidget.createFormWidgets', error, {
   *     showMessage: false,
   *     fallbackValue: null
   *   })
   * }
   * ```
   */
  static handleWidgetError(
    context: string,
    error: any,
    options: ErrorHandleOptions = {}
  ): any {
    const {
      showMessage = false,
      message,
      fallbackValue = undefined,
      rethrow = false,
      logLevel = 'error'
    } = options
    
    // 统一日志格式
    const logMessage = `[${context}] 错误`
    this.log(logLevel, logMessage, error)
    
    // 显示用户友好的错误消息
    if (showMessage) {
      const userMessage = message || this.extractErrorMessage(error)
      ElMessage.error({
        message: userMessage,
        duration: 3000,
        showClose: true
      })
    }
    
    // 重新抛出（用于需要上层捕获的场景）
    if (rethrow) {
      throw error
    }
    
    // 返回降级值
    return fallbackValue
  }
  
  /**
   * 处理 API 请求错误
   * 
   * @example
   * ```typescript
   * try {
   *   const response = await selectFuzzy(params)
   * } catch (error) {
   *   ErrorHandler.handleApiError('SelectWidget.handleSearch', error, {
   *     showMessage: true,
   *     message: '搜索失败，请重试'
   *   })
   * }
   * ```
   */
  static handleApiError(
    context: string,
    error: any,
    options: ErrorHandleOptions = {}
  ): any {
    // API 错误默认显示消息
    const finalOptions: ErrorHandleOptions = {
      showMessage: true,
      logLevel: 'error',
      ...options
    }
    
    return this.handleWidgetError(context, error, finalOptions)
  }
  
  /**
   * 处理验证错误
   * 
   * @example
   * ```typescript
   * if (!isValid) {
   *   ErrorHandler.handleValidationError('FormRenderer.handleSubmit', {
   *     field: 'price',
   *     message: '价格必须大于0'
   *   })
   * }
   * ```
   */
  static handleValidationError(
    context: string,
    validationError: { field: string; message: string },
    options: ErrorHandleOptions = {}
  ): void {
    const message = `${validationError.field}: ${validationError.message}`
    
    ElMessage.warning({
      message: message,
      duration: 3000,
      showClose: true
    })
    
    console.warn(`[${context}] 验证失败:`, validationError)
  }
  
  /**
   * 从错误对象中提取用户友好的错误消息
   */
  private static extractErrorMessage(error: any): string {
    // 优先级：
    // 1. response.data.msg (后端返回)
    // 2. response.data.message (后端返回)
    // 3. error.message (JS 错误)
    // 4. 默认消息
    
    // 🔥 统一使用 msg 字段
    if (error?.response?.data?.msg) {
      return error.response.data.msg
    }
    
    if (error?.message) {
      return error.message
    }
    
    return '操作失败，请重试'
  }
  
  /**
   * 统一日志输出
   */
  private static log(level: 'error' | 'warn' | 'info', message: string, error?: any): void {
    const timestamp = new Date().toISOString()
    const prefix = `[${timestamp}]`
    
    switch (level) {
      case 'error':
        console.error(prefix, message, error)
        break
      case 'warn':
        console.warn(prefix, message, error)
        break
      case 'info':
        console.info(prefix, message, error)
        break
    }
  }
  
  /**
   * 创建错误边界（用于组件级别的错误捕获）
   * 
   * @example
   * ```typescript
   * const safeRender = ErrorHandler.createErrorBoundary(
   *   'TableWidget.render',
   *   () => this.renderTable(),
   *   { fallbackValue: h('div', '渲染失败') }
   * )
   * return safeRender()
   * ```
   */
  static createErrorBoundary<T>(
    context: string,
    fn: () => T,
    options: ErrorHandleOptions = {}
  ): () => T | any {
    return () => {
      try {
        return fn()
      } catch (error) {
        return this.handleWidgetError(context, error, options)
      }
    }
  }
}

