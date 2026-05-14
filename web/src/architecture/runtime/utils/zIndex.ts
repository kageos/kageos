/**
 * 全局浮层层级约定。
 *
 * - mini 工作台固定在普通页面浮层之上。
 * - teleported 下拉/菜单必须高于 mini，但低于全局 dialog。
 * - dialog 内部的 select/date-picker popper 必须高于 dialog 本体。
 * - 极少数全屏预览/企业激活弹窗放在最高层。
 */
export const Z_INDEX = {
  miniWorkstation: 2500,
  imagePreview: 3000,
  floatingPopper: 10010,
  scheduledAgentDialogMask: 10019,
  scheduledAgentDialog: 10020,
  scheduledAgentDialogPopper: 10040,
  criticalPreview: 10060,
  enterpriseDialog: 12000
} as const
