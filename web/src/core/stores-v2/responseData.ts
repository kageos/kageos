/**
 * ResponseData Store - 响应数据管理（函数粒度缓存）
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 存储响应数据
 * - 提供渲染触发器（用于触发响应参数区域的重新渲染）
 * - 🔥 支持函数粒度缓存，每个函数独立存储响应数据
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

/**
 * 获取函数唯一标识
 * 优先使用 id，如果没有则使用 router
 */
function getFunctionKey(functionId?: number | string, functionRouter?: string): string {
  if (functionId && functionId !== 0) {
    return `function_${functionId}`
  }
  if (functionRouter) {
    return `router_${functionRouter}`
  }
  return 'default'
}

export const useResponseDataStore = defineStore('responseData-v2', () => {
  // 🔥 函数粒度的响应数据缓存：functionKey -> { data, renderTrigger }
  const functionResponseCache = new Map<string, { data: any, renderTrigger: number }>()
  
  // 🔥 当前激活的函数标识（用于向后兼容，如果没有传入 functionKey 则使用这个）
  const currentFunctionKey = ref<string>('default')
  
  /**
   * 获取指定函数的响应数据
   */
  function getFunctionResponse(functionKey?: string): { data: any, renderTrigger: number } {
    const key = functionKey || currentFunctionKey.value
    if (!functionResponseCache.has(key)) {
      functionResponseCache.set(key, { data: null, renderTrigger: 0 })
    }
    return functionResponseCache.get(key)!
  }
  
  /**
   * 设置当前函数标识
   */
  function setCurrentFunction(functionId?: number | string, functionRouter?: string): void {
    currentFunctionKey.value = getFunctionKey(functionId, functionRouter)
  }
  
  /**
   * 设置响应数据
   */
  function setData(newData: any, functionId?: number | string, functionRouter?: string): void {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const response = getFunctionResponse(functionKey)
    response.data = newData
    response.renderTrigger++
  }
  
  /**
   * 获取响应数据
   */
  function getData(functionId?: number | string, functionRouter?: string): any {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const response = getFunctionResponse(functionKey)
    return response.data
  }
  
  /**
   * 获取渲染触发器
   */
  function getRenderTrigger(functionId?: number | string, functionRouter?: string): number {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const response = getFunctionResponse(functionKey)
    return response.renderTrigger
  }
  
  /**
   * 清空指定函数的响应数据
   */
  function clear(functionId?: number | string, functionRouter?: string): void {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const response = getFunctionResponse(functionKey)
    response.data = null
    response.renderTrigger = 0
  }
  
  /**
   * 清空所有函数的响应数据
   */
  function clearAll(): void {
    functionResponseCache.clear()
    currentFunctionKey.value = 'default'
  }
  
  /**
   * 获取当前函数的响应数据（用于向后兼容）
   */
  const data = computed(() => {
    return getFunctionResponse(currentFunctionKey.value).data
  })
  
  /**
   * 获取当前函数的渲染触发器（用于向后兼容）
   */
  const renderTrigger = computed(() => {
    return getFunctionResponse(currentFunctionKey.value).renderTrigger
  })
  
  return {
    data,
    renderTrigger,
    setCurrentFunction,
    setData,
    getData,
    getRenderTrigger,
    clear,
    clearAll
  }
})

