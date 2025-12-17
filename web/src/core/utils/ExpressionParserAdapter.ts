/**
 * ExpressionParserAdapter - 表达式解析器适配器
 * 
 * 功能：
 * 1. 自动检测表达式语法（新语法 vs 旧语法）
 * 2. 选择合适的解析器（ExpressionParserV2 vs ExpressionParser）
 * 3. 统一接口，对上层透明
 * 
 * 使用方式：
 * - 将现有的 ExpressionParser.evaluate 替换为 ExpressionParserAdapter.evaluate
 * - 自动支持新旧两种语法，无需修改业务代码
 */

import { ExpressionParser } from './ExpressionParser'
import { ExpressionParserV2 } from './ExpressionParserV2'
import { Logger } from './logger'

export class ExpressionParserAdapter {
  /**
   * 检测表达式是否使用新语法
   * 
   * 新语法特征：
   * 1. 包含自然数学操作符：字段名后跟 *、+、-、/（不在函数参数中）
   * 2. 包含 IF ... THEN ... ELSE
   * 3. 包含 COALESCE 函数
   * 
   * 旧语法特征：
   * 1. 逗号分隔：sum(价格,*quantity)
   * 2. * 作为前缀：,*quantity, *0.9
   */
  private static isNewSyntax(expression: string): boolean {
    if (!expression) {
      return false
    }

    // 提取函数参数部分
    const match = expression.match(/^(\w+)\((.*)\)$/)
    if (!match) {
      // 不是函数调用，可能是纯文本，使用旧解析器
      return false
    }

    const [, funcName, argsStr] = match

    // 特殊函数：value() 使用旧解析器（保持兼容）
    if (funcName === 'value') {
      return false
    }

    // 检测新语法特征
    const hasNaturalOperator = /[a-zA-Z\u4e00-\u9fa5_"]\s*[*+\-/]/.test(argsStr)
    const hasIfThenElse = /IF\s+.+\s+THEN\s+.+\s+ELSE/i.test(argsStr)
    const hasCoalesce = /COALESCE\s*\(/i.test(argsStr)
    const hasIfNull = /IFNULL\s*\(/i.test(argsStr)
    const hasCaseWhen = /CASE\s+WHEN/i.test(argsStr)

    // 如果包含新语法特征，使用新解析器
    if (hasNaturalOperator || hasIfThenElse || hasCoalesce || hasIfNull || hasCaseWhen) {
      return true
    }

    // 检测旧语法特征
    const hasOldSyntax = /,\s*\*/.test(argsStr) // 包含 ,* 模式

    // 如果明确是旧语法，使用旧解析器
    if (hasOldSyntax) {
      return false
    }

    // 默认：如果包含数学操作符，使用新解析器；否则使用旧解析器
    // 这样可以支持简单的字段名（如 sum(price)），使用旧解析器
    return hasNaturalOperator
  }

  /**
   * 计算表达式（统一接口）
   * 
   * @param expression 表达式字符串
   * @param data 数据数组
   * @param selectedItem 当前选中项（用于 value() 函数），可选
   * @returns 计算结果
   */
  static evaluate(expression: string, data: any[], selectedItem?: any): any {
    try {
      // 自动检测语法，选择合适的解析器
      if (this.isNewSyntax(expression)) {
        // 使用新解析器
        return ExpressionParserV2.evaluate(expression, data, selectedItem)
      } else {
        // 使用旧解析器（向后兼容）
        return ExpressionParser.evaluate(expression, data, selectedItem)
      }
    } catch (error) {
      Logger.error('ExpressionParserAdapter', `计算表达式失败: ${expression}`, error)
      
      // 如果新解析器失败，尝试使用旧解析器（降级策略）
      if (this.isNewSyntax(expression)) {
        try {
          Logger.warn('ExpressionParserAdapter', `新解析器失败，尝试使用旧解析器: ${expression}`)
          return ExpressionParser.evaluate(expression, data, selectedItem)
        } catch (fallbackError) {
          Logger.error('ExpressionParserAdapter', `旧解析器也失败: ${expression}`, fallbackError)
          return 0
        }
      }
      
      return 0
    }
  }
}

