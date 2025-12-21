/**
 * SelectWidget 初始化器
 * 
 * 🔥 组件自治：SelectWidget 自己负责自己的初始化逻辑
 * 
 * 功能：
 * - 检查是否需要初始化（是否有 OnSelectFuzzy 回调）
 * - 如果只有 raw 值（来自 URL），通过 by_value 查询获取 display 和 meta
 * - 如果已经有完整的 display 和 meta（来自快链），则不需要初始化
 */

import type { IWidgetInitializer, WidgetInitContext } from '../interfaces/IWidgetInitializer'
import type { FieldValue } from '../../types/field'
import { selectFuzzy } from '@/api/function'
import { SelectFuzzyQueryType } from '../../constants/select'
import { convertValueToType } from '../utils/valueConverter'
import { createFieldValue } from '../utils/createFieldValue'
import { Logger } from '../../utils/logger'

/**
 * SelectWidget 初始化器
 * 
 * 🔥 组件自治：SelectWidget 自己负责自己的初始化逻辑
 */
export class SelectWidgetInitializer implements IWidgetInitializer {
  /**
   * 初始化 SelectWidget
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue | null> {
    const { field, currentValue, functionDetail, allFormData } = context
    
    console.log(`🔍 [SelectWidgetInitializer] 开始初始化字段 ${field.code}`, {
      hasCallback: field.callbacks?.includes('OnSelectFuzzy'),
      currentValue: {
        raw: currentValue.raw,
        display: currentValue.display,
        hasDisplayInfo: !!currentValue.meta?.displayInfo
      },
      initSource: context.initSource
    })
    
    // 1. 检查是否需要初始化
    // 如果字段没有 OnSelectFuzzy 回调，则不需要初始化
    if (!field.callbacks?.includes('OnSelectFuzzy')) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 没有 OnSelectFuzzy 回调，跳过初始化`)
      return null  // 不需要初始化
    }
    
    // 2. 如果已经有完整的 display 和 meta（来自快链），则不需要初始化
    if (currentValue.display && currentValue.meta?.displayInfo) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 已有完整的 display 和 meta，跳过初始化`, {
        display: currentValue.display,
        hasDisplayInfo: !!currentValue.meta?.displayInfo
      })
      return null  // 不需要初始化
    }
    
    // 3. 如果只有 raw 值（来自 URL），需要通过 by_value 查询获取 display 和 meta
    if (currentValue.raw !== null && currentValue.raw !== undefined) {
      console.log(`🔍 [SelectWidgetInitializer] 字段 ${field.code} 只有 raw 值，需要通过 by_value 查询`, {
        rawValue: currentValue.raw
      })
      try {
        const valueType = field.data?.type || 'string'
        let convertedValue: any = currentValue.raw
        
        // 类型转换：如果 raw 是字符串但类型不是 string，需要转换
        if (typeof currentValue.raw === 'string' && valueType !== 'string') {
          convertedValue = convertValueToType(currentValue.raw, valueType, 'SelectWidgetInitializer')
        }
        
        // 构建请求参数（将 allFormData 转换为请求格式）
        const requestData = this.convertFormDataToRequest(allFormData)
        
        // 调用 OnSelectFuzzy 回调接口
        console.log(`🔍 [SelectWidgetInitializer] 调用 OnSelectFuzzy 回调接口`, {
          fieldCode: field.code,
          method: functionDetail.method || 'GET',
          router: functionDetail.router || '',
          convertedValue,
          valueType
        })
        
        const response = await selectFuzzy(
          functionDetail.method || 'GET',
          functionDetail.router || '',
          {
            code: field.code,
            type: SelectFuzzyQueryType.BY_VALUE,
            value: convertedValue,
            request: requestData,
            value_type: valueType
          }
        )
        
        console.log(`🔍 [SelectWidgetInitializer] OnSelectFuzzy 回调接口返回`, {
          fieldCode: field.code,
          hasError: !!response.error_msg,
          itemsCount: response.items?.length || 0
        })
        
        if (response.error_msg) {
          console.warn(`⚠️ [SelectWidgetInitializer] 字段 ${field.code} 回调接口返回错误`, {
            error: response.error_msg
          })
          return null  // 初始化失败，返回 null
        }
        
        // 找到匹配的选项
        if (response.items && Array.isArray(response.items) && response.items.length > 0) {
          const matchedItem = response.items.find((item: any) => {
            // 支持多种类型比较
            return item.value === currentValue.raw || 
                   String(item.value) === String(currentValue.raw)
          })
          
          if (matchedItem) {
            const initializedValue = createFieldValue(
              field,
              currentValue.raw,
              matchedItem.label || String(matchedItem.value),
              {
                ...currentValue.meta,
                displayInfo: matchedItem.display_info || matchedItem.displayInfo,
                statistics: response.statistics || {}
              }
            )
            
            console.log(`✅ [SelectWidgetInitializer] 字段 ${field.code} 初始化成功`, {
              raw: initializedValue.raw,
              display: initializedValue.display,
              hasDisplayInfo: !!initializedValue.meta?.displayInfo
            })
            
            // 构建初始化后的 FieldValue
            return initializedValue
          }
        }
        
        console.warn(`⚠️ [SelectWidgetInitializer] 字段 ${field.code} 未找到匹配的选项`, {
          rawValue: currentValue.raw,
          itemsCount: response.items?.length || 0
        })
        return null  // 未找到匹配的选项，返回 null
      } catch (error: any) {
        Logger.error('[SelectWidgetInitializer]', '初始化失败', {
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

