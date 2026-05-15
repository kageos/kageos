/**
 * ExpressionParser - 表达式解析器
 * 用于解析和计算聚合统计表达式
 * 
 * 支持的表达式：
 * 1. 基础聚合：sum(字段), count(字段), avg(字段), min(字段), max(字段)
 * 2. 乘法聚合：sum(字段1,*字段2), sum(字段1,*字段2,*系数)
 * 3. List层聚合：list_sum(字段), list_avg(字段), list_count()
 * 4. 选中项字段值：value(字段) → 显示当前选中项的某个字段值（动态值）
 * 
 * 示例：
 * - sum(价格) → 计算所有行的价格总和
 * - sum(价格,*数量) → 计算所有行的 价格*数量 的总和
 * - sum(价格,*数量,*0.9) → 计算所有行的 价格*数量*0.9 的总和
 * - avg(价格) → 计算所有行的价格平均值
 * - list_sum(用户总价) → 对所有行的"用户总价"字段求和（用于 MultiSelect 二层聚合）
 * - value(余额) → 显示当前选中项的"余额"字段值（从 DisplayInfo 中获取）
 */

import { Logger } from '@/architecture/shared/logger'

type ExpressionRow = Record<string, unknown>
type ExpressionValue = unknown

function isRecord(value: unknown): value is ExpressionRow {
  return typeof value === 'object' && value !== null
}

export class LegacyExpressionParser {
  /**
   * 计算表达式
   * @param expression 表达式字符串，如 "sum(价格)", "sum(价格,*数量)", "selected(价格)"
   * @param data 数据数组，每个元素是一个对象
   * @param selectedItem 当前选中项（用于 selected() 函数），可选
   * @returns 计算结果
   */
  static evaluate(expression: string, data: ExpressionRow[], selectedItem?: unknown): ExpressionValue {
    if (!expression) {
      return ''
    }

    // 解析表达式：函数名(参数1,参数2,...)
    const match = expression.match(/^(\w+)\((.*)\)$/)
    if (!match) {
      // 🔥 不是函数调用，可能是纯文本（如 "9折优惠"），直接返回
      return expression
    }

    const funcName = match[1] ?? ''
    const argsStr = match[2] ?? ''
    
    // 🔥 特殊处理：value() 函数需要从选中项中获取值
    if (funcName === 'value') {
      return this.evaluateValue(argsStr.trim(), selectedItem)
    }
    
    // 对于聚合函数，需要数据数组
    if (!data || data.length === 0) {
      return 0
    }
    
    // 判断是 List 层聚合还是行内聚合
    if (funcName.startsWith('list_')) {
      return this.evaluateListAggregation(funcName.slice(5), argsStr, data)
    } else {
      return this.evaluateRowAggregation(funcName, argsStr, data)
    }
  }

  /**
   * 计算行内聚合（场景 1-3）
   * @param funcName 函数名：sum, count, avg, min, max
   * @param argsStr 参数字符串：字段名 或 字段1,*字段2,*系数
   * @param data 数据数组
   */
  private static evaluateRowAggregation(funcName: string, argsStr: string, data: ExpressionRow[]): number {
    // 解析参数：字段1,*字段2,*0.9
    const args = argsStr.split(',').map(arg => arg.trim())
    
    // 第一个参数是主字段
    const mainField = args[0] ?? ''
    
    // 提取乘法操作符和字段/系数
    const multipliers: Array<{ isField: boolean, value: string }> = []
    for (let i = 1; i < args.length; i++) {
      const arg = args[i]
      if (!arg) {
        continue
      }
      if (arg.startsWith('*')) {
        const value = arg.slice(1)
        // 判断是字段还是数字系数
        const isField = isNaN(Number(value))
        multipliers.push({ isField, value })
      }
    }

    // 根据函数名计算
    switch (funcName.toLowerCase()) {
      case 'sum':
        return this.calculateSum(mainField, multipliers, data)
      
      case 'count':
        return this.calculateCount(mainField, data)
      
      case 'avg':
        return this.calculateAvg(mainField, multipliers, data)
      
      case 'min':
        return this.calculateMin(mainField, data)
      
      case 'max':
        return this.calculateMax(mainField, data)
      
      default:
        Logger.warn('ExpressionParser', `未知函数: ${funcName}`)
        return 0
    }
  }

  /**
   * 计算 List 层聚合（场景 4）
   * @param funcName 函数名：sum, avg, count, min, max
   * @param argsStr 参数字符串：字段名
   * @param data 数据数组
   */
  private static evaluateListAggregation(funcName: string, argsStr: string, data: ExpressionRow[]): number {
    const field = argsStr.trim()
    
    switch (funcName.toLowerCase()) {
      case 'sum':
        return this.calculateListSum(field, data)
      
      case 'avg':
        return this.calculateListAvg(field, data)
      
      case 'count':
        return data.length
      
      case 'min':
        return this.calculateMin(field, data)
      
      case 'max':
        return this.calculateMax(field, data)
      
      default:
        Logger.warn('ExpressionParser', `未知 List 函数: list_${funcName}`)
        return 0
    }
  }

  /**
   * 计算求和
   * @param field 主字段
   * @param multipliers 乘法字段/系数
   * @param data 数据数组
   */
  private static calculateSum(
    field: string,
    multipliers: Array<{ isField: boolean, value: string }>,
    data: ExpressionRow[]
  ): number {
    return data.reduce((sum, row) => {
      const mainValue = this.getFieldValue(row, field)
      if (mainValue === null || mainValue === undefined) return sum
      
      // 计算乘法
      let result = Number(mainValue)
      for (const multiplier of multipliers) {
        if (multiplier.isField) {
          // 字段
          const fieldValue = this.getFieldValue(row, multiplier.value)
          result *= Number(fieldValue || 0)
        } else {
          // 系数
          result *= Number(multiplier.value)
        }
      }
      
      return sum + result
    }, 0)
  }

  /**
   * 计算计数
   */
  private static calculateCount(field: string, data: ExpressionRow[]): number {
    return data.filter(row => {
      const value = this.getFieldValue(row, field)
      return value !== null && value !== undefined && value !== ''
    }).length
  }

  /**
   * 计算平均值
   */
  private static calculateAvg(
    field: string,
    multipliers: Array<{ isField: boolean, value: string }>,
    data: ExpressionRow[]
  ): number {
    const sum = this.calculateSum(field, multipliers, data)
    const count = this.calculateCount(field, data)
    return count > 0 ? sum / count : 0
  }

  /**
   * 计算最小值
   */
  private static calculateMin(field: string, data: ExpressionRow[]): number {
    const values = data
      .map(row => this.getFieldValue(row, field))
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.min(...values) : 0
  }

  /**
   * 计算最大值
   */
  private static calculateMax(field: string, data: ExpressionRow[]): number {
    const values = data
      .map(row => this.getFieldValue(row, field))
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.max(...values) : 0
  }

  /**
   * List 层求和
   */
  private static calculateListSum(field: string, data: ExpressionRow[]): number {
    return data.reduce((sum, row) => {
      const value = this.getFieldValue(row, field)
      return sum + Number(value || 0)
    }, 0)
  }

  /**
   * List 层平均值
   */
  private static calculateListAvg(field: string, data: ExpressionRow[]): number {
    const sum = this.calculateListSum(field, data)
    return data.length > 0 ? sum / data.length : 0
  }

  /**
   * 获取字段值（支持中文字段名、英文字段名）
   */
  private static getFieldValue(row: unknown, fieldName: string): ExpressionValue | null {
    if (!isRecord(row) || !fieldName) {
      return null
    }
    
    // 直接访问字段
    if (Object.prototype.hasOwnProperty.call(row, fieldName)) {
      return row[fieldName]
    }
    
    // 尝试处理嵌套字段（如果未来需要支持）
    // 例如：product.price
    if (fieldName.includes('.')) {
      const parts = fieldName.split('.')
      let value: unknown = row
      for (const part of parts) {
        if (isRecord(value) && Object.prototype.hasOwnProperty.call(value, part)) {
          value = value[part]
        } else {
          return null
        }
      }
      return value
    }
    
    return null
  }

  /**
   * 计算选中项的字段值（value() 函数，动态值）
   * @param fieldName 字段名（来自 DisplayInfo 的 key）
   * @param selectedItem 当前选中项（包含 DisplayInfo）
   * @returns 字段值，如果不存在则返回空字符串
   */
  private static evaluateValue(fieldName: string, selectedItem?: unknown): ExpressionValue {
    if (!isRecord(selectedItem) || !fieldName) {
      return ''
    }
    
    // 🔥 从选中项的 DisplayInfo 中获取字段值
    // selectedItem 可能是：
    // 1. 直接是 DisplayInfo 对象：{ "价格": 100, "库存": 50 }
    // 2. 包含 displayInfo 属性的对象：{ displayInfo: { "价格": 100, "库存": 50 } }
    // 3. 包含 display_info 属性的对象：{ display_info: { "价格": 100, "库存": 50 } }
    
    let displayInfo: ExpressionRow | null = null
    
    if (isRecord(selectedItem.displayInfo)) {
      displayInfo = selectedItem.displayInfo
    } else if (isRecord(selectedItem.display_info)) {
      displayInfo = selectedItem.display_info
    } else if (!Array.isArray(selectedItem)) {
      // 如果 selectedItem 本身就是 DisplayInfo 对象
      displayInfo = selectedItem
    }
    
    if (!displayInfo || typeof displayInfo !== 'object') {
      return ''
    }
    
    // 从 DisplayInfo 中获取字段值
    const value = displayInfo[fieldName]
    
    // 如果值为 null、undefined 或空字符串，返回空字符串
    if (value === null || value === undefined || value === '') {
      return ''
    }
    
    return value
  }

  /**
   * 格式化数字显示
   * @param value 数值
   * @param decimals 小数位数，默认 2
   */
  static formatNumber(value: number, decimals: number = 2): string {
    if (value === null || value === undefined || isNaN(value)) {
      return '0'
    }
    
    return value.toFixed(decimals)
  }
}
