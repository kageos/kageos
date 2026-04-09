import { useUserInfoStore } from '@/stores/userInfo'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { FunctionDetail } from '../../domain/types'

export function useTableReferencePreload() {
  const userInfoStore = useUserInfoStore()
  const departmentInfoStore = useDepartmentInfoStore()

  const preloadUserInfoFromSearchForm = async (
    functionDetail: FunctionDetail,
    searchFormData: Record<string, any>
  ): Promise<void> => {
    try {
      const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
      const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
      const userFields = [
        ...requestFields.filter(f => f.widget?.type === 'user'),
        ...responseFields.filter(f => f.widget?.type === 'user')
      ]

      if (userFields.length === 0) {
        return
      }

      const usernames = new Set<string>()
      userFields.forEach(field => {
        const value = searchFormData[field.code]
        if (!value) {
          return
        }

        if (Array.isArray(value)) {
          value.forEach(v => {
            if (v) usernames.add(String(v))
          })
          return
        }

        usernames.add(String(value))
      })

      if (usernames.size === 0) {
        return
      }

      await userInfoStore.batchGetUserInfo([...usernames])
    } catch {
      // 预加载失败不阻断表格初始化
    }
  }

  const preloadDepartmentInfoFromSearchForm = async (
    functionDetail: FunctionDetail,
    searchFormData: Record<string, any>
  ): Promise<void> => {
    try {
      const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
      const responseFields = Array.isArray(functionDetail.response) ? functionDetail.response : []
      const departmentFields = [
        ...requestFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments'),
        ...responseFields.filter(f => f.widget?.type === 'department' || f.widget?.type === 'departments')
      ]

      if (departmentFields.length === 0) {
        return
      }

      const paths = new Set<string>()
      departmentFields.forEach(field => {
        const value = searchFormData[field.code]
        if (!value) {
          return
        }

        if (Array.isArray(value)) {
          value.forEach(v => {
            if (v) paths.add(String(v))
          })
          return
        }

        if (typeof value === 'string' && value.includes(',')) {
          value.split(',').forEach(v => {
            const trimmed = v.trim()
            if (trimmed) paths.add(trimmed)
          })
          return
        }

        paths.add(String(value))
      })

      if (paths.size === 0) {
        return
      }

      await departmentInfoStore.batchGetDepartmentInfo([...paths])
    } catch {
      // 预加载失败不阻断表格初始化
    }
  }

  return {
    preloadUserInfoFromSearchForm,
    preloadDepartmentInfoFromSearchForm,
  }
}
