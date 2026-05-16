/**
 * 全局浮层层级约定。
 *
 * - mini 工作台固定在普通页面浮层之上。
 * - teleported 下拉/菜单必须高于 mini，但低于全局 dialog。
 * - dialog 内部的 select/date-picker popper 必须高于 dialog 本体。
 */
export const Z_INDEX = {
  miniWorkstation: 2500,
  imagePreview: 3000,
  floatingPopper: 10010,
  criticalPreview: 10060
} as const
