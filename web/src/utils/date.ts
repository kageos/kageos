/**
 * 日期/时间工具函数
 */

/**
 * 格式化时间戳
 * 
 * @param timestamp 时间戳（毫秒或秒）
 * @param format 格式字符串，支持：
 *   - 'YYYY-MM-DD HH:mm:ss' - 完整日期时间
 *   - 'YYYY-MM-DD' - 仅日期
 *   - 其他格式按需扩展
 * @returns 格式化后的字符串
 * 
 * @example
 * formatTimestamp(1640995200000) // '2022-01-01 00:00:00'
 * formatTimestamp(1640995200000, 'YYYY-MM-DD') // '2022-01-01'
 */
export function formatTimestamp(timestamp: number | string | null | undefined, format = 'YYYY-MM-DD HH:mm:ss'): string {
  if (!timestamp) return '-'
  
  // 处理字符串格式的时间戳
  let numTimestamp = typeof timestamp === 'string' ? parseInt(timestamp, 10) : timestamp
  
  // 🔥 根据系统规范，统一使用毫秒级时间戳
  // 但为了兼容性，自动判断时间戳是秒级还是毫秒级
  // 规则：如果时间戳 < 9999999999（约 2001年的秒级时间戳），认为是秒级，需要乘以 1000
  // 否则认为是毫秒级
  // 注意：如果时间戳 > 9999999999，一定是毫秒级，直接使用
  const SECONDS_THRESHOLD = 9999999999  // 2001-09-09 01:46:40 UTC 的秒级时间戳
  
  // 检查是否是秒级时间戳（小于阈值）
  if (numTimestamp > 0 && numTimestamp < SECONDS_THRESHOLD) {
    // 秒级时间戳，转换为毫秒
    numTimestamp = numTimestamp * 1000
  }
  // 否则认为是毫秒级，直接使用（不做任何转换）
  
  const date = new Date(numTimestamp)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) {
    console.warn('[formatTimestamp] 无效的时间戳:', timestamp, '转换后:', numTimestamp)
    return '-'
  }
  
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  // 根据格式返回
  if (format.includes('HH:mm:ss')) {
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }
  return `${year}-${month}-${day}`
}
