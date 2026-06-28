export const FORM_LABEL_WIDTH = '150px'
export const FORM_QUESTIONNAIRE_TRIGGER_CHARS = 16

export function getVisualLength(str?: string | null): number {
  if (!str) return 0
  // 中文、全角标点符号等视为 2 个字符宽度
  return str.replace(/[\u4e00-\u9fa5\u3000-\u303f\uff00-\uffef]/g, 'xx').length
}
