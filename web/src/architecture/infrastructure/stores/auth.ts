import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { login as loginApi, logout as logoutApi, getUserInfo, refreshToken as refreshTokenApi } from '@/architecture/infrastructure/api/auth'
import { updateUser as updateUserApi, type UpdateUserReq } from '@/architecture/infrastructure/api/user'
import type { UserInfo, LoginRequest } from '@/architecture/domain/types'
import { translate } from '@/architecture/shared/i18n'
import { getCurrentRoutePath, navigateTo } from '@/architecture/shared/routing/navigation'

function normalizeOAuthRedirect(redirectAfter: string | undefined) {
  const fallback = '/workspace'
  const raw = (redirectAfter || '').trim()
  if (!raw || raw === '/workspace' || raw === '/login') {
    return fallback
  }
  if (!raw.startsWith('/') || raw.startsWith('//')) {
    return fallback
  }
  return raw
}

function getStoredValue(key: string): string {
  try {
    return localStorage.getItem(key) || ''
  } catch {
    return ''
  }
}

function setStoredValue(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Authentication must keep working when storage is unavailable or full.
  }
}

function removeStoredValue(key: string) {
  try {
    localStorage.removeItem(key)
  } catch {
    // In-memory state is still cleared when storage is unavailable.
  }
}

function getStoredUser(): UserInfo | null {
  const raw = getStoredValue('user')
  if (!raw) return null

  try {
    const value: unknown = JSON.parse(raw)
    if (
      typeof value === 'object'
      && value !== null
      && !Array.isArray(value)
      && typeof (value as Partial<UserInfo>).id === 'number'
      && typeof (value as Partial<UserInfo>).username === 'string'
      && (value as Partial<UserInfo>).username!.trim() !== ''
    ) {
      return value as UserInfo
    }
  } catch {
    // Remove corrupt data so subsequent app starts do not repeat the failure.
  }

  removeStoredValue('user')
  return null
}

export const useAuthStore = defineStore('auth', () => {
  interface LoginOptions {
    notify?: boolean
  }

  interface LogoutOptions {
    callApi?: boolean
    notify?: boolean
    redirectToLogin?: boolean
  }

  // 状态
  const token = ref<string>(getStoredValue('token'))
  const refreshToken = ref<string>(getStoredValue('refresh_token'))
  const user = ref<UserInfo | null>(getStoredUser())
  
  const isLoading = ref(false)
  let refreshingTokenPromise: Promise<string> | null = null

  // 计算属性
  const isAuthenticated = computed(() => !!token.value)
  const userName = computed(() => user.value?.username || '')
  const userEmail = computed(() => user.value?.email || '')

  function clearAuthState() {
    token.value = ''
    user.value = null
    refreshToken.value = ''
    removeStoredValue('token')
    removeStoredValue('user')
    removeStoredValue('refresh_token')
  }

  // 登录
  async function login(credentials: LoginRequest, options: LoginOptions = {}) {
    const { notify = true } = options
    try {
      isLoading.value = true
      const response = await loginApi(credentials)

      token.value = response.token
      user.value = response.user
      if (response.refresh_token) {
        refreshToken.value = response.refresh_token
        setStoredValue('refresh_token', response.refresh_token)
      }

      // 保存token和用户信息到localStorage
      setStoredValue('token', response.token)
      setStoredValue('user', JSON.stringify(response.user))

      if (notify) {
        ElMessage.success(translate('auth.loginSuccess'))
      }

      // 进入默认空间准备页。
      await navigateTo('/workspace')

      return response
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    } finally {
      isLoading.value = false
    }
  }

  async function completeOAuthLogin(accessToken: string, refreshTokenValue: string, redirectAfter?: string) {
    try {
      isLoading.value = true
      token.value = accessToken
      refreshToken.value = refreshTokenValue
      setStoredValue('token', accessToken)
      if (refreshTokenValue) {
        setStoredValue('refresh_token', refreshTokenValue)
      }

      await fetchUserInfo()
      ElMessage.success(translate('auth.loginSuccess'))

      const target = normalizeOAuthRedirect(redirectAfter)
      await navigateTo(target)
    } catch (error) {
      clearAuthState()
      throw error
    } finally {
      isLoading.value = false
    }
  }

  // 登出
  async function logout(options: LogoutOptions = {}) {
    const {
      callApi = true,
      notify = true,
      redirectToLogin = true,
    } = options

    try {
      if (callApi) {
        await logoutApi(token.value)
      }
    } catch (error) {
      console.error('Logout request failed:', error)
    } finally {
      clearAuthState()

      if (notify) {
        ElMessage.success(translate('auth.logoutSuccess'))
      }

      if (redirectToLogin && getCurrentRoutePath() !== '/login') {
        await navigateTo('/login')
      }
    }
  }

  // 获取用户信息
  async function fetchUserInfo() {
    try {
      if (!token.value) return

      const userInfo = await getUserInfo()
      user.value = userInfo
      // 保存用户信息到localStorage
      setStoredValue('user', JSON.stringify(userInfo))
      return userInfo
    } catch (error) {
      console.error('Failed to fetch user info:', error)
      // 如果获取用户信息失败，可能是token过期，清理状态
      await logout()
      throw error
    }
  }

  // 刷新token（无感刷新用：只负责刷新并保存，失败时 throw，不主动 logout）
  function getRefreshTokenValue(): string {
    return refreshToken.value || getStoredValue('refresh_token')
  }

  async function refreshUserToken(): Promise<string> {
    if (refreshingTokenPromise) {
      return refreshingTokenPromise
    }

    refreshingTokenPromise = (async () => {
      const rt = getRefreshTokenValue()
      if (!rt) {
        throw new Error('No refresh token')
      }

      const response = await refreshTokenApi(rt)
      token.value = response.token
      if (response.refresh_token) {
        refreshToken.value = response.refresh_token
        setStoredValue('refresh_token', response.refresh_token)
      }
      setStoredValue('token', response.token)
      return response.token
    })().finally(() => {
      refreshingTokenPromise = null
    })

    return refreshingTokenPromise
  }

  // 检查登录状态
  async function checkAuthStatus() {
    if (!token.value) return false

    try {
      // 如果有token但没有用户信息，尝试获取用户信息
      if (!user.value) {
        await fetchUserInfo()
      }
      return true
    } catch {
      return false
    }
  }

  // 初始化认证状态
  async function initAuth() {
    // 不在初始化时自动获取用户信息，避免调用不存在的API
    // 只有在需要时才获取用户信息
  }

  // 更新用户信息
  async function updateUser(data: UpdateUserReq) {
    try {
      const response = await updateUserApi(data)
      user.value = response.user
      // 更新 localStorage
      setStoredValue('user', JSON.stringify(response.user))
      ElMessage.success(translate('common.updateSuccess'))
      return response.user
    } catch (error) {
      console.error('Failed to update user info:', error)
      ElMessage.error(translate('common.updateFailed'))
      throw error
    }
  }

  return {
    // 状态
    token,
    refreshToken,
    user,
    isLoading,

    // 计算属性
    isAuthenticated,
    userName,
    userEmail,

    // 方法
    login,
    completeOAuthLogin,
    logout,
    clearAuthState,
    fetchUserInfo,
    refreshUserToken,
    checkAuthStatus,
    initAuth,
    updateUser
  }
})
