/**
 * MultiSelectWidget 初始化器
 * 
 * 🔥 组件自治：MultiSelectWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 检查是否需要初始化（是否有 OnSelectFuzzy 回调）
 * - 如果只有 raw 值（来自 URL），通过 by_values 查询获取 display 和 meta
 * - 如果已经有完整的 display 和 meta（来自快链），则不需要初始化
 */

import type { IWidgetInitializer, WidgetInitContext } from '../interfaces/IWidgetInitializer'
import type { FieldValue } from '../../types/field'
import { selectFuzzy } from '@/api/function'
import { SelectFuzzyQueryType } from '../../constants/select'
import { createFieldValue } from '../utils/createFieldValue'
import { Logger } from '../../utils/logger'

/**
 * MultiSelectWidget 初始化器
 * 
 * 🔥 组件自治：MultiSelectWidget 自己负责自己的初始化逻辑
 */
export class MultiSelectWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 MultiSelectWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail, allFormData } = context
    
    // 1. 检查是否需要初始化
    // 如果字段没有 OnSelectFuzzy 回调，则不需要初始化
    if (!field.callbacks?.includes('OnSelectFuzzy')) {
      return null  // 不需要初始化
    }
    
    // 2. 如果已经有完整的 display 和 meta（来自快链），则不需要初始化
    if (currentValue.display && currentValue.meta?.displayInfo) {
      Logger.debug('[MultiSelectWidgetInitializer]', '已有完整的 display 和 meta，跳过初始化', {
        fieldCode: field.code,
        display: currentValue.display,
        hasDisplayInfo: !!currentValue.meta?.displayInfo
      })
      return null  // 不需要初始化
    }
    
    // 3. 如果只有 raw 值（来自 URL），需要通过 by_values 查询获取 display 和 meta
    if (currentValue.raw !== null && currentValue.raw !== undefined) {
      try {
        // 确保 raw 是数组
        const rawArray = Array.isArray(currentValue.raw) ? currentValue.raw : [currentValue.raw]
        if (rawArray.length === 0) {
          return null  // 空数组，不需要初始化
        }
        
        // 类型转换：根据 value_type 转换数组元素类型
        const valueType = field.data?.type || '[]string'
        let convertedValue: any = rawArray
        
        if (valueType.startsWith('[]')) {
          const elementType = valueType.slice(2)
          if (elementType === 'int' || elementType === 'integer') {
            convertedValue = rawArray.map((v: any) => {
              const intVal = parseInt(String(v), 10)
              return isNaN(intVal) ? v : intVal
            })
          } else if (elementType === 'float' || elementType === 'number') {
            convertedValue = rawArray.map((v: any) => {
              const floatVal = parseFloat(String(v))
              return isNaN(floatVal) ? v : floatVal
            })
          }
        }
        
        // 构建请求参数（将 allFormData 转换为请求格式）
        const requestData = this.convertFormDataToRequest(allFormData)
        
        // 调用 OnSelectFuzzy 回调接口（使用 by_values）
        const response = await selectFuzzy(
          functionDetail.method || 'POST',
          functionDetail.router || '',
          {
            code: field.code,
            type: SelectFuzzyQueryType.BY_VALUES,
            value: convertedValue,
            request: requestData,
            value_type: valueType
          }
        )
        
        if (response.error_msg) {
          Logger.warn('[MultiSelectWidgetInitializer]', '回调接口返回错误', {
            fieldCode: field.code,
            error: response.error_msg
          })
          return null  // 初始化失败，返回 null
        }
        
        // 构建选项映射（value -> label）
        const optionMap = new Map<any, string>()
        const displayInfoMap = new Map<any, any>()
        
        if (response.items && Array.isArray(response.items)) {
          response.items.forEach((item: any) => {
            optionMap.set(item.value, item.label || String(item.value))
            if (item.display_info || item.displayInfo) {
              displayInfoMap.set(item.value, item.display_info || item.displayInfo)
            }
          })
        }
        
        // 构建 display 字符串（逗号分隔的标签）
        const displayLabels = rawArray.map((raw: any) => {
          return optionMap.get(raw) || String(raw)
        })
        const display = displayLabels.join(', ')
        
        // 构建 displayInfo（数组形式，每个元素对应一个值）
        const displayInfoArray = rawArray.map((raw: any) => {
          return displayInfoMap.get(raw) || null
        })
        
        // 构建初始化后的 FieldValue
        return createFieldValue(
          field,
          currentValue.raw,  // 保持原始 raw 值
          display,
          {
            ...currentValue.meta,
            displayInfo: displayInfoArray.length > 0 ? displayInfoArray : undefined,
            statistics: response.statistics || {}
          }
        )
      } catch (error: any) {
        Logger.error('[MultiSelectWidgetInitializer]', '初始化失败', {
          fieldCode: field.code,
          error: error?.message || error
        })
        return null  // 初始化失败，返回 null
      }
    }
    
    // 4. 没有 raw 值，不需要初始化
    return null
  }
  
  /**
   * 将表单数据转换为请求格式
   * 
   * @param formData 表单数据（FieldValue 格式）
   * @returns 请求数据（raw 值格式）
   */
  private convertFormDataToRequest(formData: Record<string, FieldValue>): Record<string, any> {
    const request: Record<string, any> = {}
    Object.keys(formData).forEach(key => {
      request[key] = formData[key].raw
    })
    return request
  }
}

