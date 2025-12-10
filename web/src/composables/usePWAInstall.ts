/**
 * usePWAInstall - PWA 安装功能
 * 
 * 功能：
 * - 监听 beforeinstallprompt 事件
 * - 提供安装提示和安装方法
 * - 检测是否已安装
 */

import { ref, onMounted, onUnmounted } from 'vue'
import type { BeforeInstallPromptEvent } from '@/types/pwa'

/**
 * PWA 安装组合式函数
 */
export function usePWAInstall() {
  // 安装提示事件
  const deferredPrompt = ref<BeforeInstallPromptEvent | null>(null)
  // 是否已安装
  const isInstalled = ref(false)
  // 是否显示安装按钮
  const showInstallButton = ref(false)

  /**
   * 处理 beforeinstallprompt 事件
   */
  function handleBeforeInstallPrompt(e: Event) {
    // 阻止默认的安装提示
    e.preventDefault()
    // 保存事件以便后续使用
    deferredPrompt.value = e as BeforeInstallPromptEvent
    // 显示安装按钮
    showInstallButton.value = true
  }

  /**
   * 处理 appinstalled 事件（已安装）
   */
  function handleAppInstalled() {
    isInstalled.value = true
    showInstallButton.value = false
    deferredPrompt.value = null
  }

  /**
   * 检测是否已安装
   */
  function checkInstalled() {
    // 检查是否在独立模式下运行（已安装）
    if (window.matchMedia('(display-mode: standalone)').matches) {
      isInstalled.value = true
      showInstallButton.value = false
      return
    }

    // 检查是否在 iOS 上已添加到主屏幕
    if ((window.navigator as any).standalone === true) {
      isInstalled.value = true
      showInstallButton.value = false
      return
    }
  }

  /**
   * 安装应用
   */
  async function install(): Promise<boolean> {
    if (!deferredPrompt.value) {
      return false
    }

    try {
      // 显示安装提示
      await deferredPrompt.value.prompt()
      // 等待用户响应
      const { outcome } = await deferredPrompt.value.userChoice
      
      if (outcome === 'accepted') {
        // 用户接受了安装
        isInstalled.value = true
        showInstallButton.value = false
        deferredPrompt.value = null
        return true
      } else {
        // 用户拒绝了安装
        return false
      }
    } catch (error) {
      console.error('安装失败:', error)
      return false
    }
  }

  // 组件挂载时监听事件
  onMounted(() => {
    window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    window.addEventListener('appinstalled', handleAppInstalled)
    checkInstalled()
  })

  // 组件卸载时移除监听
  onUnmounted(() => {
    window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt)
    window.removeEventListener('appinstalled', handleAppInstalled)
  })

  return {
    showInstallButton,
    isInstalled,
    install
  }
}

