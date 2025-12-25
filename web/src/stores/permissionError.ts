/**
 * 权限错误状态管理
 * 用于存储当前权限不足的错误信息，供详情页面显示
 */
import { defineStore } from 'pinia'
import type { PermissionInfo } from '@/utils/permission'

interface PermissionErrorState {
  currentError: PermissionInfo | null
}

export const usePermissionErrorStore = defineStore('permissionError', {
  state: (): PermissionErrorState => ({
    currentError: null
  }),

  actions: {
    /**
     * 设置当前权限错误
     */
    setError(error: PermissionInfo | null) {
      this.currentError = error
    },

    /**
     * 清除权限错误
     */
    clearError() {
      this.currentError = null
    }
  }
})

