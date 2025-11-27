/**
 * useAppEnvironment - 应用环境检测
 * 
 * 功能：
 * - 检测是否在 PWA/桌面环境中运行
 * - 检测是否在独立模式下运行
 * - 提供环境相关的工具函数
 */

import { ref, onMounted } from 'vue'

/**
 * 应用环境检测组合式函数
 */
export function useAppEnvironment() {
  // 是否在独立模式下运行（PWA/桌面应用）
  const isStandalone = ref(false)
  // 是否在 PWA 环境中运行
  const isPWA = ref(false)
  // 是否在移动设备上运行
  const isMobile = ref(false)

  /**
   * 检测运行环境
   */
  function detectEnvironment() {
    // 检测是否在独立模式下运行（PWA/桌面应用）
    if (window.matchMedia('(display-mode: standalone)').matches) {
      isStandalone.value = true
      isPWA.value = true
    }
    
    // 检测是否在 iOS 上已添加到主屏幕
    if ((window.navigator as any).standalone === true) {
      isStandalone.value = true
      isPWA.value = true
    }
    
    // 检测是否在移动设备上
    const userAgent = navigator.userAgent || navigator.vendor || (window as any).opera
    const isMobileDevice = /android|webos|iphone|ipad|ipod|blackberry|iemobile|opera mini/i.test(userAgent.toLowerCase())
    isMobile.value = isMobileDevice
    
    // 检测是否在 Electron 或其他桌面环境中运行
    if ((window as any).electron || (window as any).__TAURI__) {
      isStandalone.value = true
    }
  }

  /**
   * 判断是否应该在当前窗口打开链接
   * 
   * 在 PWA/桌面环境中，新窗口打开会跳转到浏览器，所以应该使用路由导航
   * 在浏览器环境中，可以正常使用新窗口打开
   */
  function shouldOpenInCurrentWindow(target?: string): boolean {
    // 如果明确指定在当前窗口打开，直接返回 true
    if (target === '_self' || !target) {
      return true
    }
    
    // 如果指定在新窗口打开，但在 PWA 环境中，应该在当前窗口打开
    if (target === '_blank' && isStandalone.value) {
      return true
    }
    
    // 其他情况按原样处理
    return target === '_self' || !target
  }

  // 组件挂载时检测环境
  onMounted(() => {
    detectEnvironment()
  })

  return {
    isStandalone,
    isPWA,
    isMobile,
    shouldOpenInCurrentWindow,
    detectEnvironment
  }
}

