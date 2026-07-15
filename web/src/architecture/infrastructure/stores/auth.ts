import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { login as loginApi, logout as logoutApi, getUserInfo, refreshToken as refreshTokenApi } from '@/architecture/infrastructure/api/auth'
import { updateUser as updateUserApi, type UpdateUserReq } from '@/architecture/infrastructure/api/user'
import type { UserInfo, LoginRequest } from '@/architecture/domain/types'
import { translate } from '@/architecture/shared/i18n'
import { getCurrentRoutePath, navigateTo } from '@/architecture/shared/routing/navigation'

function normalizeOAuthRedirect(redirectAfter: string | undefined, username: string) {
  const fallback = `/workspace/${username || 'me'}`
  const raw = (redirectAfter || '').trim()
  if (!raw || raw === '/workspace' || raw === '/login') {
    return fallback
  }
  if (!raw.startsWith('/') || raw.startsWith('//')) {
    return fallback
  }
  return raw
}

export const useAuthStore = defineStore('auth', () => {
  interface LogoutOptions {
    callApi?: boolean
    notify?: boolean
    redirectToLogin?: boolean
  }

  // 状态
  const token = ref<string>(localStorage.getItem('token') || '')
  const refreshToken = ref<string>(localStorage.getItem('refresh_token') || '')
  
  // 从 localStorage 读取用户信息
  const savedUserStr = localStorage.getItem('user')
  const savedUser = savedUserStr ? JSON.parse(savedUserStr) : null
  const user = ref<UserInfo | null>(savedUser)
  
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
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('refresh_token')
  }

  // 登录
  async function login(credentials: LoginRequest) {
    try {
      isLoading.value = true
      const response = await loginApi(credentials)

      token.value = response.token
      user.value = response.user
      if (response.refresh_token) {
        refreshToken.value = response.refresh_token
        localStorage.setItem('refresh_token', response.refresh_token)
      }

      // 保存token和用户信息到localStorage
      localStorage.setItem('token', response.token)
      localStorage.setItem('user', JSON.stringify(response.user))

      ElMessage.success(translate('auth.loginSuccess'))

      // 跳转到工作空间（会弹出选择工作空间）
      const username = response.user?.username || 'me'
      await navigateTo(`/workspace/${username}`)

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
      localStorage.setItem('token', accessToken)
      if (refreshTokenValue) {
        localStorage.setItem('refresh_token', refreshTokenValue)
      }

      const userInfo = await fetchUserInfo()
      ElMessage.success(translate('auth.loginSuccess'))

      const username = userInfo?.username || userName.value || 'me'
      const target = normalizeOAuthRedirect(redirectAfter, username)
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
      localStorage.setItem('user', JSON.stringify(userInfo))
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
    return refreshToken.value || localStorage.getItem('refresh_token') || ''
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
        localStorage.setItem('refresh_token', response.refresh_token)
      }
      localStorage.setItem('token', response.token)
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
      localStorage.setItem('user', JSON.stringify(response.user))
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
