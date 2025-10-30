import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { login as loginApi, logout as logoutApi, getUserInfo, refreshToken } from '@/api/auth'
import type { UserInfo, LoginRequest } from '@/types'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const token = ref<string>(localStorage.getItem('token') || '')
  const user = ref<UserInfo | null>(null)
  const isLoading = ref(false)

  // 计算属性
  const isAuthenticated = computed(() => !!token.value)
  const userName = computed(() => user.value?.username || '')
  const userEmail = computed(() => user.value?.email || '')

  // 登录
  async function login(credentials: LoginRequest) {
    try {
      isLoading.value = true
      const response = await loginApi(credentials)

      token.value = response.token
      user.value = response.user

      // 保存token到localStorage
      localStorage.setItem('token', response.token)

      ElMessage.success('登录成功')

      // 跳转到首页
      await router.push('/')

      return response
    } catch (error) {
      console.error('登录失败:', error)
      throw error
    } finally {
      isLoading.value = false
    }
  }

  // 登出
  async function logout() {
    try {
      await logoutApi()
    } catch (error) {
      console.error('登出请求失败:', error)
    } finally {
      // 清理本地状态
      token.value = ''
      user.value = null
      localStorage.removeItem('token')

      ElMessage.success('已退出登录')

      // 跳转到登录页
      await router.push('/login')
    }
  }

  // 获取用户信息
  async function fetchUserInfo() {
    try {
      if (!token.value) return

      const userInfo = await getUserInfo()
      user.value = userInfo
      return userInfo
    } catch (error) {
      console.error('获取用户信息失败:', error)
      // 如果获取用户信息失败，可能是token过期，清理状态
      await logout()
      throw error
    }
  }

  // 刷新token
  async function refreshUserToken() {
    try {
      const response = await refreshToken()
      token.value = response.token
      localStorage.setItem('token', response.token)
      return response.token
    } catch (error) {
      console.error('刷新token失败:', error)
      // 刷新失败，清理状态
      await logout()
      throw error
    }
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
    } catch (error) {
      return false
    }
  }

  // 初始化认证状态
  async function initAuth() {
    // 不在初始化时自动获取用户信息，避免调用不存在的API
    // 只有在需要时才获取用户信息
  }

  return {
    // 状态
    token,
    user,
    isLoading,

    // 计算属性
    isAuthenticated,
    userName,
    userEmail,

    // 方法
    login,
    logout,
    fetchUserInfo,
    refreshUserToken,
    checkAuthStatus,
    initAuth
  }
})