import { computed, onMounted, watch, type Ref } from 'vue'
import { Logger } from '@/architecture/runtime/utils/logger'
import type { FileItem } from '../filesWidgetTypes'

interface UserInfoStoreLike {
  userInfoCache: unknown
  getUserInfo: (username: string) => any
  batchGetUserInfo: (usernames: string[]) => Promise<any>
}

interface UseFilesUploadUsersOptions {
  mode: () => string
  currentFiles: Ref<FileItem[]>
  userInfoStore: UserInfoStoreLike
}

export function useFilesUploadUsers(options: UseFilesUploadUsersOptions) {
  const allUploadUsers = computed(() => {
    const users = new Set<string>()
    options.currentFiles.value.forEach((file: FileItem) => {
      if (file.upload_user) {
        users.add(file.upload_user)
      }
    })
    return Array.from(users)
  })

  function getFileUploadUserInfo(file: FileItem) {
    if (!file.upload_user) {
      return null
    }

    try {
      const cache = options.userInfoStore.userInfoCache as any
      const cacheMap = cache?.value || cache
      if (cacheMap instanceof Map) {
        const cachedUser = cacheMap.get(file.upload_user)
        if (cachedUser) {
          return cachedUser
        }
      }
    } catch (error) {
      Logger.warn('FilesWidget', '获取用户信息失败', error)
    }

    return null
  }

  watch(
    () => allUploadUsers.value,
    (usernames: string[]) => {
      if (usernames.length > 0 && options.mode() === 'detail') {
        options.userInfoStore.batchGetUserInfo(usernames).catch((error: any) => {
          Logger.error('[FilesWidget] 加载上传用户信息失败', error)
        })
      }
    },
    { immediate: true }
  )

  onMounted(() => {
    if (allUploadUsers.value.length > 0 && options.mode() === 'detail') {
      options.userInfoStore.batchGetUserInfo(allUploadUsers.value).catch((error: any) => {
        Logger.error('[FilesWidget] 加载上传用户信息失败', error)
      })
    }
  })

  return {
    getFileUploadUserInfo,
  }
}
