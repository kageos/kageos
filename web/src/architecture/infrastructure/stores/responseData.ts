/**
 * ResponseData Store - 响应数据管理
 * 🔥 统一架构组件
 * 
 * 功能：
 * - 存储响应数据
 * - 提供渲染触发器（用于触发响应参数区域的重新渲染）
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useResponseDataStore = defineStore('responseData', () => {
  // 响应数据
  const data = ref<any>(null)
  
  // 渲染触发器（用于触发响应参数区域的重新渲染）
  const renderTrigger = ref(0)
  
  /**
   * 设置响应数据
   */
  function setData(newData: any): void {
    data.value = newData
    renderTrigger.value++
  }
  
  /**
   * 清空响应数据
   */
  function clear(): void {
    data.value = null
    renderTrigger.value = 0
  }
  
  return {
    data,
    renderTrigger,
    setData,
    clear
  }
})
