/**
 * 统一的日志工具
 * 
 * 优势：
 * 1. 可以根据环境开关日志
 * 2. 统一格式
 * 3. 方便追踪和调试
 */

// 日志级别
enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  NONE = 99
}

class Logger {
  private static level: LogLevel = import.meta.env.DEV ? LogLevel.DEBUG : LogLevel.WARN

  /**
   * 设置日志级别
   */
  static setLevel(level: LogLevel) {
    this.level = level
  }

  /**
   * Debug 日志（开发环境）
   */
  static debug(module: string, message: string, ...data: any[]) {
    if (this.level <= LogLevel.DEBUG) {
      console.log(`[${module}] ${message}`, ...data)
    }
  }

  /**
   * Info 日志
   */
  static info(module: string, message: string, ...data: any[]) {
    if (this.level <= LogLevel.INFO) {
      console.log(`[${module}] ${message}`, ...data)
    }
  }

  /**
   * Warning 日志
   */
  static warn(module: string, message: string, ...data: any[]) {
    if (this.level <= LogLevel.WARN) {
      console.warn(`[${module}] ${message}`, ...data)
    }
  }

  /**
   * Error 日志（总是显示）
   */
  static error(module: string, message: string, ...data: any[]) {
    if (this.level <= LogLevel.ERROR) {
      console.error(`[${module}] ${message}`, ...data)
    }
  }

  /**
   * 性能测量开始
   */
  static time(label: string) {
    if (this.level <= LogLevel.DEBUG) {
      console.time(label)
    }
  }

  /**
   * 性能测量结束
   */
  static timeEnd(label: string) {
    if (this.level <= LogLevel.DEBUG) {
      console.timeEnd(label)
    }
  }
}

// 导出
export { Logger, LogLevel }

