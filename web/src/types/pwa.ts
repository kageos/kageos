/**
 * PWA 相关类型定义
 */

/**
 * beforeinstallprompt 事件类型
 */
export interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{
    outcome: 'accepted' | 'dismissed'
    platform: string
  }>
}

