/**
 * 主题状态管理
 */
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import type { ThemeMode, ThemeConfig } from '@/types/theme'
import { THEME_PRESETS, DEFAULT_THEME } from '@/types/theme'

const THEME_STORAGE_KEY = 'ai-agent-os-theme'

export const useThemeStore = defineStore('theme', () => {
  // 当前主题配置
  const currentTheme = ref<ThemeConfig>(DEFAULT_THEME)
  
  /**
   * 初始化主题
   */
  const initTheme = () => {
    // 从 localStorage 读取保存的主题
    const savedThemeName = localStorage.getItem(THEME_STORAGE_KEY)
    
    if (savedThemeName) {
      const savedTheme = THEME_PRESETS.find(t => t.name === savedThemeName)
      if (savedTheme) {
        currentTheme.value = savedTheme
      }
    } else {
      // 如果没有保存的主题，根据系统偏好设置默认主题
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      const defaultTheme = prefersDark 
        ? THEME_PRESETS.find(t => t.mode === 'dark') || DEFAULT_THEME
        : DEFAULT_THEME
      currentTheme.value = defaultTheme
    }
    
    // 应用主题
    applyTheme(currentTheme.value)
    
    console.log('[ThemeStore] 主题初始化完成:', currentTheme.value)
  }
  
  /**
   * 应用主题
   */
  const applyTheme = (theme: ThemeConfig) => {
    // 移除所有主题类
    document.documentElement.classList.remove('light', 'dark')
    
    // 添加新主题类
    document.documentElement.classList.add(theme.mode)
    
    // 保存到 localStorage
    localStorage.setItem(THEME_STORAGE_KEY, theme.name)
    
    console.log('[ThemeStore] 应用主题:', theme)
  }
  
  /**
   * 切换主题
   */
  const setTheme = (theme: ThemeConfig) => {
    currentTheme.value = theme
    applyTheme(theme)
  }
  
  /**
   * 切换到指定模式
   */
  const setThemeMode = (mode: ThemeMode) => {
    const theme = THEME_PRESETS.find(t => t.mode === mode)
    if (theme) {
      setTheme(theme)
    }
  }
  
  /**
   * 切换深色/浅色模式
   */
  const toggleTheme = () => {
    const newMode: ThemeMode = currentTheme.value.mode === 'light' ? 'dark' : 'light'
    setThemeMode(newMode)
  }
  
  /**
   * 获取所有可用主题
   */
  const getAvailableThemes = () => {
    return THEME_PRESETS
  }
  
  // 监听系统主题变化（可选）
  watch(
    () => currentTheme.value,
    (newTheme) => {
      console.log('[ThemeStore] 主题已更改:', newTheme)
    }
  )
  
  return {
    currentTheme,
    initTheme,
    setTheme,
    setThemeMode,
    toggleTheme,
    getAvailableThemes
  }
})

