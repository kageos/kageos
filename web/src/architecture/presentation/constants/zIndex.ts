/**
 * 全局浮层层级约定。
 *
 * - mini 工作台固定在普通页面浮层之上。
 * - teleported 下拉/菜单必须高于 mini，但低于全局 dialog/drawer。
 * - 全局 dialog/drawer 用于跨页面、跨详情面板的主浮层。
 * - critical preview 用于必须压过普通业务弹窗的预览层。
 */
export const Z_INDEX = {
  miniWorkstation: 2500,
  imagePreview: 3000,
  floatingPopper: 10010,
  globalOverlay: 10040,
  criticalPreview: 10060
} as const
