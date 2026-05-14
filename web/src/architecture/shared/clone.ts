/**
 * 深拷贝工具函数
 * 
 * 使用 structuredClone() 作为首选方法（性能更好，支持更多类型）
 * 如果浏览器不支持，回退到 JSON.parse(JSON.stringify())
 */

/**
 * 深拷贝对象
 * @param obj 要拷贝的对象
 * @returns 深拷贝后的新对象
 */
export function deepClone<T>(obj: T): T {
  if (obj === null || typeof obj !== 'object') {
    return obj
  }

  // 使用 structuredClone() 如果浏览器支持（现代浏览器都支持）
  if (typeof structuredClone !== 'undefined') {
    try {
      return structuredClone(obj) as T
    } catch (error) {
      // 如果 structuredClone 失败（例如对象包含不可克隆的类型），回退到 JSON 方法
    }
  }

  // 回退到 JSON 方法（兼容旧浏览器）
  try {
    return JSON.parse(JSON.stringify(obj)) as T
  } catch (error) {
    // 如果 JSON 方法也失败，返回原对象（避免程序崩溃）
    console.warn('deepClone failed, returning original object:', error)
    return obj
  }
}

